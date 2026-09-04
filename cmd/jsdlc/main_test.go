package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestStartDoesNotReplaceExistingRun(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := start([]string{"--repo", repo, "--candidate", "candidate-b"}); err == nil {
		t.Fatal("second start must not replace an existing run")
	}
	got, err := status([]string{"--repo", repo})
	if err != nil {
		t.Fatal(err)
	}
	if got["run"].(runState).Candidate != "candidate-a" || got["path"] != started["path"] {
		t.Fatalf("existing run was changed: %#v", got)
	}
}

func TestStartArchivesTerminalRunBeforeRollover(t *testing.T) {
	stateRoot := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateRoot)
	repo := t.TempDir()
	first, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transition([]string{"--repo", repo, "--candidate", "candidate-a", "--to", "CANCELLED"}); err != nil {
		t.Fatal(err)
	}
	got, err := start([]string{"--repo", repo, "--candidate", "candidate-b"})
	if err != nil {
		t.Fatal(err)
	}
	if got["run"].(runState).Candidate != "candidate-b" {
		t.Fatalf("terminal rollover did not create new run: %#v", got)
	}
	if got["run"].(runState).ID == first["run"].(runState).ID {
		t.Fatal("terminal rollover reused the previous run ID")
	}
	_, key, root, err := stateLocation(repo)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(root, key, "history"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("terminal run was not archived: entries=%v err=%v", entries, err)
	}
}

func TestPersistedIndependentAssuranceIsRejected(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	path := started["path"].(string)
	s := started["run"].(runState)
	s.Assurance = "MANAGED_INDEPENDENT"
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := status([]string{"--repo", repo}); err == nil {
		t.Fatal("forged independent assurance must be rejected")
	}
}

func TestPersistedReadyStateIsRejected(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	path := started["path"].(string)
	s := started["run"].(runState)
	s.State = "READY"
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := status([]string{"--repo", repo}); err == nil {
		t.Fatal("forged READY state must be rejected")
	}
}

func TestClassifyRelease(t *testing.T) {
	got, err := classify([]string{"--request", "Prepare this project for release"})
	if err != nil {
		t.Fatal(err)
	}
	if got["workflow"] != "release-readiness" || got["risk"] != "HIGH" {
		t.Fatalf("unexpected classification: %#v", got)
	}
}

func TestClassifyMigration(t *testing.T) {
	got, err := classify([]string{"--request", "implement account export", "--files", "db/042.sql"})
	if err != nil {
		t.Fatal(err)
	}
	if got["risk"] != "HIGH" {
		t.Fatalf("expected HIGH risk: %#v", got)
	}
	triggers := got["triggers"].([]string)
	if len(triggers) == 0 || triggers[0] != "data-migration" {
		t.Fatalf("expected migration trigger: %#v", got)
	}
}

func TestEvalTriggersRejectsDegenerateAndTrailingFixtures(t *testing.T) {
	for name, body := range map[string]string{
		"one-class":          `{"schemaVersion":1,"subjects":["app"],"cases":[{"id":"p","template":"release {project}","release":true}]}`,
		"trailing":           `{"schemaVersion":1,"subjects":["app"],"cases":[{"id":"p","template":"release {project}","release":true},{"id":"n","template":"fix {project}","release":false}]} {}`,
		"duplicate-subject":  `{"schemaVersion":1,"subjects":["App"," app "],"cases":[{"id":"p","template":"release {project}","release":true},{"id":"n","template":"fix {project}","release":false}]}`,
		"duplicate-id":       `{"schemaVersion":1,"subjects":["app"],"cases":[{"id":" P ","template":"release {project}","release":true},{"id":"p","template":"fix {project}","release":false}]}`,
		"duplicate-template": `{"schemaVersion":1,"subjects":["app"],"cases":[{"id":"p","template":" Release  {project} ","release":true},{"id":"n","template":"release {project}","release":false}]}`,
		"blank-subject":      `{"schemaVersion":1,"subjects":["  "],"cases":[{"id":"p","template":"release {project}","release":true},{"id":"n","template":"fix {project}","release":false}]}`,
		"blank-id":           `{"schemaVersion":1,"subjects":["app"],"cases":[{"id":" ","template":"release {project}","release":true},{"id":"n","template":"fix {project}","release":false}]}`,
	} {
		path := filepath.Join(t.TempDir(), name+".json")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := evalTriggers([]string{"--fixture", path}); err == nil {
			t.Fatalf("%s fixture must fail", name)
		}
	}
}

