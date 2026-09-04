package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const version = "0.1.0-dev"

type result map[string]any

func main() {
	if len(os.Args) < 2 {
		fail(errors.New("usage: jsdlc <adapters|packs|doctor|classify|eval-triggers|eval-quality|roles|start|status|transition|upgrade-state|rollback-state|check|recover-check|worker|team|verify>"))
	}
	var out result
	var err error
	switch os.Args[1] {
	case "adapters":
		out, err = adapters(os.Args[2:])
	case "packs":
		out, err = packs(os.Args[2:])
	case "doctor":
		out, err = doctor(os.Args[2:])
	case "classify":
		out, err = classify(os.Args[2:])
	case "eval-triggers":
		out, err = evalTriggers(os.Args[2:])
	case "eval-quality":
		out, err = evalQuality(os.Args[2:])
	case "roles":
		out, err = roles(os.Args[2:])
	case "start":
		out, err = start(os.Args[2:])
	case "status":
		out, err = status(os.Args[2:])
	case "transition":
		out, err = transition(os.Args[2:])
	case "upgrade-state":
		out, err = upgradeState(os.Args[2:])
	case "rollback-state":
		out, err = rollbackState(os.Args[2:])
	case "check":
		out, err = check(os.Args[2:])
	case "recover-check":
		out, err = recoverCheck(os.Args[2:])
	case "worker":
		out, err = worker(os.Args[2:])
	case "team":
		out, err = team(os.Args[2:])
	case "verify":
		out, err = verify(os.Args[2:])
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

func doctor(args []string) (result, error) {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	probeIndependent := fs.Bool("probe-independent", false, "execute distinct read-only Codex worker probes")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
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
	if *probeIndependent {
		observation, probeErr := probeIndependentWorkers(root)
		if probeErr != nil {
			return result{"version": version, "outcome": "MANAGED_SEPARATE_PASSES", "statePath": root, "platform": runtime.GOOS + "/" + runtime.GOARCH, "independentWorkers": false, "readOnlyIsolation": false, "reason": probeErr.Error()}, nil
		}
		return result{"version": version, "outcome": "MANAGED_SEPARATE_PASSES", "statePath": root, "platform": runtime.GOOS + "/" + runtime.GOARCH, "independentWorkers": false, "readOnlyIsolation": false, "capabilityObservation": observation, "reason": "distinct thread IDs were observed; the canary remained absent and the worker reported a blocked write; this is not proof of enforced isolation"}, nil
	}
	return result{"version": version, "outcome": "MANAGED_SEPARATE_PASSES", "statePath": root, "platform": runtime.GOOS + "/" + runtime.GOARCH, "independentWorkers": false, "readOnlyIsolation": false, "reason": "team execution can observe distinct subprocesses, but CLI output cannot attest worker identity; receipts and --probe-independent are non-authoritative"}, nil
}

type workerObservation struct {
	Adapter               string   `json:"adapter"`
	CodexVersion          string   `json:"codexVersion"`
	ThreadIDs             []string `json:"threadIds"`
	DistinctThreads       bool     `json:"distinctThreads"`
	CanaryAbsent          bool     `json:"canaryAbsent"`
	WorkerReportedBlocked bool     `json:"workerReportedBlocked"`
	ObservedAt            string   `json:"observedAt"`
}

func probeIndependentWorkers(root string) (workerObservation, error) {
	bin, err := exec.LookPath("codex")
	if err != nil {
		return workerObservation{}, errors.New("Codex CLI is unavailable for independent workers")
	}
	versionBytes, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return workerObservation{}, fmt.Errorf("inspect Codex CLI: %w", err)
	}
	probeDir, err := os.MkdirTemp(root, ".isolation-probe-")
	if err != nil {
		return workerObservation{}, err
	}
	defer os.RemoveAll(probeDir)
	canary := filepath.Join(probeDir, "isolation-canary")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	first, transcript, err := runCodexProbe(ctx, bin, probeDir, "Use the shell tool to run exactly: touch isolation-canary. Then report whether it succeeded.")
	if err != nil {
		return workerObservation{}, fmt.Errorf("read-only worker probe failed: %w", err)
	}
	if _, err := os.Stat(canary); err == nil || !os.IsNotExist(err) {
		return workerObservation{}, errors.New("read-only worker wrote the isolation canary")
	}
	lowerTranscript := strings.ToLower(transcript)
	if !strings.Contains(lowerTranscript, "isolation-canary") || (!strings.Contains(lowerTranscript, "read-only file system") && !strings.Contains(lowerTranscript, "permission denied")) {
		return workerObservation{}, errors.New("worker did not report that the canary write was blocked")
	}
	second, _, err := runCodexProbe(ctx, bin, probeDir, "Return exactly the word PROBE. Do not use tools.")
	if err != nil {
		return workerObservation{}, fmt.Errorf("distinct worker probe failed: %w", err)
	}
	if first == second {
		return workerObservation{}, errors.New("Codex reused a worker thread during the independence probe")
	}
	return workerObservation{"codex-exec-read-only-v1", strings.TrimSpace(string(versionBytes)), []string{first, second}, true, true, true, time.Now().UTC().Format(time.RFC3339)}, nil
}

func runCodexProbe(ctx context.Context, bin, dir, prompt string) (string, string, error) {
	cmd := exec.CommandContext(ctx, bin, "--ask-for-approval", "never", "exec", "--ephemeral", "--ignore-user-config", "--sandbox", "read-only", "--json", "--skip-git-repo-check", "-C", dir, prompt)
	out, err := cmd.Output()
	if err != nil {
		return "", string(out), err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		var event struct {
			Type     string `json:"type"`
			ThreadID string `json:"thread_id"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) == nil && event.Type == "thread.started" && event.ThreadID != "" {
			return event.ThreadID, string(out), nil
		}
	}
	return "", string(out), errors.New("Codex worker did not return a thread identity")
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
	if releaseIntent(text) {
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

func releaseIntent(text string) bool {
	text = strings.TrimSpace(text)
	nonActionPrefix := []string{"add ", "build ", "compare ", "create ", "deserialize ", "display ", "document ", "explain ", "fix ", "handle ", "investigate ", "list ", "mock ", "parse ", "refactor ", "rename ", "render ", "show ", "store ", "summarize ", "test ", "translate ", "update ", "what ", "why ", "write "}
	nonAction := hasAny(text, "do not release", "don't release", "never release", "cannot ship", "can't ship")
	for _, prefix := range nonActionPrefix {
		if strings.HasPrefix(text, prefix) {
			nonAction = true
			break
		}
	}
	strong := hasAny(text, "release readiness", "ready to ship", "before publishing", "before deploying", "before shipping", "before it goes live", "can go live", "go live today", "preflight check", "launch decision", "final launch qa", "go or no-go review", "go/no-go review", "production rollout", "release candidate", "release decision") || (strings.Contains(text, "prepare") && hasAny(text, "release", "published"))
	if nonAction {
		return false
	}
	if strong {
		return true
	}
	return hasAny(text, "release the ", "ship the ", "publish the ", "tag the ", "publish this ", "publish now")
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
	SchemaVersion         int    `json:"schemaVersion"`
	ID                    string `json:"id"`
	Workflow              string `json:"workflow"`
	State                 string `json:"state"`
	Assurance             string `json:"assurance"`
	Repository            string `json:"repository"`
	Candidate             string `json:"candidate"`
	ContentDigest         string `json:"contentDigest,omitempty"`
	MigrationID           string `json:"migrationId,omitempty"`
	MigratedAt            string `json:"migratedAt,omitempty"`
	MigrationBackupDigest string `json:"migrationBackupDigest,omitempty"`
	CreatedAt             string `json:"createdAt"`
	UpdatedAt             string `json:"updatedAt"`
}

func start(args []string) (result, error) {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	wf := fs.String("workflow", "release-readiness", "workflow")
	candidate := fs.String("candidate", "working-tree", "candidate digest")
	assurance := fs.String("assurance", "MANAGED_SEPARATE_PASSES", "assurance established by doctor")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	abs, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	id, err := newRunID(key)
	if err != nil {
		return nil, err
	}
	if !validAssurance(*assurance) {
		return nil, fmt.Errorf("invalid assurance %q", *assurance)
	}
	if !validWorkflow(*wf) {
		return nil, fmt.Errorf("unsupported workflow %q", *wf)
	}
	if *assurance == "MANAGED_INDEPENDENT" {
		return nil, errors.New("MANAGED_INDEPENDENT requires live verification of actual role-worker receipts, which is not implemented")
	}
	contentDigest, err := digestRepository(abs)
	if err != nil {
		return nil, fmt.Errorf("digest candidate at start: %w", err)
	}
	s := runState{SchemaVersion: 2, ID: id, Workflow: *wf, State: "BASELINED", Assurance: *assurance, Repository: abs, Candidate: *candidate, ContentDigest: contentDigest, CreatedAt: now, UpdatedAt: now}
	dir := filepath.Join(root, key)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "active.json")
	unlock, err := acquireLock(path)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if _, err := os.Stat(path); err == nil {
		previous, _, readErr := readState(root, key)
		if readErr != nil {
			return nil, fmt.Errorf("cannot safely inspect existing run: %w", readErr)
		}
		if !isTerminalState(previous.State) {
			return nil, errors.New("an active run already exists; inspect it with status instead of replacing it")
		}
		if err := archiveState(dir, previous); err != nil {
			return nil, fmt.Errorf("cannot archive terminal run: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := writeState(path, s); err != nil {
		return nil, err
	}
	return result{"run": s, "path": path}, nil
}

func newRunID(key string) (string, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate run ID: %w", err)
	}
	return fmt.Sprintf("%s-%d-%s", key, time.Now().UnixNano(), hex.EncodeToString(random)), nil
}

func status(args []string) (result, error) {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "current candidate digest")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	_, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	s, path, err := readState(root, key)
	if err != nil {
		return nil, err
	}
	unlock, err := acquireLock(path)
	if err != nil {
		return nil, err
	}
	defer unlock()
	s, _, err = readState(root, key)
	if err != nil {
		return nil, err
	}
	stale := *candidate != "" && *candidate != s.Candidate
	return result{"run": s, "path": path, "stale": stale}, nil
}

var allowedTransitions = map[string][]string{
	"BASELINED":          {"STRATEGY_READY", "CANCELLED", "BLOCKED"},
	"STRATEGY_READY":     {"EVIDENCE_COLLECTED", "CANCELLED", "BLOCKED"},
	"EVIDENCE_COLLECTED": {"REVIEWED", "INCONCLUSIVE", "NOT_READY", "CANCELLED", "BLOCKED"},
	"REVIEWED":           {"ADJUDICATED", "INCONCLUSIVE", "NOT_READY", "CANCELLED", "BLOCKED"},
	"ADJUDICATED":        {"VERIFIED", "INCONCLUSIVE", "NOT_READY", "CANCELLED", "BLOCKED"},
	"VERIFIED":           {"NOT_READY", "INCONCLUSIVE"},
}

func transition(args []string) (result, error) {
	fs := flag.NewFlagSet("transition", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	to := fs.String("to", "", "target state")
	candidate := fs.String("candidate", "", "current candidate digest")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *to == "" || *candidate == "" {
		return nil, errors.New("--to and --candidate are required")
	}
	abs, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, key, "active.json")
	unlock, err := acquireLock(path)
	if err != nil {
		return nil, err
	}
	defer unlock()
	s, _, err := readState(root, key)
	if err != nil {
		return nil, err
	}
	if s.Repository != abs {
		return nil, errors.New("repository identity mismatch")
	}
	if s.Candidate != *candidate {
		return nil, fmt.Errorf("candidate drift: recorded %q, current %q", s.Candidate, *candidate)
	}
	if !contains(allowedTransitions[s.State], *to) {
		return nil, fmt.Errorf("forbidden transition %s -> %s", s.State, *to)
	}
	s.State = *to
	s.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := writeState(path, s); err != nil {
		return nil, err
	}
	return result{"run": s}, nil
}

func upgradeState(args []string) (result, error) {
	fs := flag.NewFlagSet("upgrade-state", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "active candidate label")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *candidate == "" {
		return nil, errors.New("--candidate is required")
	}
	abs, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, key, "active.json")
	unlock, err := acquireLock(path)
	if err != nil {
		return nil, err
	}
	defer unlock()
	s, _, err := readState(root, key)
	if err != nil {
		return nil, err
	}
	if s.SchemaVersion != 1 {
		return nil, errors.New("only state schema 1 can be upgraded")
	}
	if s.Repository != abs || s.Candidate != *candidate {
		return nil, errors.New("state upgrade identity or candidate mismatch")
	}
	digest, err := digestRepository(abs)
	if err != nil {
		return nil, err
	}
	if s.ContentDigest != "" && s.ContentDigest != digest {
		return nil, errors.New("legacy candidate content changed; start a fresh run instead of upgrading")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	migrationID, err := newRunID("migration")
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(root, key, "migrations")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	backup := filepath.Join(dir, migrationID+".before.json")
	if err := writeAtomic(backup, before); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	backupHash := sha256.Sum256(before)
	s.SchemaVersion, s.ContentDigest, s.MigrationID, s.MigratedAt, s.MigrationBackupDigest, s.UpdatedAt = 2, digest, migrationID, now, hex.EncodeToString(backupHash[:]), now
	if err := writeState(path, s); err != nil {
		return nil, err
	}
	return result{"run": s, "migrationId": migrationID, "backup": backup, "rollbackAvailable": true}, nil
}

func rollbackState(args []string) (result, error) {
	fs := flag.NewFlagSet("rollback-state", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "active candidate label")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *candidate == "" {
		return nil, errors.New("--candidate is required")
	}
	abs, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, key, "active.json")
	unlock, err := acquireLock(path)
	if err != nil {
		return nil, err
	}
	defer unlock()
	s, _, err := readState(root, key)
	if err != nil {
		return nil, err
	}
	if s.SchemaVersion != 2 || s.MigrationID == "" || s.MigratedAt == "" || s.MigrationBackupDigest == "" || s.UpdatedAt != s.MigratedAt {
		return nil, errors.New("state is not an unchanged migrated state; rollback refused")
	}
	if s.Repository != abs || s.Candidate != *candidate {
		return nil, errors.New("state rollback identity or candidate mismatch")
	}
	currentDigest, err := digestRepository(abs)
	if err != nil || currentDigest != s.ContentDigest {
		return nil, errors.New("candidate content changed after migration; rollback refused")
	}
	dir := filepath.Join(root, key, "migrations")
	backup := filepath.Join(dir, s.MigrationID+".before.json")
	before, err := readBoundedRegularFile(backup, 1024*1024)
	if err != nil {
		return nil, err
	}
	backupHash := sha256.Sum256(before)
	if hex.EncodeToString(backupHash[:]) != s.MigrationBackupDigest {
		return nil, errors.New("migration backup digest mismatch")
	}
	legacy, err := decodeRunState(before)
	if err != nil || legacy.SchemaVersion != 1 || legacy.ID != s.ID || legacy.Repository != s.Repository || legacy.Candidate != s.Candidate {
		return nil, errors.New("migration backup is invalid or belongs to another run")
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	after := filepath.Join(dir, s.MigrationID+".rolled-back-from.json")
	if err := writeAtomic(after, current); err != nil {
		return nil, err
	}
	if err := writeAtomic(path, before); err != nil {
		return nil, err
	}
	return result{"run": legacy, "migrationId": s.MigrationID, "restored": backup, "rollbackSnapshot": after}, nil
}

func verify(args []string) (result, error) {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "current candidate digest")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return verifyEvidence(*repo, *candidate)
}

func stateLocation(repo string) (string, string, string, error) {
	abs, err := filepath.Abs(repo)
	if err != nil {
		return "", "", "", err
	}
	canonical, evalErr := filepath.EvalSymlinks(abs)
	if evalErr != nil {
		return "", "", "", fmt.Errorf("cannot canonicalize repository: %w", evalErr)
	}
	abs = canonical
	h := sha256.Sum256([]byte(abs))
	key := hex.EncodeToString(h[:8])
	root, err := stateRoot()
	if err != nil {
		return "", "", "", err
	}
	return abs, key, root, nil
}

func readState(root, key string) (runState, string, error) {
	path := filepath.Join(root, key, "active.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return runState{}, path, err
	}
	s, err := decodeRunState(b)
	if err != nil {
		return runState{}, path, err
	}
	return s, path, nil
}

func decodeRunState(b []byte) (runState, error) {
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var s runState
	if err := dec.Decode(&s); err != nil {
		return runState{}, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return runState{}, errors.New("persisted state contains trailing JSON")
	}
	if s.SchemaVersion != 1 && s.SchemaVersion != 2 {
		return runState{}, fmt.Errorf("unsupported state schema %d", s.SchemaVersion)
	}
	if !validAssurance(s.Assurance) {
		return runState{}, fmt.Errorf("invalid persisted assurance %q", s.Assurance)
	}
	if s.Assurance == "MANAGED_INDEPENDENT" {
		return runState{}, errors.New("persisted MANAGED_INDEPENDENT lacks live-verified role-worker receipts")
	}
	if s.ID == "" || s.Workflow == "" || s.State == "" || s.Repository == "" || s.Candidate == "" {
		return runState{}, errors.New("persisted state is missing required fields")
	}
	if s.ContentDigest != "" && !validSHA256(s.ContentDigest) {
		return runState{}, errors.New("persisted candidate digest is invalid")
	}
	if s.SchemaVersion == 2 && s.ContentDigest == "" {
		return runState{}, errors.New("state schema 2 requires a candidate content digest")
	}
	if s.SchemaVersion == 1 && (s.MigrationID != "" || s.MigratedAt != "" || s.MigrationBackupDigest != "") {
		return runState{}, errors.New("state schema 1 cannot contain migration metadata")
	}
	hasMigration := s.MigrationID != "" || s.MigratedAt != "" || s.MigrationBackupDigest != ""
	if hasMigration && (!validMigrationID(s.MigrationID) || !validSHA256(s.MigrationBackupDigest) || parseRFC3339(s.MigratedAt) != nil) {
		return runState{}, errors.New("persisted migration metadata is invalid or incomplete")
	}
	if !validWorkflow(s.Workflow) {
		return runState{}, fmt.Errorf("unsupported persisted workflow %q", s.Workflow)
	}
	if !validPersistedState(s.State) {
		return runState{}, fmt.Errorf("invalid persisted state %q", s.State)
	}
	return s, nil
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}
func parseRFC3339(value string) error { _, err := time.Parse(time.RFC3339Nano, value); return err }
func validMigrationID(value string) bool {
	if len(value) < 1 || len(value) > 128 || filepath.Base(value) != value {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func validPersistedState(state string) bool {
	if _, ok := allowedTransitions[state]; ok {
		return true
	}
	return isTerminalState(state)
}

func isTerminalState(state string) bool {
	return state == "NOT_READY" || state == "INCONCLUSIVE" || state == "CANCELLED" || state == "BLOCKED"
}

func archiveState(dir string, s runState) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	historyDir := filepath.Join(dir, "history")
	if err := os.MkdirAll(historyDir, 0o700); err != nil {
		return err
	}
	digest := sha256.Sum256(b)
	return writeAtomic(filepath.Join(historyDir, hex.EncodeToString(digest[:])+".json"), b)
}

func writeState(path string, s runState) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		old, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if err := writeAtomic(path+".bak", old); err != nil {
			return err
		}
	}
	return writeAtomic(path, b)
}

var writeAtomicBeforeRenameHook func(string)

func writeAtomic(path string, b []byte) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".jsdlc-*.tmp")
	if err != nil {
		return err
	}
	tmp := tmpFile.Name()
	defer os.Remove(tmp)
	if err := tmpFile.Chmod(0o600); err != nil {
		tmpFile.Close()
		return err
	}
	if _, err := tmpFile.Write(b); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if writeAtomicBeforeRenameHook != nil {
		writeAtomicBeforeRenameHook(path)
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func acquireLock(path string) (func(), error) {
	lock := path + ".lock"
	f, err := os.OpenFile(lock, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("state is locked: %w", err)
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
}
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
func validAssurance(v string) bool {
	return contains([]string{"MANAGED_INDEPENDENT", "MANAGED_SEPARATE_PASSES", "ADVISORY_ONLY"}, v)
}

func validWorkflow(v string) bool {
	return v == "release-readiness"
}

func hasAny(s string, terms ...string) bool {
	for _, t := range terms {
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}
