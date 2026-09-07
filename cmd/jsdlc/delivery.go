package main

import (
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
	"sort"
	"strings"
	"syscall"
	"time"
)

var deliveryWorkflowRoles = map[string][]string{
	"feature":        {"delivery-planner", "implementer", "qa-executor", "code-reviewer", "verifier"},
	"bug-fix":        {"debugger", "implementer", "qa-executor", "code-reviewer", "verifier"},
	"bug-diagnosis":  {"debugger", "code-reviewer"},
	"pr-review":      {"code-reviewer", "qa-executor", "verifier"},
	"trivial-change": {"implementer", "verifier"},
	"incident":       {"incident-commander", "debugger", "qa-executor", "verifier"},
}

var deliveryCompletionMeanings = map[string]string{
	"feature":        "The everyday feature workflow contract is satisfied for this frozen repository candidate.",
	"bug-fix":        "The everyday bug-fix workflow contract is satisfied for this frozen repository candidate.",
	"bug-diagnosis":  "The repository-side diagnosis evidence is complete for this frozen candidate; no fix is implied.",
	"pr-review":      "The everyday PR-review contract is satisfied for this frozen repository candidate.",
	"trivial-change": "The everyday trivial-change workflow contract is satisfied for this frozen repository candidate.",
	"incident":       "Repository-side incident workflow evidence is complete; production mitigation or resolution is not implied.",
}