func TestEvalQualityComputesGraduationMetrics(t *testing.T) {
	tasks := make([]qualityTask, 20)
	for i := range tasks {
		baseline := qualityRun{Findings: []adjudicatedFinding{{ID: "medium", Outcome: "TRUE_POSITIVE"}}, WallMilliseconds: 100, Tokens: 100, HumanReviewMinutes: 10}
		jerry := qualityRun{Findings: []adjudicatedFinding{{ID: "high", Outcome: "TRUE_POSITIVE"}, {ID: "medium", Outcome: "TRUE_POSITIVE"}}, WallMilliseconds: 200, Tokens: 200, HumanReviewMinutes: 5}
		tasks[i] = qualityTask{ID: fmt.Sprintf("task-%d", i), Source: fmt.Sprintf("repo-%d", i), Candidate: fmt.Sprintf("commit-%d", i), Known: []knownFinding{{ID: "high", Severity: "HIGH"}, {ID: "medium", Severity: "MEDIUM"}}, Baseline: []qualityRun{baseline, baseline, baseline}, Jerry: []qualityRun{jerry, jerry, jerry}}
	}
	b, err := json.Marshal(qualitySuite{SchemaVersion: 1, TrialsPerArm: 3, MaxJerryTokensPerRun: 500, Tasks: tasks})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "quality.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := evalQuality([]string{"--fixture", path})
	if err != nil {
		t.Fatal(err)
	}
	if got["passed"] != true || got["tasks"] != 20 || got["trialsPerArm"] != 3 || got["totalRunsPerArm"] != 60 {
		t.Fatalf("unexpected quality result: %#v", got)
	}
	tasks[0].Jerry[0].Tokens = 501
	b, err = json.Marshal(qualitySuite{SchemaVersion: 1, TrialsPerArm: 3, MaxJerryTokensPerRun: 500, Tasks: tasks})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = evalQuality([]string{"--fixture", path})
	if err != nil {
		t.Fatal(err)
	}
	if got["passed"] != false || got["hardTokenBudgetExceeded"] != true {
		t.Fatalf("hard budget must fail: %#v", got)
	}
	tasks[0].Jerry = append(tasks[0].Jerry, tasks[0].Jerry[0])
	b, err = json.Marshal(qualitySuite{SchemaVersion: 1, TrialsPerArm: 3, MaxJerryTokensPerRun: 500, Tasks: tasks})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := evalQuality([]string{"--fixture", path}); err == nil {
		t.Fatal("unequal task weighting must be rejected")
	}
}

func TestQualityRunRejectsMetricGaming(t *testing.T) {
	known := map[string]int{"known": 5}
	cases := []qualityRun{
		{Findings: []adjudicatedFinding{{ID: "unknown", Outcome: "TRUE_POSITIVE"}}, WallMilliseconds: 1, Tokens: 1},
		{Findings: []adjudicatedFinding{{ID: "known", Outcome: "FALSE_POSITIVE"}}, WallMilliseconds: 1, Tokens: 1},
		{Findings: []adjudicatedFinding{}, WallMilliseconds: 1, Tokens: 1, UnauthorizedActions: -1},
	}
	for _, run := range cases {
		if _, err := scoreQualityRun(run, known); err == nil {
			t.Fatalf("accepted gameable run: %#v", run)
		}
	}
}

