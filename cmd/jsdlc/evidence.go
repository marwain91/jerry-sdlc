package main

import (
	"bytes"
	"context"
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
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type checkEvidence struct {
	SchemaVersion     int      `json:"schemaVersion"`
	ID                string   `json:"id"`
	RunID             string   `json:"runId"`
	Repository        string   `json:"repository"`
	Candidate         string   `json:"candidateLabel"`
	RepositoryDigest  string   `json:"repositoryDigest"`
	Domains           []string `json:"domains"`
	Command           []string `json:"command"`
	WorkingDirectory  string   `json:"workingDirectory"`
	ExitStatus        int      `json:"exitStatus"`
	TimedOut          bool     `json:"timedOut"`
	RepositoryChanged bool     `json:"repositoryChanged"`
	Trust             string   `json:"trust"`
	StdoutDigest      string   `json:"stdoutDigest"`
	StderrDigest      string   `json:"stderrDigest"`
	StartedAt         string   `json:"startedAt"`
	CompletedAt       string   `json:"completedAt"`
}

type checkReservation struct {
	SchemaVersion  int    `json:"schemaVersion"`
	RunID          string `json:"runId"`
	ID             string `json:"id"`
	OwnerPID       int    `json:"ownerPid"`
	ProcessGroupID int    `json:"processGroupId"`
	CreatedAt      string `json:"createdAt"`
}

var checkAfterStartHook func()

func check(args []string) (result, error) {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "active candidate label")
	id := fs.String("id", "", "stable evidence ID")
	domainsText := fs.String("domains", "", "comma-separated release-risk domains")
	timeout := fs.Duration("timeout", 10*time.Minute, "command timeout, at most 30m")
	authorized := fs.Bool("authorized", false, "confirm the exact fully privileged local command was authorized")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	command := fs.Args()
	if *candidate == "" || !validEvidenceID(*id) || len(command) == 0 || !*authorized {
		return nil, errors.New("--candidate, a safe --id, --authorized, and a command after -- are required")
	}
	if *timeout <= 0 || *timeout > 30*time.Minute {
		return nil, errors.New("--timeout must be positive and at most 30m")
	}
	domains, err := parseEvidenceDomains(*domainsText)
	if err != nil {
		return nil, err
	}
	abs, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	state, _, err := readState(root, key)
	if err != nil {
		return nil, err
	}
	if state.Repository != abs || state.Candidate != *candidate || isTerminalState(state.State) {
		return nil, errors.New("check input does not match an active run")
	}
	before, err := digestRepository(abs)
	if err != nil || before != state.ContentDigest {
		return nil, errors.New("candidate content changed since start; start a fresh run")
	}
	evidencePath := filepath.Join(root, key, "evidence", state.ID, *id+".json")
	if err := os.MkdirAll(filepath.Dir(evidencePath), 0o700); err != nil {
		return nil, err
	}
	reservationPath := filepath.Join(root, key, "check-reservations", state.ID, *id+".json")
	reservation := checkReservation{SchemaVersion: 1, RunID: state.ID, ID: *id, OwnerPID: os.Getpid(), CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if err := createCheckReservation(reservationPath, reservation); err != nil {
		return nil, err
	}
	defer os.Remove(reservationPath)
	if _, err := os.Lstat(evidencePath); err == nil {
		return nil, errors.New("evidence ID already exists")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	started := time.Now().UTC()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = abs
	cmd.Env = workerEnvironment()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stdout, stderr cappedBuffer
	stdout.limit, stderr.limit = 8*1024*1024, 1024*1024
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	if checkAfterStartHook != nil {
		checkAfterStartHook()
	}
	reservation.ProcessGroupID = cmd.Process.Pid
	if err := replaceCheckReservation(reservationPath, reservation); err != nil {
		_ = terminateProcessGroup(cmd.Process.Pid)
		_ = cmd.Wait()
		return nil, err
	}
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	var runErr error
	select {
	case runErr = <-waited:
	case <-ctx.Done():
		_ = terminateProcessGroup(cmd.Process.Pid)
		runErr = <-waited
	}
	if err := terminateProcessGroup(cmd.Process.Pid); err != nil {
		return nil, err
	}
	exitStatus := 0
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitStatus = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			exitStatus = -1
		} else {
			return nil, runErr
		}
	}
	after, digestErr := digestRepository(abs)
	if digestErr != nil {
		return nil, digestErr
	}
	outHash, errHash := sha256.Sum256(stdout.Bytes()), sha256.Sum256(stderr.Bytes())
	evidence := checkEvidence{SchemaVersion: 1, ID: *id, RunID: state.ID, Repository: abs, Candidate: state.Candidate, RepositoryDigest: before, Domains: domains, Command: append([]string{}, command...), WorkingDirectory: abs, ExitStatus: exitStatus, TimedOut: ctx.Err() == context.DeadlineExceeded, RepositoryChanged: after != before, Trust: "LOCAL_UNATTESTED_FULLY_PRIVILEGED", StdoutDigest: hex.EncodeToString(outHash[:]), StderrDigest: hex.EncodeToString(errHash[:]), StartedAt: started.Format(time.RFC3339Nano), CompletedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	statePath := filepath.Join(root, key, "active.json")
	unlock, err := acquireLock(statePath)
	if err != nil {
		return nil, err
	}
	defer unlock()
	current, _, err := readState(root, key)
	if err != nil {
		return nil, err
	}
	if current.ID != state.ID || current.Candidate != state.Candidate || current.ContentDigest != before || isTerminalState(current.State) {
		return nil, errors.New("active run changed while the check was executing")
	}
	if _, err := os.Lstat(evidencePath); err == nil {
		return nil, errors.New("evidence ID already exists")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := persistCheckEvidence(evidencePath, evidence); err != nil {
		return nil, err
	}
	passed := exitStatus == 0 && !evidence.TimedOut && !evidence.RepositoryChanged
	return result{"evidence": evidence, "path": evidencePath, "reportedPass": passed, "readinessEffect": "NONE_UNATTESTED", "execution": "FULLY_PRIVILEGED_LOCAL_COMMAND"}, nil
}

func createCheckReservation(path string, reservation checkReservation) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(reservation)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return errors.New("evidence ID is already reserved or recorded")
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}

func replaceCheckReservation(path string, reservation checkReservation) error {
	b, err := json.Marshal(reservation)
	if err != nil {
		return err
	}
	return writeAtomic(path, b)
}

func terminateProcessGroup(pgid int) error {
	if pgid <= 0 {
		return nil
	}
	err := syscall.Kill(-pgid, syscall.SIGKILL)
	if err != nil && err != syscall.ESRCH {
		return fmt.Errorf("terminate check process group: %w", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for processGroupRunning(pgid) {
		if time.Now().After(deadline) {
			return errors.New("check process group did not become quiescent")
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil
}

func recoverCheck(args []string) (result, error) {
	fs := flag.NewFlagSet("recover-check", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "active candidate label")
	id := fs.String("id", "", "reserved evidence ID")
	authorized := fs.Bool("authorized", false, "confirm recovery of the interrupted reservation")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *candidate == "" || !validEvidenceID(*id) || !*authorized {
		return nil, errors.New("--candidate, a safe --id, and --authorized are required")
	}
	abs, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	statePath := filepath.Join(root, key, "active.json")
	unlock, err := acquireLock(statePath)
	if err != nil {
		return nil, err
	}
	defer unlock()
	state, _, err := readState(root, key)
	if err != nil {
		return nil, err
	}
	if state.Repository != abs || state.Candidate != *candidate || isTerminalState(state.State) {
		return nil, errors.New("recovery input does not match an active run")
	}
	reservationPath := filepath.Join(root, key, "check-reservations", state.ID, *id+".json")
	reservation, err := readCheckReservation(reservationPath)
	if err != nil {
		return nil, err
	}
	if reservation.RunID != state.ID || reservation.ID != *id {
		return nil, errors.New("reservation identity mismatch")
	}
	if reservation.ProcessGroupID == 0 {
		return nil, errors.New("reservation was interrupted before command identity became durable; recovery is unsafe and a fresh run is required")
	}
	if processOrGroupExists(reservation.OwnerPID, reservation.ProcessGroupID) {
		return nil, errors.New("check owner or command process group is still alive; recovery refused")
	}
	evidencePath := filepath.Join(root, key, "evidence", state.ID, *id+".json")
	if _, err := os.Lstat(evidencePath); err == nil {
		return nil, errors.New("committed evidence already exists; recovery refused")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.Remove(reservationPath); err != nil {
		return nil, err
	}
	return result{"recovered": true, "id": *id, "runId": state.ID}, nil
}

func readCheckReservation(path string) (checkReservation, error) {
	b, err := readBoundedRegularFile(path, 4096)
	if err != nil {
		return checkReservation{}, err
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var reservation checkReservation
	if err := dec.Decode(&reservation); err != nil {
		return checkReservation{}, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return checkReservation{}, errors.New("check reservation contains trailing JSON")
	}
	if reservation.SchemaVersion != 1 || reservation.RunID == "" || !validEvidenceID(reservation.ID) || reservation.OwnerPID <= 0 || reservation.ProcessGroupID < 0 || parseRFC3339(reservation.CreatedAt) != nil {
		return checkReservation{}, errors.New("invalid check reservation")
	}
	return reservation, nil
}

func processOrGroupExists(pid, pgid int) bool {
	return processRunning(pid) || processGroupRunning(pgid)
}

func processRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS == "linux" {
		b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if os.IsNotExist(err) {
			return false
		}
		if err == nil {
			closing := strings.LastIndexByte(string(b), ')')
			if closing < 0 {
				return true
			}
			fields := strings.Fields(string(b[closing+1:]))
			return len(fields) > 0 && fields[0] != "Z"
		}
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

func processGroupRunning(pgid int) bool {
	if pgid <= 0 {
		return false
	}
	if runtime.GOOS == "linux" {
		entries, err := os.ReadDir("/proc")
		if err == nil {
			for _, entry := range entries {
				if _, err := strconv.Atoi(entry.Name()); err != nil {
					continue
				}
				b, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
				if err != nil {
					continue
				}
				closing := strings.LastIndexByte(string(b), ')')
				if closing < 0 {
					continue
				}
				fields := strings.Fields(string(b[closing+1:]))
				if len(fields) < 3 {
					continue
				}
				group, err := strconv.Atoi(fields[2])
				if err == nil && group == pgid && fields[0] != "Z" {
					return true
				}
			}
			return false
		}
	}
	err := syscall.Kill(-pgid, 0)
	return err == nil || err == syscall.EPERM
}

func validEvidenceID(value string) bool {
	if len(value) < 1 || len(value) > 80 {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func parseEvidenceDomains(value string) ([]string, error) {
	seen := map[string]bool{}
	var domains []string
	for _, domain := range strings.Split(value, ",") {
		domain = strings.TrimSpace(domain)
		if !contains(requiredReleaseDomains, domain) || seen[domain] {
			return nil, errors.New("--domains must contain unique known release-risk domains")
		}
		seen[domain] = true
		domains = append(domains, domain)
	}
	if len(domains) == 0 {
		return nil, errors.New("--domains is required")
	}
	sort.Strings(domains)
	return domains, nil
}

func persistCheckEvidence(path string, evidence checkEvidence) error {
	if err := validateCheckEvidence(evidence); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, b)
}

func validateCheckEvidence(e checkEvidence) error {
	if e.SchemaVersion != 1 || !validEvidenceID(e.ID) || e.RunID == "" || e.Repository == "" || e.Candidate == "" || !validSHA256(e.RepositoryDigest) || !validSHA256(e.StdoutDigest) || !validSHA256(e.StderrDigest) || len(e.Command) == 0 || e.WorkingDirectory != e.Repository || len(e.Domains) == 0 || e.Trust != "LOCAL_UNATTESTED_FULLY_PRIVILEGED" {
		return errors.New("invalid check evidence identity or digest")
	}
	if parseRFC3339(e.StartedAt) != nil || parseRFC3339(e.CompletedAt) != nil {
		return errors.New("invalid check evidence timestamps")
	}
	seen := map[string]bool{}
	for _, domain := range e.Domains {
		if !contains(requiredReleaseDomains, domain) || seen[domain] {
			return errors.New("invalid or duplicated check evidence domain")
		}
		seen[domain] = true
	}
	return nil
}

func loadCheckEvidence(root, key, runID string) (map[string]checkEvidence, error) {
	dir := filepath.Join(root, key, "evidence", runID)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return map[string]checkEvidence{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(entries) > 256 {
		return nil, errors.New("too many check evidence records")
	}
	result := map[string]checkEvidence{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil, errors.New("evidence directory contains an unexpected entry")
		}
		b, err := readBoundedRegularFile(filepath.Join(dir, entry.Name()), 256*1024)
		if err != nil {
			return nil, err
		}
		dec := json.NewDecoder(strings.NewReader(string(b)))
		dec.DisallowUnknownFields()
		var evidence checkEvidence
		if err := dec.Decode(&evidence); err != nil {
			return nil, err
		}
		var trailing any
		if err := dec.Decode(&trailing); err != io.EOF {
			return nil, errors.New("check evidence contains trailing JSON")
		}
		if err := validateCheckEvidence(evidence); err != nil {
			return nil, err
		}
		if evidence.RunID != runID || entry.Name() != evidence.ID+".json" || result[evidence.ID].ID != "" {
			return nil, errors.New("check evidence path, run, or ID mismatch")
		}
		result[evidence.ID] = evidence
	}
	return result, nil
}

func ensureNoCheckReservations(root, key, runID string) error {
	dir := filepath.Join(root, key, "check-reservations", runID)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return errors.New("one or more checks are running or require explicit recovery")
	}
	return nil
}

func persistTeamEvidence(root, key string, evidence teamEvidence) (string, string, error) {
	if err := validateTeamEvidence(evidence); err != nil {
		return "", "", err
	}
	b, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return "", "", err
	}
	digest := sha256.Sum256(b)
	digestText := hex.EncodeToString(digest[:])
	dir := filepath.Join(root, key, "team-evidence", evidence.RunID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", err
	}
	path := filepath.Join(dir, digestText+".json")
	if _, err := os.Lstat(path); err == nil {
		return "", "", errors.New("team evidence already exists")
	} else if !os.IsNotExist(err) {
		return "", "", err
	}
	if err := writeAtomic(path, b); err != nil {
		return "", "", err
	}
	return path, digestText, nil
}

func canonicalJSONDigest(value []byte) (string, error) {
	var compact bytes.Buffer
	if err := json.Compact(&compact, value); err != nil {
		return "", err
	}
	digest := sha256.Sum256(compact.Bytes())
	return hex.EncodeToString(digest[:]), nil
}

func validateTeamEvidence(e teamEvidence) error {
	if e.SchemaVersion != 1 || e.RunID == "" || e.Repository == "" || e.Candidate == "" || e.Workflow != "release-readiness" || !validSHA256(e.RepositoryDigest) || !validSHA256(e.ContractSetDigest) || !validSHA256(e.ChecksDigest) || !validAssurance(e.Assurance) || !contains([]string{"READY", "NOT_READY", "INCONCLUSIVE"}, e.Verdict) || strings.TrimSpace(e.Reason) == "" || parseRFC3339(e.CompletedAt) != nil || len(e.Roles) != 4 {
		return errors.New("invalid team evidence envelope")
	}
	seenRoles, seenThreads := map[string]bool{}, map[string]bool{}
	for _, role := range e.Roles {
		if !contains([]string{"qa-architect", "qa-executor", "specialist-reviewer", "independent-verifier"}, role.Role) || seenRoles[role.Role] || role.Receipt.SchemaVersion != 1 || role.Receipt.Role != role.Role || role.Receipt.RunID != e.RunID || role.Receipt.Repository != e.Repository || role.Receipt.Candidate != e.Candidate || role.Receipt.RepositoryDigest != e.RepositoryDigest || role.Receipt.ThreadID == "" || seenThreads[role.Receipt.ThreadID] || !validSHA256(role.Receipt.RoleContractDigest) || !validSHA256(role.Receipt.WorkflowDigest) || !validSHA256(role.Receipt.SchemaDigest) || !validSHA256(role.Receipt.PromptDigest) || !validSHA256(role.Receipt.OutputDigest) || !validSHA256(role.Receipt.ReportDigest) || role.Receipt.SandboxModeRequested != "read-only" || role.Receipt.ExitStatus != 0 {
			return errors.New("team role receipt is missing, duplicated, or not bound to the evidence envelope")
		}
		if err := validateWorkerReport(string(role.Report)); err != nil {
			return fmt.Errorf("invalid persisted %s report: %w", role.Role, err)
		}
		reportDigest, err := canonicalJSONDigest(role.Report)
		if err != nil || reportDigest != role.Receipt.ReportDigest {
			return errors.New("persisted report digest does not match its worker receipt")
		}
		seenRoles[role.Role], seenThreads[role.Receipt.ThreadID] = true, true
	}
	return nil
}

func loadLatestTeamEvidence(root, key, runID string) (teamEvidence, string, error) {
	dir := filepath.Join(root, key, "team-evidence", runID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return teamEvidence{}, "", err
	}
	if len(entries) == 0 || len(entries) > 32 {
		return teamEvidence{}, "", errors.New("team evidence count is invalid")
	}
	var latest teamEvidence
	latestPath := ""
	var latestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() || len(entry.Name()) != 69 || filepath.Ext(entry.Name()) != ".json" {
			return teamEvidence{}, "", errors.New("team evidence directory contains an unexpected entry")
		}
		path := filepath.Join(dir, entry.Name())
		b, err := readBoundedRegularFile(path, 2*1024*1024)
		if err != nil {
			return teamEvidence{}, "", err
		}
		digest := sha256.Sum256(b)
		if hex.EncodeToString(digest[:])+".json" != entry.Name() {
			return teamEvidence{}, "", errors.New("team evidence filename digest mismatch")
		}
		dec := json.NewDecoder(strings.NewReader(string(b)))
		dec.DisallowUnknownFields()
		var candidate teamEvidence
		if err := dec.Decode(&candidate); err != nil {
			return teamEvidence{}, "", err
		}
		var trailing any
		if err := dec.Decode(&trailing); err != io.EOF {
			return teamEvidence{}, "", errors.New("team evidence contains trailing JSON")
		}
		if err := validateTeamEvidence(candidate); err != nil {
			return teamEvidence{}, "", err
		}
		if candidate.RunID != runID {
			return teamEvidence{}, "", errors.New("team evidence belongs to another run")
		}
		completedAt, _ := time.Parse(time.RFC3339Nano, candidate.CompletedAt)
		if latestPath == "" || completedAt.After(latestTime) {
			latest, latestPath, latestTime = candidate, path, completedAt
		}
	}
	return latest, latestPath, nil
}

func verifyEvidence(repo, candidate string) (result, error) {
	abs, key, root, err := stateLocation(repo)
	if err != nil {
		return nil, err
	}
	state, _, err := readState(root, key)
	if err != nil {
		return nil, err
	}
	if state.Repository != abs || candidate == "" || state.Candidate != candidate {
		return result{"verdict": "BLOCKED", "reason": "candidate drift or identity mismatch", "run": state}, nil
	}
	currentDigest, err := digestRepository(abs)
	if err != nil || currentDigest != state.ContentDigest {
		return result{"verdict": "BLOCKED", "reason": "candidate content changed since run start", "run": state}, nil
	}
	bundle, path, err := loadLatestTeamEvidence(root, key, state.ID)
	if err != nil {
		return result{"verdict": "INCONCLUSIVE", "reason": "no valid persisted team evidence: " + err.Error(), "run": state}, nil
	}
	if bundle.Repository != abs || bundle.Candidate != state.Candidate || bundle.RepositoryDigest != currentDigest || bundle.Assurance != state.Assurance {
		return result{"verdict": "BLOCKED", "reason": "persisted team evidence is stale or identity-mismatched", "run": state}, nil
	}
	checks, err := loadCheckEvidence(root, key, state.ID)
	if err != nil {
		return result{"verdict": "INCONCLUSIVE", "reason": "checked evidence is invalid: " + err.Error(), "run": state}, nil
	}
	checksBytes, err := json.Marshal(checks)
	if err != nil {
		return nil, err
	}
	checksHash := sha256.Sum256(checksBytes)
	if hex.EncodeToString(checksHash[:]) != bundle.ChecksDigest {
		return result{"verdict": "BLOCKED", "reason": "checked evidence changed after team assessment", "run": state}, nil
	}
	roles := []string{"qa-architect", "qa-executor", "specialist-reviewer", "independent-verifier"}
	contracts, cleanup, err := loadContractSnapshot(root, state.Workflow, roles)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	if contracts.setDigest != bundle.ContractSetDigest {
		return result{"verdict": "BLOCKED", "reason": "workflow or role contracts changed after team assessment", "run": state}, nil
	}
	workflowHash := sha256.Sum256(contracts.workflow)
	for _, role := range bundle.Roles {
		roleHash := sha256.Sum256(contracts.roles[role.Role])
		if role.Receipt.RoleContractDigest != hex.EncodeToString(roleHash[:]) || role.Receipt.WorkflowDigest != hex.EncodeToString(workflowHash[:]) || role.Receipt.SchemaDigest != contracts.schemaHash {
			return result{"verdict": "BLOCKED", "reason": "worker receipt contract digests do not match the frozen contract set", "run": state}, nil
		}
	}
	outputs := make([]result, 0, len(bundle.Roles))
	for _, role := range bundle.Roles {
		outputs = append(outputs, result{"role": role.Role, "report": role.Report})
	}
	verdict, reason := aggregateTeamVerdict(outputs, checks, false)
	if verdict == "READY" && state.Assurance != "MANAGED_INDEPENDENT" {
		verdict, reason = "INCONCLUSIVE", "independent assurance is unavailable; clean separate passes cannot satisfy the readiness independence gate"
	}
	if verdict != bundle.Verdict || reason != bundle.Reason {
		return result{"verdict": "BLOCKED", "reason": "persisted verdict does not reproduce from its evidence", "run": state}, nil
	}
	return result{"verdict": verdict, "reason": reason, "run": state, "evidencePath": path, "reproduced": true}, nil
}
