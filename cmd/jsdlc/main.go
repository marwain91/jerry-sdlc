package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const version = "0.1.0-dev"

type result map[string]any

func main() {
	if len(os.Args) < 2 {
		fail(errors.New("usage: jsdlc <doctor|classify|roles|start|status>"))
	}
	var out result
	var err error
	switch os.Args[1] {
	case "doctor":
		out, err = doctor()
	case "classify":
		out, err = classify(os.Args[2:])
	case "roles":
		out, err = roles(os.Args[2:])
	case "start":
		out, err = start(os.Args[2:])
	case "status":
		out, err = status(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fail(err)
	}
}

func fail(err error) {
	_ = json.NewEncoder(os.Stderr).Encode(result{"error": err.Error()})
	os.Exit(1)
}

func stateRoot() (string, error) {
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "jsdlc"), nil
	}
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "jsdlc"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "jsdlc"), nil
}

func doctor() (result, error) {
	root, err := stateRoot()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return result{"version": version, "outcome": "ADVISORY_ONLY", "statePath": root, "reason": err.Error()}, nil
	}
	probe := filepath.Join(root, ".doctor")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return result{"version": version, "outcome": "ADVISORY_ONLY", "statePath": root, "reason": err.Error()}, nil
	}
	_ = os.Remove(probe)
	return result{"version": version, "outcome": "MANAGED_SEPARATE_PASSES", "statePath": root, "platform": runtime.GOOS + "/" + runtime.GOARCH, "independentWorkers": false, "note": "runtime isolation must be confirmed by the Codex adapter"}, nil
}

func classify(args []string) (result, error) {
	fs := flag.NewFlagSet("classify", flag.ContinueOnError)
	req := fs.String("request", "", "user request")
	files := fs.String("files", "", "comma-separated changed paths")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	text := strings.ToLower(*req + " " + *files)
	workflow, risk := "feature", "NORMAL"
	if hasAny(text, "release", "ship", "publish", "tag") {
		workflow, risk = "release-readiness", "HIGH"
	}
	if hasAny(text, "incident", "outage", "production down") {
		workflow, risk = "incident", "HIGH"
	}
	triggers := []string{}
	for _, pair := range [][2]string{{"auth", "security"}, {"migration", "data-migration"}, {".sql", "data-migration"}, {"api", "api-compatibility"}, {"ui", "ux-accessibility"}} {
		if strings.Contains(text, pair[0]) {
			triggers = append(triggers, pair[1])
			risk = "HIGH"
		}
	}
	return result{"workflow": workflow, "risk": risk, "triggers": triggers, "semanticReviewRequired": true}, nil
}

func roles(args []string) (result, error) {
	fs := flag.NewFlagSet("roles", flag.ContinueOnError)
	wf := fs.String("workflow", "release-readiness", "workflow name")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *wf == "release-readiness" {
		return result{"workflow": *wf, "roles": []string{"orchestrator", "qa-architect", "qa-executor", "specialist-reviewer", "independent-verifier"}}, nil
	}
	return result{"workflow": *wf, "roles": []string{"orchestrator", "qa-executor", "independent-verifier"}}, nil
}

type runState struct {
	SchemaVersion int    `json:"schemaVersion"`
	ID            string `json:"id"`
	Workflow      string `json:"workflow"`
	State         string `json:"state"`
	Assurance     string `json:"assurance"`
	Repository    string `json:"repository"`
	Candidate     string `json:"candidate"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

func start(args []string) (result, error) {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	wf := fs.String("workflow", "release-readiness", "workflow")
	candidate := fs.String("candidate", "working-tree", "candidate digest")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(*repo)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256([]byte(abs))
	key := hex.EncodeToString(h[:8])
	now := time.Now().UTC().Format(time.RFC3339)
	id := fmt.Sprintf("%s-%d", key, time.Now().Unix())
	s := runState{1, id, *wf, "BASELINED", "MANAGED_SEPARATE_PASSES", abs, *candidate, now, now}
	root, err := stateRoot()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(root, key)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "active.json"), b, 0o600); err != nil {
		return nil, err
	}
	return result{"run": s, "path": filepath.Join(dir, "active.json")}, nil
}

func status(args []string) (result, error) {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(*repo)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256([]byte(abs))
	key := hex.EncodeToString(h[:8])
	root, err := stateRoot()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(root, key, "active.json"))
	if err != nil {
		return nil, err
	}
	var s runState
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return result{"run": s}, nil
}

func hasAny(s string, terms ...string) bool {
	for _, t := range terms {
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}