func TestReleaseRoles(t *testing.T) {
	got, err := roles([]string{"--workflow", "release-readiness"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got["roles"].([]string)) != 5 {
		t.Fatalf("unexpected roles: %#v", got)
	}
}

func TestAdapterCatalogFailsClosed(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("JSDLC_PLUGIN_ROOT", filepath.Clean(filepath.Join(workingDir, "..", "..", "plugins", "jerry-sdlc")))
	got, err := adapters(nil)
	if err != nil {
		t.Fatal(err)
	}
	items := got["adapters"].([]adapterDescriptor)
	if len(items) != 3 || got["assuranceRule"] != "DESCRIPTORS_NEVER_ESTABLISH_MANAGED_INDEPENDENT" {
		t.Fatalf("unexpected catalog: %#v", got)
	}
	for _, item := range items {
		if item.Capabilities.AttestedWorkerIdentity || item.Capabilities.WriteIsolationAttested {
			t.Fatalf("gated adapter claims identity: %#v", item)
		}
	}
	packResult, err := packs(nil)
	if err != nil {
		t.Fatal(err)
	}
	if packResult["activationAllowed"] != false || packResult["status"] != "GATED_BY_PHASE_3" {
		t.Fatalf("packs must remain gated: %#v", packResult)
	}
}

func TestWorkerReceiptBindsActiveRun(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("JSDLC_PLUGIN_ROOT", filepath.Clean(filepath.Join(workingDir, "..", "..", "plugins", "jerry-sdlc")))
	binDir := t.TempDir()
	fakeCodex := filepath.Join(binDir, "codex")
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then
  echo "codex-test 1.0"
  exit 0
fi
cat >/dev/null
printf '%s\n' '{"type":"thread.started","thread_id":"thread-worker-a"}' '{"type":"item.completed","item":{"type":"agent_message","text":"{\"disposition\":\"CLEAN\",\"evidence\":[\"checked\"],\"findings\":[],\"limitations\":[],\"domains\":[]}"}}'
`
	if err := os.WriteFile(fakeCodex, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	prompt := filepath.Join(t.TempDir(), "assignment.txt")
	if err := os.WriteFile(prompt, []byte("Review the candidate."), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := worker([]string{"--repo", repo, "--candidate", "candidate-a", "--role", "specialist-reviewer", "--prompt-file", prompt})
	if err != nil {
		t.Fatal(err)
	}
	receipt := got["receipt"].(workerReceipt)
	if receipt.RunID != started["run"].(runState).ID || receipt.Candidate != "candidate-a" || receipt.ThreadID != "thread-worker-a" || len(receipt.SchemaDigest) != 64 || got["assuranceEffect"] != "EVIDENCE_ONLY" {
		t.Fatalf("receipt is not bound correctly: %#v", got)
	}
	if _, err := os.Stat(got["path"].(string)); err != nil {
		t.Fatal(err)
	}
	if got["persistence"] != "DIGESTS_ONLY_REPORT_NOT_STORED" {
		t.Fatalf("unexpected persistence policy: %#v", got)
	}
}

func TestWorkerRejectsCandidateDriftAndWritableRole(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	if _, err := start([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	prompt := filepath.Join(t.TempDir(), "assignment.txt")
	if err := os.WriteFile(prompt, []byte("Review."), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := worker([]string{"--repo", repo, "--candidate", "candidate-b", "--role", "specialist-reviewer", "--prompt-file", prompt}); err == nil {
		t.Fatal("candidate drift must block worker launch")
	}
	if _, err := worker([]string{"--repo", repo, "--candidate", "candidate-a", "--role", "orchestrator", "--prompt-file", prompt}); err == nil {
		t.Fatal("writable role must not use read-only worker adapter")
	}
}

func TestTeamRunsDistinctFrozenWorkers(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("JSDLC_PLUGIN_ROOT", filepath.Clean(filepath.Join(workingDir, "..", "..", "plugins", "jerry-sdlc")))
	binDir := t.TempDir()
	fakeCodex := filepath.Join(binDir, "codex")
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then echo "codex-test 1.0"; exit 0; fi
while IFS= read -r line; do :; done
id=thread-$$
printf '%s\n' "{\"type\":\"thread.started\",\"thread_id\":\"$id\"}" '{"type":"item.completed","item":{"type":"agent_message","text":"{\"disposition\":\"CLEAN\",\"evidence\":[\"checked\"],\"findings\":[],\"limitations\":[],\"domains\":[]}"}}'
`
	if err := os.WriteFile(fakeCodex, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	repo := t.TempDir()
	if _, err := start([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	got, err := team([]string{"--repo", repo, "--candidate", "candidate-a", "--objective", "Release review"})
	if err != nil {
		t.Fatal(err)
	}
	if got["assurance"] != "MANAGED_SEPARATE_PASSES" || got["workerObservation"] != "OBSERVED_DISTINCT_SUBPROCESSES" || got["scope"] != "THIS_COMMAND_ONLY" || got["verdict"] != "INCONCLUSIVE" {
		t.Fatalf("unexpected team result: %#v", got)
	}
	if len(got["roles"].([]result)) != 4 {
		t.Fatalf("expected four role results: %#v", got)
	}
	if digest, ok := got["contractSetDigest"].(string); !ok || len(digest) != 64 {
		t.Fatalf("missing frozen contract-set digest: %#v", got)
	}
}

func TestTeamRejectsChangesSinceStart(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	if _, err := start([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "changed.txt"), []byte("drift"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := team([]string{"--repo", repo, "--candidate", "candidate-a"}); err == nil {
		t.Fatal("team must reject content changed after start")
	}
}

func TestRoleWorkerRejectsDifferentExpectedRun(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	if _, err := start([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoleWorker(repo, "candidate-a", "different-run", "qa-architect", []byte("Review."), nil); err == nil {
		t.Fatal("worker must reject a different expected team run")
	}
}

func TestWorkerReportRequiresCompleteFindingContract(t *testing.T) {
	valid := `{"disposition":"FINDINGS","evidence":[],"findings":[{"id":"F-1","severity":"HIGH","confidence":"HIGH","requirement":"No traversal","location":"main.go:1","evidence":"observed","recommendation":"validate"}],"limitations":[],"domains":[]}`
	if err := validateWorkerReport(valid); err != nil {
		t.Fatal(err)
	}
	missingConfidence := `{"disposition":"FINDINGS","evidence":[],"findings":[{"id":"F-1","severity":"HIGH","requirement":"No traversal","location":"main.go:1","evidence":"observed","recommendation":"validate"}],"limitations":[],"domains":[]}`
	if err := validateWorkerReport(missingConfidence); err == nil {
		t.Fatal("finding without confidence must be rejected")
	}
}

func TestWorkerReportRequiresSemanticDisposition(t *testing.T) {
	invalid := []string{
		`{"disposition":"CLEAN","evidence":[],"findings":[],"limitations":[]}`,
		`{"disposition":"CLEAN","evidence":["checked"],"findings":[{"id":"F-1","severity":"LOW","confidence":"HIGH","requirement":"r","location":"x","evidence":"e","recommendation":"do"}],"limitations":[]}`,
		`{"disposition":"FINDINGS","evidence":["checked"],"findings":[],"limitations":[]}`,
		`{"disposition":"BLOCKED","evidence":[],"findings":[],"limitations":[]}`,
		`{"disposition":"CLEAN","evidence":["  "],"findings":[],"limitations":[]}`,
		`{"disposition":"INCONCLUSIVE","evidence":[],"findings":[],"limitations":[""]}`,
	}
	for _, report := range invalid {
		if err := validateWorkerReport(report); err == nil {
			t.Fatalf("accepted inconsistent report: %s", report)
		}
	}
}

func TestAggregateTeamVerdictRequiresEveryDomain(t *testing.T) {
	domains := make([]domainResult, 0, len(requiredReleaseDomains))
	for _, domain := range requiredReleaseDomains {
		domains = append(domains, domainResult{Domain: domain, Status: "PASS", Evidence: "verified"})
	}
	report, err := json.Marshal(workerReport{Disposition: "CLEAN", Evidence: []string{"checked"}, Findings: []workerFinding{}, Limitations: []string{}, Domains: domains})
	if err != nil {
		t.Fatal(err)
	}
	cleanEmpty, err := json.Marshal(workerReport{Disposition: "CLEAN", Evidence: []string{"checked"}, Findings: []workerFinding{}, Limitations: []string{}, Domains: []domainResult{}})
	if err != nil {
		t.Fatal(err)
	}
	outputs := []result{{"role": "qa-architect", "report": json.RawMessage(cleanEmpty)}, {"role": "qa-executor", "report": json.RawMessage(cleanEmpty)}, {"role": "specialist-reviewer", "report": json.RawMessage(cleanEmpty)}, {"role": "independent-verifier", "report": json.RawMessage(report)}}
	verdict, _ := aggregateTeamVerdict(outputs)
	if verdict != "READY" {
		t.Fatalf("expected READY, got %s", verdict)
	}
	report, err = json.Marshal(workerReport{Disposition: "CLEAN", Evidence: []string{"checked"}, Findings: []workerFinding{}, Limitations: []string{}, Domains: domains[:len(domains)-1]})
	if err != nil {
		t.Fatal(err)
	}
	outputs[3]["report"] = json.RawMessage(report)
	verdict, _ = aggregateTeamVerdict(outputs)
	if verdict != "INCONCLUSIVE" {
		t.Fatalf("missing domain must be inconclusive, got %s", verdict)
	}
	verdict, _ = aggregateTeamVerdict(outputs[:3])
	if verdict != "INCONCLUSIVE" {
		t.Fatalf("missing role must be inconclusive, got %s", verdict)
	}
}

func TestAggregateTeamVerdictFailsClosed(t *testing.T) {
	clean := func(disposition string, domains []domainResult) json.RawMessage {
		findings := []workerFinding{}
		if disposition == "FINDINGS" {
			findings = append(findings, workerFinding{ID: "F-1", Severity: "HIGH", Confidence: "HIGH", Requirement: "safe", Location: "x", Evidence: "broken", Recommendation: "fix"})
		}
		b, err := json.Marshal(workerReport{Disposition: disposition, Evidence: []string{"checked"}, Findings: findings, Limitations: []string{}, Domains: domains})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	passes := make([]domainResult, 0, len(requiredReleaseDomains))
	for _, domain := range requiredReleaseDomains {
		passes = append(passes, domainResult{Domain: domain, Status: "PASS", Evidence: "verified"})
	}
	base := func() []result {
		return []result{{"role": "qa-architect", "report": clean("CLEAN", nil)}, {"role": "qa-executor", "report": clean("CLEAN", nil)}, {"role": "specialist-reviewer", "report": clean("CLEAN", nil)}, {"role": "independent-verifier", "report": clean("CLEAN", passes)}}
	}
	duplicate := base()
	duplicate[3]["role"] = "specialist-reviewer"
	finding := base()
	finding[1] = result{"role": "qa-executor", "report": clean("FINDINGS", nil)}
	blocked := base()
	blocked[1] = result{"role": "qa-executor", "report": clean("BLOCKED", nil)}
	naDomains := append([]domainResult{}, passes...)
	naDomains[0] = domainResult{Domain: requiredReleaseDomains[0], Status: "NOT_APPLICABLE", Evidence: "claimed n/a"}
	notApplicable := base()
	notApplicable[3] = result{"role": "independent-verifier", "report": clean("CLEAN", naDomains)}
	for name, tc := range map[string]struct {
		outputs []result
		want    string
	}{"duplicate": {duplicate, "INCONCLUSIVE"}, "finding": {finding, "NOT_READY"}, "blocked": {blocked, "INCONCLUSIVE"}, "not-applicable": {notApplicable, "INCONCLUSIVE"}} {
		if got, _ := aggregateTeamVerdict(tc.outputs); got != tc.want {
			t.Fatalf("%s: got %s want %s", name, got, tc.want)
		}
	}
}

func TestDoctorIndependentAssuranceFailsClosed(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	got, err := doctor(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["outcome"] != "MANAGED_SEPARATE_PASSES" {
		t.Fatalf("unexpected outcome: %#v", got)
	}
	if _, err := start([]string{"--repo", t.TempDir(), "--candidate", "x", "--assurance", "MANAGED_INDEPENDENT"}); err == nil {
		t.Fatal("independent assurance must require unavailable adapter attestation")
	}
}

func TestStateResumeTransitionAndDrift(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	if _, err := start([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	got, err := status([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if got["stale"] != false {
		t.Fatalf("unexpected stale status: %#v", got)
	}
	if _, err := transition([]string{"--repo", repo, "--candidate", "candidate-b", "--to", "STRATEGY_READY"}); err == nil {
		t.Fatal("candidate drift should block transition")
	}
	if _, err := transition([]string{"--repo", repo, "--candidate", "candidate-a", "--to", "STRATEGY_READY"}); err != nil {
		t.Fatal(err)
	}
	got, err = status([]string{"--repo", repo, "--candidate", "candidate-b"})
	if err != nil {
		t.Fatal(err)
	}
	if got["stale"] != true {
		t.Fatalf("expected stale status: %#v", got)
	}
}

func TestStartRejectsUnsupportedWorkflow(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, err := start([]string{"--repo", t.TempDir(), "--candidate", "candidate-a", "--workflow", "../../secret"}); err == nil {
		t.Fatal("unsupported workflow path must be rejected")
	}
}

func TestVerifyCannotEmitReadyInPhaseOne(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	if _, err := start([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	got, err := verify([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if got["verdict"] != "INCONCLUSIVE" {
		t.Fatalf("unexpected verdict: %#v", got)
	}
	for _, state := range []string{"STRATEGY_READY", "EVIDENCE_COLLECTED", "REVIEWED", "ADJUDICATED", "VERIFIED"} {
		if _, err := transition([]string{"--repo", repo, "--candidate", "candidate-a", "--to", state}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := transition([]string{"--repo", repo, "--candidate", "candidate-a", "--to", "READY"}); err == nil {
		t.Fatal("Phase 1 transition must never emit READY")
	}
}

func TestLockAndSymlinkIdentity(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	link := filepath.Join(t.TempDir(), "repo-link")
	if err := os.Symlink(repo, link); err != nil {
		t.Fatal(err)
	}
	if _, err := start([]string{"--repo", link, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	_, key, root, err := stateLocation(repo)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, key, "active.json")
	unlock, err := acquireLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, err := transition([]string{"--repo", repo, "--candidate", "candidate-a", "--to", "STRATEGY_READY"}); err == nil {
		t.Fatal("concurrent transition should fail while lock is held")
	}
}

func TestForbiddenTransitionAndIncompatibleSchema(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", root)
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transition([]string{"--repo", repo, "--candidate", "candidate-a", "--to", "READY"}); err == nil {
		t.Fatal("forbidden transition should fail")
	}
	path := started["path"].(string)
	if err := os.WriteFile(path, []byte(`{"schemaVersion":99}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := status([]string{"--repo", repo}); err == nil {
		t.Fatal("incompatible schema should fail")
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatal(err)
	}
}
