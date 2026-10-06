package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var lightweightDeliveryRoles = map[string][]string{
	"feature":        {"implementer", "code-reviewer"},
	"bug-fix":        {"implementer", "code-reviewer"},
	"trivial-change": {"implementer"},
	"bug-diagnosis":  {"debugger"},
	"pr-review":      {"code-reviewer"},
	"incident":       {"incident-commander", "debugger", "verifier"},
}

func splitDeliveryRoles(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}

func resolveDeliveryProfileRoles(workflow, experience, profile string, extra []string, reason string) ([]string, error) {
	legacy, err := resolveDeliveryRoles(workflow, experience)
	if err != nil {
		return nil, err
	}
	if profile == "" || profile == "full" {
		if len(extra) != 0 || reason != "" {
			return nil, errors.New("additional roles apply only to the lightweight profile")
		}
		return legacy, nil
	}
	if profile != "lightweight" {
		return nil, errors.New("profile must be lightweight or full")
	}
	if len(extra) > 8 || len(reason) > 4096 || len(extra) > 0 && strings.TrimSpace(reason) == "" || len(extra) == 0 && reason != "" {
		return nil, errors.New("additional roles require a specific --role-reason")
	}
	selected := append([]string(nil), lightweightDeliveryRoles[workflow]...)
	for _, role := range extra {
		if !contains([]string{"delivery-planner", "debugger", "qa-executor", "verifier", "code-reviewer", "product-designer", "ux-reviewer"}, role) || contains(selected, role) {
			return nil, fmt.Errorf("invalid or duplicate additional role %q", role)
		}
		selected = append(selected, role)
	}
	return selected, nil
}