type deliveryCatalog struct {
	SchemaVersion int                         `json:"schemaVersion"`
	Workflows     map[string]deliveryContract `json:"workflows"`
}
type deliveryContract struct {
	Roles                  []string `json:"roles"`
	CompletionMeaning      string   `json:"completionMeaning"`
	ReleaseReadinessEffect string   `json:"releaseReadinessEffect"`
}
type deliveryRun struct {
	SchemaVersion  int      `json:"schemaVersion"`
	ID             string   `json:"id"`
	Workflow       string   `json:"workflow"`
	Risk           string   `json:"risk"`
	State          string   `json:"state"`
	Repository     string   `json:"repository"`
	Candidate      string   `json:"candidate"`
	ContentDigest  string   `json:"contentDigest"`
	ContractDigest string   `json:"contractDigest"`
	RequiredRoles  []string `json:"requiredRoles"`
	EvidenceDigest string   `json:"evidenceDigest,omitempty"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}
type deliveryFinding struct {
	ID             string `json:"id"`
	Severity       string `json:"severity"`
	Location       string `json:"location"`
	Evidence       string `json:"evidence"`
	Recommendation string `json:"recommendation"`
}
type deliveryReportInput struct {
	Disposition string            `json:"disposition"`
	Summary     string            `json:"summary"`
	Evidence    []string          `json:"evidence"`
	CheckIDs    []string          `json:"checkIds"`
	Findings    []deliveryFinding `json:"findings"`
	Limitations []string          `json:"limitations"`
}
type deliveryRoleRecord struct {
	SchemaVersion  int                 `json:"schemaVersion"`
	RunID          string              `json:"runId"`
	Workflow       string              `json:"workflow"`
	Role           string              `json:"role"`
	ProducerMode   string              `json:"producerMode"`
	Repository     string              `json:"repository"`
	Candidate      string              `json:"candidate"`
	ContentDigest  string              `json:"contentDigest"`
	ContractDigest string              `json:"contractDigest"`
	ReportDigest   string              `json:"reportDigest"`
	Report         deliveryReportInput `json:"report"`
	RecordedAt     string              `json:"recordedAt"`
}

func deliveryStart(args []string) (result, error) {
	fs := flag.NewFlagSet("delivery-start", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "frozen candidate label")
	workflow := fs.String("workflow", "", "everyday workflow")
	risk := fs.String("risk", "NORMAL", "LOW, NORMAL, or HIGH")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	validRisk := contains([]string{"LOW", "NORMAL"}, *risk) || *workflow == "incident" && *risk == "HIGH"
	if fs.NArg() != 0 || strings.TrimSpace(*candidate) != *candidate || len(*candidate) < 1 || len(*candidate) > 256 || !validDeliveryWorkflow(*workflow) || !validRisk {
		return nil, errors.New("valid --candidate, --workflow, and --risk are required; positional arguments are forbidden")
	}
	abs, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	contentDigest, err := digestRepository(abs)
	if err != nil {
		return nil, err
	}
	catalog, contractDigest, err := loadDeliveryCatalog(*workflow)
	if err != nil {
		return nil, err
	}
	roles := catalog.Workflows[*workflow].Roles
	id, err := newRunID("delivery-" + key)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	run := deliveryRun{SchemaVersion: 1, ID: id, Workflow: *workflow, Risk: *risk, State: "ACTIVE", Repository: abs, Candidate: *candidate, ContentDigest: contentDigest, ContractDigest: contractDigest, RequiredRoles: append([]string(nil), roles...), CreatedAt: now, UpdatedAt: now}
	dir := filepath.Join(root, key, "delivery")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "active.json")
	unlock, err := acquireLock(path)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if previous, _, readErr := readDeliveryRun(root, key); readErr == nil {
		if previous.State == "ACTIVE" {
			return nil, errors.New("an active everyday delivery run already exists")
		}
		archiveDir := filepath.Join(dir, "archive")
		if err := os.MkdirAll(archiveDir, 0o700); err != nil {
			return nil, err
		}
		if err := os.Rename(path, filepath.Join(archiveDir, previous.ID+".json")); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(readErr) {
		return nil, readErr
	}
	if err := os.MkdirAll(filepath.Join(dir, "runs", id, "reports"), 0o700); err != nil {
		return nil, err
	}
	if err := writeDeliveryJSON(path, run); err != nil {
		return nil, err
	}
	return deliveryResult(run, result{"path": path}), nil
}

func deliveryCheck(args []string) (result, error) {
	fs := flag.NewFlagSet("delivery-check", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "candidate label")
	id := fs.String("id", "", "stable evidence ID")
	domainsText := fs.String("domains", "functional", "comma-separated evidence domains")
	timeout := fs.Duration("timeout", 10*time.Minute, "command timeout, at most 30m")
	authorized := fs.Bool("authorized", false, "confirm the exact fully privileged local command was authorized")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	command := fs.Args()
	if *candidate == "" || !validEvidenceID(*id) || len(command) == 0 || !*authorized || *timeout <= 0 || *timeout > 30*time.Minute {
		return nil, errors.New("--candidate, a safe --id, --authorized, a valid timeout, and a command after -- are required")
	}
	domains, err := parseEvidenceDomains(*domainsText)
	if err != nil {
		return nil, err
	}
	abs, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	statePath := filepath.Join(root, key, "delivery", "active.json")
	run, _, err := readDeliveryRun(root, key)
	if err != nil {
		return nil, err
	}
	if run.State != "ACTIVE" || run.Repository != abs || run.Candidate != *candidate {
		return nil, errors.New("check input does not match an active everyday delivery run")
	}
	if err := validateDeliveryBinding(run, abs); err != nil {
		return nil, err
	}
	evidencePath := filepath.Join(root, key, "delivery", "runs", run.ID, "checks", *id+".json")
	reservationPath := filepath.Join(root, key, "delivery", "check-reservations", run.ID, *id+".json")
	reservation := checkReservation{SchemaVersion: 1, RunID: run.ID, ID: *id, OwnerPID: os.Getpid(), CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if err := createCheckReservation(reservationPath, reservation); err != nil {
		return nil, err
	}
	defer os.Remove(reservationPath)
	if _, err := os.Lstat(evidencePath); err == nil {
		return nil, errors.New("delivery evidence ID already exists")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	started := time.Now().UTC()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir, cmd.Env, cmd.SysProcAttr = abs, workerEnvironment(), &syscall.SysProcAttr{Setpgid: true}
	var stdout, stderr cappedBuffer
	stdout.limit, stderr.limit = 8*1024*1024, 1024*1024
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
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
	after, err := digestRepository(abs)
	if err != nil {
		return nil, err
	}
	outHash, errHash := sha256.Sum256(stdout.Bytes()), sha256.Sum256(stderr.Bytes())
	evidence := checkEvidence{1, *id, run.ID, abs, run.Candidate, run.ContentDigest, domains, append([]string(nil), command...), abs, exitStatus, ctx.Err() == context.DeadlineExceeded, after != run.ContentDigest, "LOCAL_UNATTESTED_FULLY_PRIVILEGED", hex.EncodeToString(outHash[:]), hex.EncodeToString(errHash[:]), started.Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano)}
	unlock, err := acquireLock(statePath)
	if err != nil {
		return nil, err
	}
	defer unlock()
	current, _, err := readDeliveryRun(root, key)
	if err != nil {
		return nil, err
	}
	if current.ID != run.ID || current.State != "ACTIVE" || current.ContentDigest != run.ContentDigest {
		return nil, errors.New("active everyday delivery run changed while the check was executing")
	}
	if err := persistCheckEvidence(evidencePath, evidence); err != nil {
		return nil, err
	}
	passed := exitStatus == 0 && !evidence.TimedOut && !evidence.RepositoryChanged
	return deliveryResult(run, result{"check": evidence, "path": evidencePath, "reportedPass": passed, "execution": "FULLY_PRIVILEGED_LOCAL_COMMAND"}), nil
}

func deliveryRecoverCheck(args []string) (result, error) {
	fs := flag.NewFlagSet("delivery-recover-check", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "candidate label")
	id := fs.String("id", "", "reserved evidence ID")
	authorized := fs.Bool("authorized", false, "confirm recovery of the interrupted reservation")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() != 0 || *candidate == "" || !validEvidenceID(*id) || !*authorized {
		return nil, errors.New("--candidate, a safe --id, and --authorized are required")
	}
	abs, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	statePath := filepath.Join(root, key, "delivery", "active.json")
	unlock, err := acquireLock(statePath)
	if err != nil {
		return nil, err
	}
	defer unlock()
	run, _, err := readDeliveryRun(root, key)
	if err != nil {
		return nil, err
	}
	if run.State != "ACTIVE" || run.Repository != abs || run.Candidate != *candidate {
		return nil, errors.New("recovery input does not match an active everyday delivery run")
	}
	reservationPath := filepath.Join(root, key, "delivery", "check-reservations", run.ID, *id+".json")
	reservation, err := readCheckReservation(reservationPath)
	if err != nil {
		return nil, err
	}
	if reservation.RunID != run.ID || reservation.ID != *id {
		return nil, errors.New("delivery reservation identity mismatch")
	}
	if reservation.ProcessGroupID == 0 {
		return nil, errors.New("reservation was interrupted before command identity became durable; recovery is unsafe and a fresh run is required")
	}
	if processOrGroupExists(reservation.OwnerPID, reservation.ProcessGroupID) {
		return nil, errors.New("check owner or command process group is still alive; recovery refused")
	}
	evidencePath := filepath.Join(root, key, "delivery", "runs", run.ID, "checks", *id+".json")
	if _, err := os.Lstat(evidencePath); err == nil {
		return nil, errors.New("committed delivery evidence already exists; recovery refused")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.Remove(reservationPath); err != nil {
		return nil, err
	}
	return deliveryResult(run, result{"recovered": true, "id": *id, "reservation": reservationPath}), nil
}

func deliveryStatus(args []string) (result, error) {
	repo, candidate, err := deliveryIdentityFlags("delivery-status", args)
	if err != nil {
		return nil, err
	}
	abs, key, root, err := stateLocation(repo)
	if err != nil {
		return nil, err
	}
	run, path, err := readDeliveryRun(root, key)
	if err != nil {
		return nil, err
	}
	if run.Repository != abs {
		return nil, errors.New("repository identity mismatch")
	}
	current, digestErr := digestRepository(abs)
	stale := candidate != "" && candidate != run.Candidate || digestErr != nil || current != run.ContentDigest
	return deliveryResult(run, result{"path": path, "stale": stale}), nil
}

func deliveryRecordPass(args []string) (result, error) {
	fs := flag.NewFlagSet("delivery-record", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "candidate label")
	role := fs.String("role", "", "required role")
	file := fs.String("file", "", "role report JSON")
	mode := fs.String("producer-mode", "SEPARATE_PASS", "SELF_REVIEW or SEPARATE_PASS")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() != 0 || *candidate == "" || *role == "" || *file == "" || !contains([]string{"SELF_REVIEW", "SEPARATE_PASS"}, *mode) {
		return nil, errors.New("--candidate, --role, --file, and a valid --producer-mode are required")
	}
	abs, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	activePath := filepath.Join(root, key, "delivery", "active.json")
	unlock, err := acquireLock(activePath)
	if err != nil {
		return nil, err
	}
	defer unlock()
	run, _, err := readDeliveryRun(root, key)
	if err != nil {
		return nil, err
	}
	if run.State != "ACTIVE" || run.Repository != abs || run.Candidate != *candidate || !contains(run.RequiredRoles, *role) {
		return nil, errors.New("report does not match the active everyday delivery run")
	}
	if err := validateDeliveryBinding(run, abs); err != nil {
		return nil, err
	}
	b, err := readBoundedRegularFile(*file, 256*1024)
	if err != nil {
		return nil, err
	}
	var report deliveryReportInput
	if err := decodeDeliveryJSON(b, &report, "delivery report"); err != nil {
		return nil, err
	}
	if err := validateDeliveryReport(report); err != nil {
		return nil, err
	}
	canonicalReport, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	reportHash := sha256.Sum256(canonicalReport)
	record := deliveryRoleRecord{1, run.ID, run.Workflow, *role, *mode, run.Repository, run.Candidate, run.ContentDigest, run.ContractDigest, hex.EncodeToString(reportHash[:]), report, time.Now().UTC().Format(time.RFC3339Nano)}
	path := filepath.Join(root, key, "delivery", "runs", run.ID, "reports", *role+".json")
	if _, err := os.Lstat(path); err == nil {
		return nil, errors.New("role evidence is append-only and already exists")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := writeDeliveryJSON(path, record); err != nil {
		return nil, err
	}
	return deliveryResult(run, result{"role": *role, "producerMode": *mode, "evidencePath": path, "reportDigest": record.ReportDigest}), nil
}

func deliveryVerify(args []string) (result, error) {
	repo, candidate, err := deliveryIdentityFlags("delivery-verify", args)
	if err != nil {
		return nil, err
	}
	if candidate == "" {
		return nil, errors.New("--candidate is required")
	}
	abs, key, root, err := stateLocation(repo)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, key, "delivery", "active.json")
	unlock, err := acquireLock(path)
	if err != nil {
		return nil, err
	}
	defer unlock()
	run, _, err := readDeliveryRun(root, key)
	if err != nil {
		return nil, err
	}
	if run.Repository != abs || run.Candidate != candidate {
		return nil, errors.New("delivery verification identity or candidate mismatch")
	}
	if run.State == "CANCELLED" {
		return nil, errors.New("cancelled everyday delivery runs cannot be verified")
	}
	if err := validateDeliveryBinding(run, abs); err != nil {
		return deliveryResult(run, result{"outcome": "BLOCKED", "reason": err.Error()}), nil
	}
	missing, recordDigests, checkDigests := []string{}, []string{}, []string{}
	seenChecks := map[string]bool{}
	qaChecks := map[string]bool{}
	outcome, reason := "COMPLETE", "all required role reports are clean and bound to the frozen candidate"
	for _, role := range run.RequiredRoles {
		recordPath := filepath.Join(root, key, "delivery", "runs", run.ID, "reports", role+".json")
		b, readErr := readBoundedRegularFile(recordPath, 512*1024)
		if os.IsNotExist(readErr) {
			missing = append(missing, role)
			continue
		}
		if readErr != nil {
			return deliveryResult(run, result{"outcome": "BLOCKED", "reason": "persisted everyday role evidence is unreadable: " + readErr.Error()}), nil
		}
		var record deliveryRoleRecord
		if err := decodeDeliveryJSON(b, &record, "delivery role evidence"); err != nil {
			return deliveryResult(run, result{"outcome": "BLOCKED", "reason": "persisted everyday role evidence is invalid: " + err.Error()}), nil
		}
		if err := validateDeliveryRecord(record, run, role); err != nil {
			return deliveryResult(run, result{"outcome": "BLOCKED", "reason": err.Error()}), nil
		}
		if contains([]string{"qa-executor", "verifier"}, role) && record.Report.Disposition == "CLEAN" {
			if len(record.Report.CheckIDs) == 0 {
				return deliveryResult(run, result{"outcome": "BLOCKED", "reason": "clean QA and verifier reports require successful delivery check evidence"}), nil
			}
			for _, checkID := range record.Report.CheckIDs {
				checkDigest, err := validateDeliveryCheck(root, key, run, checkID)
				if err != nil {
					return deliveryResult(run, result{"outcome": "BLOCKED", "reason": err.Error()}), nil
				}
				if !seenChecks[checkID] {
					seenChecks[checkID] = true
					checkDigests = append(checkDigests, checkID+":"+checkDigest)
				}
			}
			if role == "qa-executor" {
				for _, id := range record.Report.CheckIDs {
					qaChecks[id] = true
				}
			}
			if role == "verifier" && len(qaChecks) > 0 {
				distinct := false
				for _, id := range record.Report.CheckIDs {
					if !qaChecks[id] {
						distinct = true
					}
				}
				if !distinct {
					return deliveryResult(run, result{"outcome": "BLOCKED", "reason": "verifier must cite at least one successful check not reused from QA"}), nil
				}
			}
		}
		h := sha256.Sum256(b)
		recordDigests = append(recordDigests, hex.EncodeToString(h[:]))
		switch record.Report.Disposition {
		case "BLOCKED":
			outcome, reason = "BLOCKED", "a required role is blocked"
		case "FINDINGS":
			if outcome != "BLOCKED" {
				outcome, reason = "ISSUES", "one or more required roles reported findings"
			}
		case "INCONCLUSIVE":
			if outcome == "COMPLETE" {
				outcome, reason = "INCOMPLETE", "one or more required roles are inconclusive"
			}
		}
	}
	if len(missing) > 0 && outcome != "BLOCKED" {
		outcome, reason = "INCOMPLETE", "required role evidence is missing"
	}
	sort.Strings(recordDigests)
	sort.Strings(checkDigests)
	bundleHasher := sha256.New()
	for _, digest := range append(append([]string(nil), recordDigests...), checkDigests...) {
		_, _ = io.WriteString(bundleHasher, digest+"\n")
	}
	bundleDigest := hex.EncodeToString(bundleHasher.Sum(nil))
	if run.EvidenceDigest != "" && run.EvidenceDigest != bundleDigest {
		return deliveryResult(run, result{"outcome": "BLOCKED", "reason": "finalized everyday delivery evidence changed"}), nil
	}
	state := outcome
	if outcome == "INCOMPLETE" && len(missing) > 0 {
		state = "ACTIVE"
	}
	run.State, run.UpdatedAt = state, time.Now().UTC().Format(time.RFC3339Nano)
	if state != "ACTIVE" && run.EvidenceDigest == "" {
		run.EvidenceDigest = bundleDigest
	}
	if err := writeDeliveryJSON(path, run); err != nil {
		return nil, err
	}
	return deliveryResult(run, result{"outcome": outcome, "reason": reason, "missingRoles": missing, "roleEvidenceDigests": recordDigests, "checkEvidenceDigests": checkDigests, "evidenceDigest": run.EvidenceDigest}), nil
}

func deliveryCancel(args []string) (result, error) {
	repo, candidate, err := deliveryIdentityFlags("delivery-cancel", args)
	if err != nil {
		return nil, err
	}
	abs, key, root, err := stateLocation(repo)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, key, "delivery", "active.json")
	unlock, err := acquireLock(path)
	if err != nil {
		return nil, err
	}
	defer unlock()
	run, _, err := readDeliveryRun(root, key)
	if err != nil {
		return nil, err
	}
	if run.State != "ACTIVE" || run.Repository != abs || candidate == "" || run.Candidate != candidate {
		return nil, errors.New("cancel does not match an active everyday delivery run")
	}
	run.State, run.UpdatedAt = "CANCELLED", time.Now().UTC().Format(time.RFC3339Nano)
	if err := writeDeliveryJSON(path, run); err != nil {
		return nil, err
	}
	return deliveryResult(run, result{"outcome": "CANCELLED"}), nil
}

func loadDeliveryCatalog(workflow string) (deliveryCatalog, string, error) {
	path := filepath.Join(os.Getenv("JSDLC_PLUGIN_ROOT"), "workflows", "everyday-delivery.json")
	b, err := readBoundedRegularFile(path, 128*1024)
	if err != nil {
		return deliveryCatalog{}, "", err
	}
	var catalog deliveryCatalog
	if err := decodeDeliveryJSON(b, &catalog, "everyday workflow catalog"); err != nil {
		return deliveryCatalog{}, "", err
	}
	if catalog.SchemaVersion != 1 || len(catalog.Workflows) != len(deliveryWorkflowRoles) {
		return deliveryCatalog{}, "", errors.New("everyday workflow catalog identity is incompatible")
	}
	for name, roles := range deliveryWorkflowRoles {
		c, ok := catalog.Workflows[name]
		if !ok || !equalStrings(c.Roles, roles) || c.CompletionMeaning != deliveryCompletionMeanings[name] || c.ReleaseReadinessEffect != "NONE" {
			return deliveryCatalog{}, "", fmt.Errorf("everyday workflow %q does not match runtime policy", name)
		}
	}
	roles, ok := deliveryWorkflowRoles[workflow]
	if !ok {
		return deliveryCatalog{}, "", errors.New("unsupported everyday workflow for contract snapshot")
	}
	h := sha256.New()
	_, _ = h.Write(b)
	schemaBytes, err := readBoundedRegularFile(filepath.Join(os.Getenv("JSDLC_PLUGIN_ROOT"), "schemas", "delivery-report-input.schema.json"), 128*1024)
	if err != nil {
		return deliveryCatalog{}, "", err
	}
	_, _ = h.Write(schemaBytes)
	for _, role := range roles {
		roleBytes, readErr := readBoundedRegularFile(filepath.Join(os.Getenv("JSDLC_PLUGIN_ROOT"), "roles", role+".md"), 128*1024)
		if readErr != nil {
			return deliveryCatalog{}, "", readErr
		}
		_, _ = io.WriteString(h, role+"\x00")
		_, _ = h.Write(roleBytes)
	}
	return catalog, hex.EncodeToString(h.Sum(nil)), nil
}

func readDeliveryRun(root, key string) (deliveryRun, string, error) {
	path := filepath.Join(root, key, "delivery", "active.json")
	b, err := readBoundedRegularFile(path, 512*1024)
	if err != nil {
		return deliveryRun{}, path, err
	}
	var run deliveryRun
	if err := decodeDeliveryJSON(b, &run, "delivery state"); err != nil {
		return deliveryRun{}, path, err
	}
	validRisk := contains([]string{"LOW", "NORMAL"}, run.Risk) || run.Workflow == "incident" && run.Risk == "HIGH"
	if run.SchemaVersion != 1 || !validDeliveryWorkflow(run.Workflow) || !validRisk || !contains([]string{"ACTIVE", "COMPLETE", "ISSUES", "INCOMPLETE", "BLOCKED", "CANCELLED"}, run.State) || !validMigrationID(run.ID) || !filepath.IsAbs(run.Repository) || filepath.Clean(run.Repository) != run.Repository || strings.TrimSpace(run.Candidate) != run.Candidate || len(run.Candidate) < 1 || len(run.Candidate) > 256 || !validSHA256(run.ContentDigest) || !validSHA256(run.ContractDigest) || run.EvidenceDigest != "" && !validSHA256(run.EvidenceDigest) || !equalStrings(run.RequiredRoles, deliveryWorkflowRoles[run.Workflow]) || parseRFC3339(run.CreatedAt) != nil || parseRFC3339(run.UpdatedAt) != nil {
		return deliveryRun{}, path, errors.New("persisted everyday delivery state is invalid")
	}
	return run, path, nil
}

func validateDeliveryBinding(run deliveryRun, repo string) error {
	if run.Repository != repo {
		return errors.New("repository identity mismatch")
	}
	digest, err := digestRepository(repo)
	if err != nil {
		return err
	}
	if digest != run.ContentDigest {
		return errors.New("frozen everyday delivery candidate content changed")
	}
	_, contractDigest, err := loadDeliveryCatalog(run.Workflow)
	if err != nil {
		return err
	}
	if contractDigest != run.ContractDigest {
		return errors.New("everyday delivery contract changed after candidate freeze")
	}
	return nil
}

func validateDeliveryReport(report deliveryReportInput) error {
	if !contains([]string{"CLEAN", "FINDINGS", "INCONCLUSIVE", "BLOCKED"}, report.Disposition) || strings.TrimSpace(report.Summary) == "" || len(report.Summary) > 4096 || report.Evidence == nil || report.Findings == nil || report.Limitations == nil || len(report.Evidence) > 100 || len(report.Findings) > 100 || len(report.Limitations) > 100 {
		return errors.New("delivery report envelope is invalid")
	}
	if report.CheckIDs == nil || len(report.CheckIDs) > 100 || report.Disposition == "CLEAN" && (len(report.Evidence) == 0 || len(report.Findings) != 0) || report.Disposition == "FINDINGS" && len(report.Findings) == 0 {
		return errors.New("delivery report disposition contradicts its evidence or findings")
	}
	seen := map[string]bool{}
	for _, id := range report.CheckIDs {
		if !validEvidenceID(id) || seen[id] {
			return errors.New("delivery report contains an invalid or duplicate check ID")
		}
		seen[id] = true
	}
	seen = map[string]bool{}
	for _, f := range report.Findings {
		if !validEvidenceID(f.ID) || seen[f.ID] || !contains([]string{"CRITICAL", "HIGH", "MEDIUM", "LOW"}, f.Severity) || strings.TrimSpace(f.Location) == "" || strings.TrimSpace(f.Evidence) == "" || strings.TrimSpace(f.Recommendation) == "" {
			return errors.New("delivery report contains an invalid or duplicate finding")
		}
		seen[f.ID] = true
	}
	for _, values := range [][]string{report.Evidence, report.Limitations} {
		for _, value := range values {
			if strings.TrimSpace(value) == "" || len(value) > 4096 {
				return errors.New("delivery report contains invalid evidence or limitation text")
			}
		}
	}
	return nil
}

func validateDeliveryCheck(root, key string, run deliveryRun, id string) (string, error) {
	if !validEvidenceID(id) {
		return "", errors.New("delivery report cites an invalid check ID")
	}
	path := filepath.Join(root, key, "delivery", "runs", run.ID, "checks", id+".json")
	b, err := readBoundedRegularFile(path, 256*1024)
	if err != nil {
		return "", fmt.Errorf("load cited delivery check %q: %w", id, err)
	}
	var evidence checkEvidence
	if err := decodeDeliveryJSON(b, &evidence, "delivery check evidence"); err != nil {
		return "", err
	}
	if err := validateCheckEvidence(evidence); err != nil {
		return "", err
	}
	if evidence.ID != id || evidence.RunID != run.ID || evidence.Repository != run.Repository || evidence.Candidate != run.Candidate || evidence.RepositoryDigest != run.ContentDigest || evidence.ExitStatus != 0 || evidence.TimedOut || evidence.RepositoryChanged {
		return "", fmt.Errorf("cited delivery check %q is failed, stale, or mismatched", id)
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func validateDeliveryRecord(record deliveryRoleRecord, run deliveryRun, role string) error {
	if record.SchemaVersion != 1 || record.RunID != run.ID || record.Workflow != run.Workflow || record.Role != role || !contains([]string{"SELF_REVIEW", "SEPARATE_PASS"}, record.ProducerMode) || record.Repository != run.Repository || record.Candidate != run.Candidate || record.ContentDigest != run.ContentDigest || record.ContractDigest != run.ContractDigest || !validSHA256(record.ReportDigest) || parseRFC3339(record.RecordedAt) != nil {
		return errors.New("delivery role evidence binding is invalid")
	}
	if err := validateDeliveryReport(record.Report); err != nil {
		return err
	}
	b, err := json.Marshal(record.Report)
	if err != nil {
		return err
	}
	h := sha256.Sum256(b)
	if record.ReportDigest != hex.EncodeToString(h[:]) {
		return errors.New("delivery role report digest mismatch")
	}
	return nil
}

func deliveryIdentityFlags(name string, args []string) (string, string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "candidate label")
	if err := fs.Parse(args); err != nil {
		return "", "", err
	}
	if fs.NArg() != 0 {
		return "", "", errors.New("positional arguments are forbidden")
	}
	return *repo, *candidate, nil
}
func validDeliveryWorkflow(value string) bool { _, ok := deliveryWorkflowRoles[value]; return ok }
func decodeDeliveryJSON(b []byte, target any, label string) error {
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return errors.New(label + " contains trailing JSON")
	}
	return nil
}
func writeDeliveryJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, append(b, '\n'))
}
func deliveryResult(run deliveryRun, extra result) result {
	out := result{"run": run, "scope": "EVERYDAY_DELIVERY", "assurance": "LOCAL_UNATTESTED", "releaseReadinessEffect": "NONE"}
	for key, value := range extra {
		out[key] = value
	}
	return out
}