func deliveryProfileDigest(workflow, experience, profile string, extra []string, reason string) (string, error) {
	_, legacy, err := loadDeliveryCatalog(workflow, experience)
	if err != nil {
		return "", err
	}
	roles, err := resolveDeliveryProfileRoles(workflow, experience, profile, extra, reason)
	if err != nil {
		return "", err
	}
	if profile != "lightweight" {
		return legacy, nil
	}
	h := sha256.New()
	_, _ = io.WriteString(h, legacy+"\x00lightweight\x00"+reason)
	for _, role := range roles {
		roleBytes, err := readBoundedRegularFile(filepath.Join(os.Getenv("JSDLC_PLUGIN_ROOT"), "roles", role+".md"), 128*1024)
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(h, "\x00"+role)
		_, _ = h.Write(roleBytes)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// A caller-declared scope is local evidence, never a release artifact identity.
// Directories bind membership as well as file contents and modes. Symlinks are
// rejected, including intermediate path components, instead of escaping scope.
func validateDeliveryInputs(paths []string) error {
	if len(paths) > 256 {
		return errors.New("at most 256 dependency paths are supported")
	}
	seen := map[string]bool{}
	for _, path := range paths {
		if path == "" || len(path) > 4096 || filepath.IsAbs(path) || filepath.Clean(path) != path || path == "." || path == ".." || strings.HasPrefix(path, "../") || strings.Contains(path, "\\") || strings.ContainsAny(path, "\x00\n\r") || contains(strings.Split(path, "/"), ".git") || seen[path] {
			return fmt.Errorf("invalid or duplicate repository dependency path %q", path)
		}
		seen[path] = true
	}
	return nil
}

func readDeliveryInputs(file string) ([]string, error) {
	if file == "" {
		return nil, nil
	}
	b, err := readBoundedRegularFile(file, 128*1024)
	if err != nil {
		return nil, err
	}
	var paths []string
	if err := decodeDeliveryJSON(b, &paths, "dependency paths"); err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, errors.New("dependency path file must contain a nonempty JSON array")
	}
	if err := validateDeliveryInputs(paths); err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func digestDeliveryInputs(repo string, paths []string) (string, error) {
	if len(paths) == 0 {
		return digestRepository(repo)
	}
	if err := validateDeliveryInputs(paths); err != nil {
		return "", err
	}
	ordered := append([]string(nil), paths...)
	sort.Strings(ordered)
	h := sha256.New()
	count := 0
	for _, relative := range ordered {
		current := repo
		for _, part := range strings.Split(relative, "/") {
			current = filepath.Join(current, part)
			info, err := os.Lstat(current)
			if os.IsNotExist(err) {
				break
			}
			if err != nil {
				return "", err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return "", errors.New("dependency scope cannot traverse symlinks")
			}
		}
		path := filepath.Join(repo, relative)
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			_, _ = fmt.Fprintf(h, "%s\x00absent\n", relative)
			continue
		} else if err != nil {
			return "", err
		}
		err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			count++
			if count > 100000 {
				return errors.New("dependency scope exceeds 100000 entries")
			}
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() && !info.Mode().IsRegular() {
				return errors.New("dependencies must be regular files or directories without symlinks")
			}
			rel, err := filepath.Rel(repo, p)
			if err != nil {
				return err
			}
			if contains(strings.Split(rel, string(filepath.Separator)), ".git") {
				return errors.New("dependency scope cannot include .git")
			}
			_, _ = fmt.Fprintf(h, "%s\x00%s\x00", rel, info.Mode())
			if info.IsDir() {
				_, _ = io.WriteString(h, "directory\n")
				return nil
			}
			file, err := os.Open(p)
			if err != nil {
				return err
			}
			fileHash := sha256.New()
			_, copyErr := io.Copy(fileHash, file)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			_, _ = fmt.Fprintf(h, "%x\n", fileHash.Sum(nil))
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

type deliveryDependencies struct {
	SchemaVersion int      `json:"schemaVersion"`
	RunID         string   `json:"runId"`
	CheckID       string   `json:"checkId"`
	Inputs        []string `json:"inputs"`
	InputDigest   string   `json:"inputDigest"`
	Environment   string   `json:"environment"`
	CommandDigest string   `json:"commandDigest"`
	SourceRun     string   `json:"sourceRun,omitempty"`
	SourceCheck   string   `json:"sourceCheck,omitempty"`
	SourceDigest  string   `json:"sourceDigest,omitempty"`
}

func deliveryCommandDigest(command []string) string {
	b, _ := json.Marshal(command)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func prepareDeliveryDependencies(repo, file, environment string, command []string) (*deliveryDependencies, error) {
	if file == "" {
		if environment != "" {
			return nil, errors.New("--environment requires --inputs")
		}
		return nil, nil
	}
	if strings.TrimSpace(environment) == "" || len(environment) > 4096 {
		return nil, errors.New("reusable checks require an immutable --environment identity")
	}
	paths, err := readDeliveryInputs(file)
	if err != nil {
		return nil, err
	}
	digest, err := digestDeliveryInputs(repo, paths)
	if err != nil {
		return nil, err
	}
	return &deliveryDependencies{SchemaVersion: 1, Inputs: paths, InputDigest: digest, Environment: environment, CommandDigest: deliveryCommandDigest(command)}, nil
}

func deliveryDependenciesPath(root, key, runID, checkID string) string {
	return filepath.Join(root, key, "delivery", "runs", runID, "dependencies", checkID+".json")
}

func loadDeliveryDependencies(root, key, runID, checkID string) (deliveryDependencies, []byte, error) {
	b, err := readBoundedRegularFile(deliveryDependenciesPath(root, key, runID, checkID), 256*1024)
	if err != nil {
		return deliveryDependencies{}, nil, err
	}
	var d deliveryDependencies
	if err := decodeDeliveryJSON(b, &d, "delivery dependencies"); err != nil {
		return d, nil, err
	}
	if d.SchemaVersion != 1 || d.RunID != runID || d.CheckID != checkID || len(d.Inputs) == 0 || !validSHA256(d.InputDigest) || !validSHA256(d.CommandDigest) || strings.TrimSpace(d.Environment) == "" || len(d.Environment) > 4096 {
		return d, nil, errors.New("invalid delivery dependency record")
	}
	if err := validateDeliveryInputs(d.Inputs); err != nil {
		return d, nil, err
	}
	return d, b, nil
}

func validateDeliveryDependencies(root, key string, run deliveryRun, e checkEvidence) (string, error) {
	d, b, err := loadDeliveryDependencies(root, key, run.ID, e.ID)
	if os.IsNotExist(err) && !contains(run.DependencyChecks, e.ID) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !contains(run.DependencyChecks, e.ID) {
		return "", errors.New("unregistered check dependency record")
	}
	current, err := digestDeliveryInputs(run.Repository, d.Inputs)
	if err != nil || current != d.InputDigest || d.CommandDigest != deliveryCommandDigest(e.Command) {
		return "", errors.New("check dependency or command identity changed")
	}
	if d.SourceRun != "" {
		if !validMigrationID(d.SourceRun) || !validEvidenceID(d.SourceCheck) || !validSHA256(d.SourceDigest) || d.SourceRun == run.ID {
			return "", errors.New("invalid reused-check provenance")
		}
		source, sourceBytes, err := readDeliveryCheckSource(root, key, d.SourceRun, d.SourceCheck)
		if err != nil {
			return "", err
		}
		if err := validateDeliveryCheckSourceRun(root, key, source); err != nil {
			return "", err
		}
		original, originalBytes, err := loadDeliveryDependencies(root, key, d.SourceRun, d.SourceCheck)
		if err != nil || original.SourceRun != "" || original.InputDigest != d.InputDigest || original.Environment != d.Environment || original.CommandDigest != d.CommandDigest || !equalStrings(original.Inputs, d.Inputs) {
			return "", errors.New("reused check has incompatible or missing original dependencies")
		}
		h := sha256.Sum256(append(sourceBytes, originalBytes...))
		if hex.EncodeToString(h[:]) != d.SourceDigest || source.Repository != run.Repository || source.ExitStatus != 0 || source.TimedOut || source.RepositoryChanged || !equalStrings(source.Command, e.Command) || source.StdoutDigest != e.StdoutDigest || source.StderrDigest != e.StderrDigest || source.StartedAt != e.StartedAt || source.CompletedAt != e.CompletedAt || !equalStrings(source.Domains, e.Domains) {
			return "", errors.New("reused check original evidence changed or failed")
		}
	} else if d.SourceCheck != "" || d.SourceDigest != "" {
		return "", errors.New("partial reused-check provenance")
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func readDeliveryCheckSource(root, key, runID, id string) (checkEvidence, []byte, error) {
	b, err := readBoundedRegularFile(filepath.Join(root, key, "delivery", "runs", runID, "checks", id+".json"), 256*1024)
	if err != nil {
		return checkEvidence{}, nil, err
	}
	var e checkEvidence
	if err := decodeDeliveryJSON(b, &e, "original delivery check"); err != nil {
		return e, nil, err
	}
	if err := validateCheckEvidence(e); err != nil {
		return e, nil, err
	}
	if e.RunID != runID || e.ID != id {
		return e, nil, errors.New("original check identity mismatch")
	}
	return e, b, nil
}

func validateDeliveryCheckSourceRun(root, key string, e checkEvidence) error {
	path := filepath.Join(root, key, "delivery", "archive", e.RunID+".json")
	b, err := readBoundedRegularFile(path, 512*1024)
	if os.IsNotExist(err) {
		b, err = readBoundedRegularFile(filepath.Join(root, key, "delivery", "active.json"), 512*1024)
	}
	if err != nil {
		return err
	}
	var source deliveryRun
	if err := decodeDeliveryJSON(b, &source, "original delivery run"); err != nil {
		return err
	}
	roles, err := resolveDeliveryProfileRoles(source.Workflow, source.Experience, source.Profile, source.AdditionalRoles, source.RoleReason)
	if err != nil {
		return err
	}
	contract, err := deliveryProfileDigest(source.Workflow, source.Experience, source.Profile, source.AdditionalRoles, source.RoleReason)
	if err != nil {
		return err
	}
	if source.SchemaVersion != 1 || source.ID != e.RunID || source.Repository != e.Repository || source.Candidate != e.Candidate || source.ContentDigest != e.RepositoryDigest || source.ContractDigest != contract || !equalStrings(source.RequiredRoles, roles) || !contains(source.DependencyChecks, e.ID) {
		return errors.New("original check is not bound to its recorded candidate and contract")
	}
	return ensureNoCheckReservations(root, filepath.Join(key, "delivery"), source.ID)
}

func deliveryReuseCheck(args []string) (result, error) {
	fs := flag.NewFlagSet("delivery-reuse-check", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "target candidate")
	id := fs.String("id", "", "new check ID")
	sourceRun := fs.String("source-run", "", "original run ID")
	sourceID := fs.String("source-check", "", "original executed check ID")
	environment := fs.String("environment", "", "same immutable runtime identity")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() != 0 || *candidate == "" || !validEvidenceID(*id) || !validMigrationID(*sourceRun) || !validEvidenceID(*sourceID) {
		return nil, errors.New("target identity and safe source run/check IDs are required")
	}
	abs, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	unlock, err := acquireLock(filepath.Join(root, key, "delivery", "active.json"))
	if err != nil {
		return nil, err
	}
	defer unlock()
	run, _, err := readDeliveryRun(root, key)
	if err != nil {
		return nil, err
	}
	if run.State != "ACTIVE" || run.Profile != "lightweight" || run.Candidate != *candidate || run.Repository != abs || run.ID == *sourceRun {
		return nil, errors.New("reuse requires a fresh active lightweight candidate")
	}
	if err := validateDeliveryBinding(run, abs); err != nil {
		return nil, err
	}
	source, b, err := readDeliveryCheckSource(root, key, *sourceRun, *sourceID)
	if err != nil {
		return nil, err
	}
	if err := validateDeliveryCheckSourceRun(root, key, source); err != nil {
		return nil, err
	}
	d, db, err := loadDeliveryDependencies(root, key, *sourceRun, *sourceID)
	if err != nil {
		return nil, err
	}
	if d.SourceRun != "" || d.Environment != *environment || source.Repository != abs || source.ExitStatus != 0 || source.TimedOut || source.RepositoryChanged {
		return nil, errors.New("only successful executed checks with matching runtime identity can be reused")
	}
	digest, err := digestDeliveryInputs(abs, d.Inputs)
	if err != nil || digest != d.InputDigest || d.CommandDigest != deliveryCommandDigest(source.Command) {
		return nil, errors.New("original check dependencies changed; execute the affected check")
	}
	path := filepath.Join(root, key, "delivery", "runs", run.ID, "checks", *id+".json")
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		return nil, errors.New("check ID exists or cannot be inspected")
	}
	if err := ensureNoCheckReservations(root, filepath.Join(key, "delivery"), run.ID); err != nil {
		return nil, err
	}
	h := sha256.Sum256(append(b, db...))
	d.RunID, d.CheckID, d.SourceRun, d.SourceCheck, d.SourceDigest = run.ID, *id, *sourceRun, *sourceID, hex.EncodeToString(h[:])
	source.RunID, source.ID, source.Candidate, source.RepositoryDigest = run.ID, *id, run.Candidate, run.ContentDigest
	if err := writeDeliveryJSON(deliveryDependenciesPath(root, key, run.ID, *id), d); err != nil {
		return nil, err
	}
	run.DependencyChecks = append(run.DependencyChecks, *id)
	if err := writeDeliveryJSON(filepath.Join(root, key, "delivery", "active.json"), run); err != nil {
		return nil, err
	}
	if err := persistCheckEvidence(path, source); err != nil {
		return nil, err
	}
	return deliveryResult(run, result{"path": path, "check": source, "reportedPass": true, "execution": "REUSED_LOCAL_CHECK", "dependencyCoverage": "CALLER_DECLARED", "environmentIdentity": "CALLER_DECLARED"}), nil
}
