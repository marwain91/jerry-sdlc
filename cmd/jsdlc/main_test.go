package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
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

func TestStateUpgradeAndRollbackAreReversible(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	path := started["path"].(string)
	legacy := started["run"].(runState)
	legacy.SchemaVersion = 1
	legacy.ContentDigest = ""
	b, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	upgraded, err := upgradeState([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if upgraded["run"].(runState).SchemaVersion != 2 || upgraded["run"].(runState).ContentDigest == "" {
		t.Fatalf("upgrade failed: %#v", upgraded)
	}
	rolledBack, err := rollbackState([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack["run"].(runState).SchemaVersion != 1 {
		t.Fatalf("rollback failed: %#v", rolledBack)
	}
	current, _, err := readStateLocationForTest(repo)
	if err != nil || current.SchemaVersion != 1 {
		t.Fatalf("legacy state not restored: %#v %v", current, err)
	}
}

func readStateLocationForTest(repo string) (runState, string, error) {
	_, key, root, err := stateLocation(repo)
	if err != nil {
		return runState{}, "", err
	}
	return readState(root, key)
}

func TestStateRollbackRefusesPostMigrationTransition(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	path := started["path"].(string)
	legacy := started["run"].(runState)
	legacy.SchemaVersion = 1
	b, _ := json.Marshal(legacy)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := upgradeState([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := transition([]string{"--repo", repo, "--candidate", "candidate-a", "--to", "STRATEGY_READY"}); err != nil {
		t.Fatal(err)
	}
	if _, err := rollbackState([]string{"--repo", repo, "--candidate", "candidate-a"}); err == nil {
		t.Fatal("rollback after state progress must be refused")
	}
}

func TestInterruptedAtomicWriteLeavesActiveStateReadable(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	path := started["path"].(string)
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), ".jsdlc-interrupted.tmp"), []byte(`{"partial":`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := status([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil || got["run"].(runState).ID != started["run"].(runState).ID {
		t.Fatalf("interrupted temp affected active state: %#v %v", got, err)
	}
}

func TestHelperProcessAtomicWrite(t *testing.T) {
	if os.Getenv("JSDLC_HELPER_ATOMIC_WRITE") != "1" {
		return
	}
	pause := func(string) {
		if err := os.WriteFile(os.Getenv("JSDLC_HELPER_MARKER"), []byte("ready"), 0o600); err != nil {
			os.Exit(3)
		}
		for {
			time.Sleep(time.Hour)
		}
	}
	if os.Getenv("JSDLC_HELPER_BOUNDARY") == "after" {
		writeAtomicAfterRenameHook = pause
	} else {
		writeAtomicBeforeRenameHook = pause
	}
	if err := writeAtomic(os.Getenv("JSDLC_HELPER_TARGET"), []byte(os.Getenv("JSDLC_HELPER_PAYLOAD"))); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func TestProcessKillAfterRenameLeavesCompletePublishedState(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	base := runState{SchemaVersion: 2, ID: "run", Workflow: "release-readiness", State: "BASELINED", Assurance: "MANAGED_SEPARATE_PASSES", Repository: "/repo", Candidate: "candidate", ContentDigest: strings.Repeat("a", 64), CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}
	next := base
	next.State = "STRATEGY_READY"
	newBytes, _ := json.Marshal(next)
	for _, old := range [][]byte{nil, mustJSON(t, base)} {
		dir := t.TempDir()
		target, marker := filepath.Join(dir, "active.json"), filepath.Join(dir, "marker")
		if old != nil {
			if err := os.WriteFile(target, old, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(executable, "-test.run=^TestHelperProcessAtomicWrite$")
		cmd.Env = append(os.Environ(), "JSDLC_HELPER_ATOMIC_WRITE=1", "JSDLC_HELPER_BOUNDARY=after", "JSDLC_HELPER_TARGET="+target, "JSDLC_HELPER_MARKER="+marker, "JSDLC_HELPER_PAYLOAD="+string(newBytes))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for {
			if _, err := os.Stat(marker); err == nil {
				break
			}
			if time.Now().After(deadline) {
				_ = cmd.Process.Kill()
				t.Fatal("helper never reached post-rename boundary")
			}
			time.Sleep(5 * time.Millisecond)
		}
		if err := cmd.Process.Kill(); err != nil {
			t.Fatal(err)
		}
		_ = cmd.Wait()
		published, err := os.ReadFile(target)
		if err != nil || !bytes.Equal(published, newBytes) {
			t.Fatalf("post-rename state was partial or absent: %q %v", published, err)
		}
		if got, err := decodeRunState(published); err != nil || got.State != "STRATEGY_READY" {
			t.Fatalf("post-rename state invalid: %#v %v", got, err)
		}
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestProcessKillAtRenameBoundaryPreservesAtomicState(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	base := runState{SchemaVersion: 2, ID: "run", Workflow: "release-readiness", State: "BASELINED", Assurance: "MANAGED_SEPARATE_PASSES", Repository: "/repo", Candidate: "candidate", ContentDigest: strings.Repeat("a", 64), CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}
	oldBytes, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	next := base
	next.State = "STRATEGY_READY"
	newBytes, err := json.Marshal(next)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		old  []byte
	}{{"create", nil}, {"replace", oldBytes}} {
		dir := t.TempDir()
		target, marker := filepath.Join(dir, "active.json"), filepath.Join(dir, "marker")
		if tc.old != nil {
			if err := os.WriteFile(target, tc.old, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(executable, "-test.run=^TestHelperProcessAtomicWrite$")
		var helperStderr bytes.Buffer
		cmd.Stderr = &helperStderr
		cmd.Env = append(os.Environ(), "JSDLC_HELPER_ATOMIC_WRITE=1", "JSDLC_HELPER_TARGET="+target, "JSDLC_HELPER_MARKER="+marker, "JSDLC_HELPER_PAYLOAD="+string(newBytes))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for {
			if _, err := os.Stat(marker); err == nil {
				break
			}
			if time.Now().After(deadline) {
				_ = cmd.Process.Kill()
				t.Fatal("helper never reached rename boundary")
			}
			time.Sleep(5 * time.Millisecond)
		}
		if err := cmd.Process.Kill(); err != nil {
			t.Fatal(err)
		}
		err := cmd.Wait()
		exitErr, ok := err.(*exec.ExitError)
		if !ok || !exitErr.Sys().(syscall.WaitStatus).Signaled() || exitErr.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
			t.Fatalf("helper was not killed at boundary: %v: %s", err, helperStderr.String())
		}
		b, readErr := os.ReadFile(target)
		if tc.old == nil {
			if !os.IsNotExist(readErr) {
				t.Fatalf("create published before rename: %q %v", b, readErr)
			}
		} else if readErr != nil || !bytes.Equal(b, tc.old) {
			t.Fatalf("replacement lost old state: %q %v", b, readErr)
		}
		if readErr == nil {
			if _, err := decodeRunState(b); err != nil {
				t.Fatalf("surviving state is invalid: %v", err)
			}
		}
		if err := writeAtomic(target, newBytes); err != nil {
			t.Fatal(err)
		}
		b, err = os.ReadFile(target)
		if err != nil || !bytes.Equal(b, newBytes) {
			t.Fatalf("successful write not committed: %q %v", b, err)
		}
		if got, err := decodeRunState(b); err != nil || got.State != "STRATEGY_READY" {
			t.Fatalf("committed state is invalid: %#v %v", got, err)
		}
	}
}

func TestStateDecoderRejectsTraversalAndNonHexDigest(t *testing.T) {
	base := runState{SchemaVersion: 2, ID: "run", Workflow: "release-readiness", State: "BASELINED", Assurance: "MANAGED_SEPARATE_PASSES", Repository: "/repo", Candidate: "candidate", ContentDigest: strings.Repeat("a", 64), CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}
	badDigest := base
	badDigest.ContentDigest = strings.Repeat("z", 64)
	badMigration := base
	badMigration.MigrationID = "../escape"
	badMigration.MigratedAt = "2026-01-01T00:00:00Z"
	badMigration.MigrationBackupDigest = strings.Repeat("b", 64)
	for _, state := range []runState{badDigest, badMigration} {
		b, err := json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeRunState(b); err == nil {
			t.Fatalf("accepted invalid state: %#v", state)
		}
	}
}

func TestStateRollbackRejectsTamperedBackup(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	path := started["path"].(string)
	legacy := started["run"].(runState)
	legacy.SchemaVersion = 1
	legacy.ContentDigest = ""
	b, _ := json.Marshal(legacy)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	upgraded, err := upgradeState([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	backup := upgraded["backup"].(string)
	if err := os.WriteFile(backup, []byte(`{"schemaVersion":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := rollbackState([]string{"--repo", repo, "--candidate", "candidate-a"}); err == nil {
		t.Fatal("tampered backup must be rejected")
	}
}

func TestStateRollbackRejectsRepositoryMutation(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	path := started["path"].(string)
	legacy := started["run"].(runState)
	legacy.SchemaVersion = 1
	legacy.ContentDigest = ""
	b, _ := json.Marshal(legacy)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := upgradeState([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "drift.txt"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := rollbackState([]string{"--repo", repo, "--candidate", "candidate-a"}); err == nil {
		t.Fatal("rollback after repository drift must be rejected")
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

func TestClassifyEverydayWorkflows(t *testing.T) {
	tests := map[string]string{
		"Fix this bug in the account exporter":            "bug-fix",
		"Investigate this bug and explain the root cause": "bug-diagnosis",
		"Review this PR for correctness":                  "pr-review",
		"Fix this typo in the README":                     "trivial-change",
		"Implement account export":                        "feature",
		"Investigate the production outage":               "incident",
		"Prepare this project for release":                "release-readiness",
	}
	for request, want := range tests {
		got, err := classify([]string{"--request", request})
		if err != nil {
			t.Fatalf("classify %q: %v", request, err)
		}
		if got["workflow"] != want {
			t.Errorf("classify %q = %q, want %q", request, got["workflow"], want)
		}
	}
}

func TestRolesMatchEverydayWorkflowTeams(t *testing.T) {
	for workflow, want := range map[string]string{
		"feature":        "delivery-planner,implementer,qa-executor,code-reviewer,verifier",
		"bug-fix":        "debugger,implementer,qa-executor,code-reviewer,verifier",
		"bug-diagnosis":  "debugger,code-reviewer",
		"pr-review":      "code-reviewer,qa-executor,verifier",
		"trivial-change": "implementer,verifier",
		"incident":       "incident-commander,debugger,qa-executor,verifier",
	} {
		got, err := roles([]string{"--workflow", workflow})
		if err != nil {
			t.Fatalf("roles %s: %v", workflow, err)
		}
		if strings.Join(got["roles"].([]string), ",") != want {
			t.Errorf("roles %s = %#v, want %s", workflow, got["roles"], want)
		}
	}
	if _, err := roles([]string{"--workflow", "invented"}); err == nil {
		t.Fatal("unknown workflow roles must fail")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, err := start([]string{"--repo", t.TempDir(), "--candidate", "candidate-a", "--workflow", "feature"}); err == nil {
		t.Fatal("Codex-native workflow must not claim the persisted release contract")
	}
}

func TestReleaseIntentComposesActionAndReleaseConcept(t *testing.T) {
	for _, request := range []string{
		"Decide whether the gateway can be promoted to production",
		"Certify the worker for tonight's deployment",
		"Approve or block shipping the SDK",
		"Validate the editor for general availability",
		"Sign off the platform for launch",
		"Tell me whether we should deploy the gateway",
	} {
		if !releaseIntent(strings.ToLower(request)) {
			t.Errorf("expected release intent: %q", request)
		}
	}
}

func TestReleaseIntentRejectsReferentialReleaseLanguage(t *testing.T) {
	for _, request := range []string{
		"Explain the release gate used by the gateway",
		"Test the launch sign-off parser in the worker",
		"Build a release QA dashboard for the SDK",
		"Read the launch decision from the editor",
		"Persist a release decision in the platform",
	} {
		if releaseIntent(strings.ToLower(request)) {
			t.Errorf("unexpected release intent: %q", request)
		}
	}
}

func TestReleaseIntentHandlesMixedCommandsNegationAndIncidentalLanguage(t *testing.T) {
	for _, request := range []string{
		"update the dependencies, then release the service",
		"do not deploy the old service; instead release the replacement",
	} {
		if !releaseIntent(request) {
			t.Errorf("explicit release clause must trigger: %q", request)
		}
	}
	for _, request := range []string{
		"do not deploy the service",
		"where is the production config?",
		"publish the internal architecture documentation for the service to the team wiki",
		"explain the difference between deploy and release workflows",
	} {
		if releaseIntent(request) {
			t.Errorf("unexpected release intent: %q", request)
		}
	}
}

func TestReleaseIntentCoversRiskAndReadinessLanguage(t *testing.T) {
	for _, request := range []string{
		"audit the service before deployment and identify unresolved risks",
		"is the library in good enough shape for a public release?",
		"what would prevent the app from being safely released today?",
		"evaluate the worker against a practical ship checklist",
		"before rollout, inspect the API and tell me whether to sign off",
		"find blockers that should stop tomorrow's launch of the service",
		"check whether the app is ready for final store submission and customer availability",
		"finish hardening the worker so I can confidently promote it to production",
		"turn the library into a release candidate we can send to customers this week",
		"treat the current build as a release candidate and provide a readiness verdict",
		"before I cut the release build, confirm whether the service meets the bar to ship",
	} {
		if !releaseIntent(request) {
			t.Errorf("expected release intent: %q", request)
		}
	}
	if releaseIntent("find where the production endpoint is defined") {
		t.Fatal("referential production lookup must not trigger release readiness")
	}
	if releaseIntent("turn the sentence 'the app is ready to ship' into a concise slide heading") {
		t.Fatal("quoted release language transformation must not trigger release readiness")
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

func TestClassifyTriggersAreBoundedAndDeduplicated(t *testing.T) {
	got, err := classify([]string{"--request", "review the build", "--files", "db/migration.sql"})
	if err != nil {
		t.Fatal(err)
	}
	triggers := got["triggers"].([]string)
	if len(triggers) != 1 || triggers[0] != "data-migration" {
		t.Fatalf("substring or duplicate trigger leaked through: %#v", got)
	}
	got, err = classify([]string{"--request", "review authentication and API UI changes"})
	if err != nil {
		t.Fatal(err)
	}
	triggers = got["triggers"].([]string)
	if strings.Join(triggers, ",") != "security,api-compatibility,ux-accessibility" {
		t.Fatalf("expected bounded triggers: %#v", got)
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

func TestEvalTriggersBindsOutputToFixture(t *testing.T) {
	body := []byte(`{"schemaVersion":1,"subjects":["app"],"cases":[{"id":"p","template":"release the {project}","release":true},{"id":"n","template":"fix the {project}","release":false}]}`)
	path := filepath.Join(t.TempDir(), "triggers.json")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := evalTriggers([]string{"--fixture", path, "--repeats", "2"})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(body)
	if got["evaluationSchemaVersion"] != 1 || got["fixtureDigest"] != fmt.Sprintf("%x", digest) || got["failureCount"] != 0 || got["failuresTruncated"] != false {
		t.Fatalf("evaluation provenance is incomplete: %#v", got)
	}
}

func TestEvalQualityComputesGraduationMetrics(t *testing.T) {
	zero := 0
	tasks := make([]qualityTask, 20)
	for i := range tasks {
		baseline := qualityRun{Findings: []adjudicatedFinding{{ID: "medium", Outcome: "TRUE_POSITIVE"}}, WallMilliseconds: 100, Tokens: 100, HumanReviewMinutes: 10, EvidenceFabrications: &zero, CorrectionRegressions: &zero}
		jerry := qualityRun{Findings: []adjudicatedFinding{{ID: "high", Outcome: "TRUE_POSITIVE"}, {ID: "medium", Outcome: "TRUE_POSITIVE"}}, WallMilliseconds: 200, Tokens: 200, HumanReviewMinutes: 5, EvidenceFabrications: &zero, CorrectionRegressions: &zero}
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
	fixtureDigest := sha256.Sum256(b)
	if got["evaluationSchemaVersion"] != 1 || got["fixtureDigest"] != fmt.Sprintf("%x", fixtureDigest) {
		t.Fatalf("quality result lacks fixture provenance: %#v", got)
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
	zero, negative, tooMany := 0, -1, maxEventCount+1
	cases := []qualityRun{
		{Findings: []adjudicatedFinding{{ID: "unknown", Outcome: "TRUE_POSITIVE"}}, WallMilliseconds: 1, Tokens: 1, EvidenceFabrications: &zero, CorrectionRegressions: &zero},
		{Findings: []adjudicatedFinding{{ID: "known", Outcome: "FALSE_POSITIVE"}}, WallMilliseconds: 1, Tokens: 1, EvidenceFabrications: &zero, CorrectionRegressions: &zero},
		{Findings: []adjudicatedFinding{}, WallMilliseconds: 1, Tokens: 1, UnauthorizedActions: -1, EvidenceFabrications: &zero, CorrectionRegressions: &zero},
		{Findings: []adjudicatedFinding{}, WallMilliseconds: int64(^uint64(0) >> 1), Tokens: 1, EvidenceFabrications: &zero, CorrectionRegressions: &zero},
		{Findings: []adjudicatedFinding{}, WallMilliseconds: 1, Tokens: int64(^uint64(0) >> 1), EvidenceFabrications: &zero, CorrectionRegressions: &zero},
		{Findings: []adjudicatedFinding{}, WallMilliseconds: 1, Tokens: 1, EvidenceFabrications: &negative, CorrectionRegressions: &zero},
		{Findings: []adjudicatedFinding{}, WallMilliseconds: 1, Tokens: 1, EvidenceFabrications: &zero, CorrectionRegressions: &tooMany},
		{Findings: []adjudicatedFinding{}, WallMilliseconds: 1, Tokens: 1, EvidenceFabrications: nil, CorrectionRegressions: &zero},
		{Findings: []adjudicatedFinding{}, WallMilliseconds: 1, Tokens: 1, EvidenceFabrications: &zero, CorrectionRegressions: nil},
	}
	for _, run := range cases {
		if _, err := scoreQualityRun(run, known); err == nil {
			t.Fatalf("accepted gameable run: %#v", run)
		}
	}
}

func TestQualityEvaluatorFailsFabricationAndCorrectionRegression(t *testing.T) {
	zero := 0
	tasks := make([]qualityTask, 20)
	for i := range tasks {
		baseline := qualityRun{Findings: []adjudicatedFinding{{ID: "medium", Outcome: "TRUE_POSITIVE"}}, WallMilliseconds: 100, Tokens: 100, EvidenceFabrications: &zero, CorrectionRegressions: &zero}
		jerry := qualityRun{Findings: []adjudicatedFinding{{ID: "high", Outcome: "TRUE_POSITIVE"}, {ID: "medium", Outcome: "TRUE_POSITIVE"}}, WallMilliseconds: 100, Tokens: 100, EvidenceFabrications: &zero, CorrectionRegressions: &zero}
		tasks[i] = qualityTask{ID: fmt.Sprintf("task-%d", i), Source: fmt.Sprintf("repo-%d", i), Candidate: fmt.Sprintf("commit-%d", i), Known: []knownFinding{{ID: "high", Severity: "HIGH"}, {ID: "medium", Severity: "MEDIUM"}}, Baseline: []qualityRun{baseline, baseline, baseline}, Jerry: []qualityRun{jerry, jerry, jerry}}
	}
	one := 1
	tasks[0].Jerry[0].EvidenceFabrications = &one
	tasks[1].Jerry[0].CorrectionRegressions = &one
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
	if got["passed"] != false || got["evidenceFabrications"] != 1 || got["correctionRegressions"] != 1 {
		t.Fatalf("fabrication and regression must fail graduation: %#v", got)
	}
}

func TestQualityEvaluatorRejectsUndefinedRelativeUplift(t *testing.T) {
	zero := 0
	tasks := make([]qualityTask, 20)
	for i := range tasks {
		baseline := qualityRun{Findings: []adjudicatedFinding{}, WallMilliseconds: 100, Tokens: 100, EvidenceFabrications: &zero, CorrectionRegressions: &zero}
		jerry := qualityRun{Findings: []adjudicatedFinding{{ID: "high", Outcome: "TRUE_POSITIVE"}}, WallMilliseconds: 100, Tokens: 100, EvidenceFabrications: &zero, CorrectionRegressions: &zero}
		tasks[i] = qualityTask{ID: fmt.Sprintf("zero-task-%d", i), Source: fmt.Sprintf("zero-repo-%d", i), Candidate: fmt.Sprintf("zero-commit-%d", i), Known: []knownFinding{{ID: "high", Severity: "HIGH"}}, Baseline: []qualityRun{baseline, baseline, baseline}, Jerry: []qualityRun{jerry, jerry, jerry}}
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
	if got["passed"] != false || got["relativeRecallImprovementDefined"] != false || got["absoluteRecallImprovement"] != 1.0 {
		t.Fatalf("zero baseline cannot establish relative uplift: %#v", got)
	}
	thresholds := got["thresholds"].(result)
	if thresholds["baselineRecallPositive"] != false || thresholds["relativeRecallAtLeast25Percent"] != false {
		t.Fatalf("undefined relative uplift thresholds must fail: %#v", thresholds)
	}
}

func TestMedianAvoidsOverflow(t *testing.T) {
	max := int64(^uint64(0) >> 1)
	if got := median([]int64{max - 2, max}); got != max-1 {
		t.Fatalf("overflow-safe median got %d", got)
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

func TestCheckCapturesCandidateBoundCommandEvidence(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := check([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "unit-tests", "--domains", "functional,reliability", "--authorized", "--", "/bin/true"})
	if err != nil {
		t.Fatal(err)
	}
	if got["reportedPass"] != true || got["readinessEffect"] != "NONE_UNATTESTED" || got["execution"] != "FULLY_PRIVILEGED_LOCAL_COMMAND" {
		t.Fatalf("successful check did not pass: %#v", got)
	}
	_, key, root, err := stateLocation(repo)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := loadCheckEvidence(root, key, started["run"].(runState).ID)
	if err != nil {
		t.Fatal(err)
	}
	evidence := loaded["unit-tests"]
	if evidence.RunID != started["run"].(runState).ID || evidence.Candidate != "candidate-a" || evidence.RepositoryDigest != started["run"].(runState).ContentDigest || evidence.ExitStatus != 0 {
		t.Fatalf("evidence was not candidate-bound: %#v", evidence)
	}
	failed, err := check([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "failing-test", "--domains", "functional", "--authorized", "--", "/bin/false"})
	if err != nil {
		t.Fatal(err)
	}
	if failed["reportedPass"] != false {
		t.Fatalf("failed command must not pass: %#v", failed)
	}
	if _, err := check([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "not-authorized", "--domains", "functional", "--", "/bin/true"}); err == nil {
		t.Fatal("fully privileged check must require explicit authorization acknowledgement")
	}
}

func TestCheckRecordsRepositoryMutationAsFailure(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	if _, err := start([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	got, err := check([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "mutating", "--domains", "functional", "--authorized", "--", "/bin/sh", "-c", "printf changed > changed.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if got["reportedPass"] != false || got["evidence"].(checkEvidence).RepositoryChanged != true {
		t.Fatalf("mutating check must be recorded as failed: %#v", got)
	}
}

func TestCheckedEvidenceCannotBeReusedOrTampered(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := check([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "unit-tests", "--domains", "functional", "--authorized", "--", "/bin/true"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := check([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "unit-tests", "--domains", "functional", "--authorized", "--", "/bin/true"}); err == nil {
		t.Fatal("evidence ID reuse must fail")
	}
	path := got["path"].(string)
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, key, root, err := stateLocation(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadCheckEvidence(root, key, started["run"].(runState).ID); err == nil {
		t.Fatal("tampered evidence must fail")
	}
}

func TestConcurrentCheckIDIsReservedBeforeExecution(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	if _, err := start([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	args := []string{"--repo", repo, "--candidate", "candidate-a", "--id", "same-id", "--domains", "functional", "--authorized", "--", "/bin/sh", "-c", "sleep 0.1"}
	startGate := make(chan struct{})
	errs := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for i := 0; i < 2; i++ {
		go func() { ready.Done(); <-startGate; _, err := check(args); errs <- err }()
	}
	ready.Wait()
	close(startGate)
	successes := 0
	for i := 0; i < 2; i++ {
		if <-errs == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("exactly one reserved check must execute successfully, got %d", successes)
	}
}

func TestCheckRejectsRunTransitionDuringExecution(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	if _, err := start([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "started")
	done := make(chan error, 1)
	go func() {
		_, err := check([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "slow", "--domains", "functional", "--authorized", "--", "/bin/sh", "-c", "touch \"$1\"; sleep 0.2", "sh", marker})
		done <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("check command did not start")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := transition([]string{"--repo", repo, "--candidate", "candidate-a", "--to", "CANCELLED"}); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err == nil || !strings.Contains(err.Error(), "active run changed") {
		t.Fatalf("transitioned run accepted check evidence: %v", err)
	}
}

func TestCheckTerminatesBackgroundDescendantsBeforeDigest(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	if _, err := start([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	late := filepath.Join(repo, "late.txt")
	childReady := filepath.Join(t.TempDir(), "child-ready")
	command := fmt.Sprintf("(touch %q; sleep 0.2; printf late > %q) >/dev/null 2>&1 & while [ ! -e %q ]; do :; done", childReady, late, childReady)
	got, err := check([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "background", "--domains", "functional", "--authorized", "--", "/bin/sh", "-c", command})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if _, err := os.Stat(late); !os.IsNotExist(err) {
		t.Fatalf("background descendant survived process-group containment: %v", err)
	}
	if got["reportedPass"] != true || got["evidence"].(checkEvidence).RepositoryChanged {
		t.Fatalf("contained check was recorded incorrectly: %#v", got)
	}
}

func TestHelperCheckStartGap(t *testing.T) {
	if os.Getenv("JSDLC_HELPER_CHECK_START_GAP") != "1" {
		return
	}
	checkAfterStartHook = func() {
		_ = os.WriteFile(os.Getenv("JSDLC_HELPER_MARKER"), []byte("started"), 0o600)
		for {
			time.Sleep(time.Hour)
		}
	}
	_, _ = check([]string{"--repo", os.Getenv("JSDLC_HELPER_REPO"), "--candidate", "candidate-a", "--id", "start-gap", "--domains", "functional", "--authorized", "--", "/bin/sh", "-c", "echo $$ > \"$1\"; sleep 30", "sh", os.Getenv("JSDLC_HELPER_CHILD_PID")})
	os.Exit(4)
}

func TestStartGapReservationCannotBeUnsafelyRecovered(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	marker, childPIDPath := filepath.Join(tmp, "hook"), filepath.Join(tmp, "child-pid")
	cmd := exec.Command(executable, "-test.run=^TestHelperCheckStartGap$")
	cmd.Env = append(os.Environ(), "JSDLC_HELPER_CHECK_START_GAP=1", "JSDLC_HELPER_REPO="+repo, "JSDLC_HELPER_MARKER="+marker, "JSDLC_HELPER_CHILD_PID="+childPIDPath)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		_, markerErr := os.Stat(marker)
		pidBytes, pidErr := os.ReadFile(childPIDPath)
		if markerErr == nil && pidErr == nil && len(pidBytes) > 0 {
			break
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			t.Fatal("helper did not reach post-start gap")
		}
		time.Sleep(5 * time.Millisecond)
	}
	pidBytes, err := os.ReadFile(childPIDPath)
	if err != nil {
		t.Fatal(err)
	}
	childPID, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		t.Fatal(err)
	}
	defer terminateProcessGroup(childPID)
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	if _, err := recoverCheck([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "start-gap", "--authorized"}); err == nil || !strings.Contains(err.Error(), "recovery is unsafe") {
		t.Fatalf("indeterminate reservation was recovered: %v", err)
	}
	_, key, root, err := stateLocation(repo)
	if err != nil {
		t.Fatal(err)
	}
	reservationPath := filepath.Join(root, key, "check-reservations", started["run"].(runState).ID, "start-gap.json")
	if err := terminateProcessGroup(childPID); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(reservationPath); err != nil {
		t.Fatal(err)
	}
}

func TestHelperInterruptedCheck(t *testing.T) {
	if os.Getenv("JSDLC_HELPER_INTERRUPTED_CHECK") != "1" {
		return
	}
	_, _ = check([]string{"--repo", os.Getenv("JSDLC_HELPER_REPO"), "--candidate", "candidate-a", "--id", "interrupted", "--domains", "functional", "--authorized", "--", "/bin/sh", "-c", "touch \"$1\"; sleep 30", "sh", os.Getenv("JSDLC_HELPER_MARKER")})
	os.Exit(4)
}

func TestKilledCheckReservationCanBeSafelyRecovered(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "started")
	cmd := exec.Command(executable, "-test.run=^TestHelperInterruptedCheck$")
	cmd.Env = append(os.Environ(), "JSDLC_HELPER_INTERRUPTED_CHECK=1", "JSDLC_HELPER_REPO="+repo, "JSDLC_HELPER_MARKER="+marker)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			t.Fatal("interrupted check did not start")
		}
		time.Sleep(5 * time.Millisecond)
	}
	_, key, root, err := stateLocation(repo)
	if err != nil {
		t.Fatal(err)
	}
	reservationPath := filepath.Join(root, key, "check-reservations", started["run"].(runState).ID, "interrupted.json")
	deadline = time.Now().Add(5 * time.Second)
	var reservation checkReservation
	for {
		reservation, err = readCheckReservation(reservationPath)
		if err == nil && reservation.ProcessGroupID > 0 && processGroupRunning(reservation.ProcessGroupID) {
			break
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			t.Fatalf("check process group identity did not become durable: %#v %v", reservation, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	defer terminateProcessGroup(reservation.ProcessGroupID)
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	if _, err := recoverCheck([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "interrupted", "--authorized"}); err == nil {
		t.Fatal("live command group must prevent recovery")
	}
	if err := terminateProcessGroup(reservation.ProcessGroupID); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for processOrGroupExists(reservation.OwnerPID, reservation.ProcessGroupID) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	got, err := recoverCheck([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "interrupted", "--authorized"})
	if err != nil {
		t.Fatal(err)
	}
	if got["recovered"] != true {
		t.Fatalf("reservation was not recovered: %#v", got)
	}
	if _, err := check([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "interrupted", "--domains", "functional", "--authorized", "--", "/bin/true"}); err != nil {
		t.Fatalf("recovered ID could not be retried: %v", err)
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

func TestPackCatalogParityAndDeterministicMerge(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(workingDir, "..", "..", "plugins", "jerry-sdlc"))
	t.Setenv("JSDLC_PLUGIN_ROOT", root)
	first, err := packs([]string{"--conform-set", "frontend,database"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := packs([]string{"--conform-set", "database,frontend"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first["merged"], second["merged"]) || first["activationAllowed"] != false || first["conformanceOnly"] != true {
		t.Fatalf("pack merge is non-deterministic or activated: %#v %#v", first, second)
	}
	merged := first["merged"].(result)
	if strings.Join(merged["packs"].([]string), ",") != "database,frontend" || merged["conflictFree"] != true {
		t.Fatalf("unexpected inert merge: %#v", merged)
	}
}

func TestPackCatalogRejectsDriftAndValidatesRelations(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(workingDir, "..", "..", "plugins", "jerry-sdlc"))
	b, err := os.ReadFile(filepath.Join(root, "packs", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var base packCatalog
	if err := json.Unmarshal(b, &base); err != nil {
		t.Fatal(err)
	}
	clone := func() packCatalog {
		encoded, _ := json.Marshal(base)
		var copied packCatalog
		if err := json.Unmarshal(encoded, &copied); err != nil {
			t.Fatal(err)
		}
		return copied
	}
	for name, mutate := range map[string]func(*packCatalog){
		"gate":          func(c *packCatalog) { c.Packs[0].Status = "ACTIVE" },
		"policy digest": func(c *packCatalog) { c.Packs[0].PolicyDigest = strings.Repeat("0", 64) },
		"lens digest":   func(c *packCatalog) { c.Lenses[0].PolicyDigest = strings.Repeat("0", 64) },
		"unknown lens":  func(c *packCatalog) { c.Packs[0].Lenses = []string{"unknown"} },
		"duplicate ID":  func(c *packCatalog) { c.Packs[1].ID = c.Packs[0].ID },
		"dangling dep":  func(c *packCatalog) { c.Packs[0].Dependencies = []string{"unknown"} },
		"self conflict": func(c *packCatalog) { c.Packs[0].Conflicts = []string{c.Packs[0].ID} },
	} {
		t.Run(name, func(t *testing.T) {
			changed := clone()
			mutate(&changed)
			if err := validatePackCatalog(root, changed); err == nil {
				t.Fatal("invalid pack catalog was accepted")
			}
		})
	}
	dependent := clone()
	dependent.Packs[0].Dependencies = []string{"backend"}
	if err := validatePackCatalog(root, dependent); err != nil {
		t.Fatal(err)
	}
	merged, _, err := conformPackSet(dependent, "frontend")
	if err != nil || strings.Join(merged["packs"].([]string), ",") != "backend,frontend" {
		t.Fatalf("dependency closure is not deterministic: %#v %v", merged, err)
	}
	cyclic := clone()
	cyclic.Packs[0].Dependencies, cyclic.Packs[1].Dependencies = []string{"backend"}, []string{"frontend"}
	if err := validatePackCatalog(root, cyclic); err == nil {
		t.Fatal("dependency cycle was accepted")
	}
	conflicting := clone()
	conflicting.Packs[0].Conflicts, conflicting.Packs[1].Conflicts = []string{"backend"}, []string{"frontend"}
	if err := validatePackCatalog(root, conflicting); err != nil {
		t.Fatal(err)
	}
	merged, conflicts, err := conformPackSet(conflicting, "backend,frontend")
	if err != nil || merged["conflictFree"] != false || strings.Join(conflicts, ",") != "backend:frontend" {
		t.Fatalf("conflict was not reported deterministically: %#v %#v %v", merged, conflicts, err)
	}
}

func TestPluginMetadataReferencesStayInRootAndExist(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(workingDir, "..", "..", "plugins", "jerry-sdlc"))
	if err := validatePluginJSONReferences(root); err != nil {
		t.Fatal(err)
	}
	for name, reference := range map[string]string{"escape": "../../outside.json", "dangling": "missing.json"} {
		t.Run(name, func(t *testing.T) {
			if err := validateJSONReferences(root, filepath.Join(root, "schemas"), map[string]any{"$ref": reference}); err == nil {
				t.Fatal("invalid metadata reference was accepted")
			}
		})
	}
}

func TestAdapterProtocolValidationCannotUpgradeAssurance(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	pluginRoot := filepath.Clean(filepath.Join(workingDir, "..", "..", "plugins", "jerry-sdlc"))
	t.Setenv("JSDLC_PLUGIN_ROOT", pluginRoot)
	policy, err := os.ReadFile(filepath.Join(pluginRoot, "workflows", "release-readiness.json"))
	if err != nil {
		t.Fatal(err)
	}
	role, err := os.ReadFile(filepath.Join(pluginRoot, "roles", "qa-executor.md"))
	if err != nil {
		t.Fatal(err)
	}
	schemaBytes, err := os.ReadFile(filepath.Join(pluginRoot, "schemas", "worker-result.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	resultBytes := []byte(`{"disposition":"CLEAN","evidence":["observed"],"findings":[],"limitations":[],"domains":[]}`)
	assignmentBytes := []byte("execute the bounded QA assignment")
	resultPath, assignmentPath := filepath.Join(t.TempDir(), "result.json"), filepath.Join(t.TempDir(), "assignment.txt")
	if err := os.WriteFile(resultPath, resultBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assignmentPath, assignmentBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	policyHash, roleHash, schemaHash := sha256.Sum256(policy), sha256.Sum256(role), sha256.Sum256(schemaBytes)
	resultHash, assignmentHash := sha256.Sum256(resultBytes), sha256.Sum256(assignmentBytes)
	contractHasher := sha256.New()
	_, _ = contractHasher.Write(policy)
	_, _ = contractHasher.Write(schemaBytes)
	_, _ = contractHasher.Write([]byte("qa-executor\x00"))
	_, _ = contractHasher.Write(role)
	record := adapterRecord{SchemaVersion: 2, ProtocolVersion: 2, AdapterName: "generic-test", RunID: "run", Candidate: "candidate", Workflow: "release-readiness", WorkflowVersion: 1, Role: "qa-executor", AssignmentID: "qa-execution", AssignmentDigest: fmt.Sprintf("%x", assignmentHash), RepositoryDigest: strings.Repeat("a", 64), PolicyDigest: fmt.Sprintf("%x", policyHash), RoleContractDigest: fmt.Sprintf("%x", roleHash), ContractSetDigest: fmt.Sprintf("%x", contractHasher.Sum(nil)), SchemaDigest: fmt.Sprintf("%x", schemaHash), WorkerInstanceID: "worker", ResultDigest: fmt.Sprintf("%x", resultHash), Mode: "READ_ONLY", Capabilities: adapterCapabilities{EphemeralWorkersObserved: true, StructuredOutputObserved: true}, Trust: "OBSERVED_UNATTESTED", RecordedAt: "2026-01-01T00:00:00Z"}
	record.ReplayID = adapterReplayID(record)
	validRecord := record
	b, _ := json.Marshal(record)
	path := filepath.Join(t.TempDir(), "adapter.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	validate := func() (result, error) {
		return validateAdapter([]string{"--file", path, "--result", resultPath, "--assignment", assignmentPath})
	}
	got, err := validate()
	if err != nil {
		t.Fatal(err)
	}
	if got["valid"] != true || got["activationAllowed"] != false || got["assuranceEffect"] != "EVIDENCE_ONLY" || got["provenance"] != "CALLER_SUPPLIED_UNATTESTED_BYTES" || got["replayId"] != record.ReplayID {
		t.Fatalf("adapter protocol overstated assurance: %#v", got)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	raw["unknown"] = true
	unknown, _ := json.Marshal(raw)
	if err := os.WriteFile(path, unknown, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validate(); err == nil {
		t.Fatal("adapter record with unknown field was accepted")
	}
	if err := os.WriteFile(path, append(b, []byte(" {}")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validate(); err == nil {
		t.Fatal("adapter record with trailing JSON was accepted")
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, []byte(`{"disposition":"CLEAN","evidence":[],"findings":[],"limitations":[],"domains":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validate(); err == nil {
		t.Fatal("invalid worker result was accepted")
	}
	if err := os.WriteFile(resultPath, resultBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	resultLink := filepath.Join(t.TempDir(), "result-link.json")
	if err := os.Symlink(resultPath, resultLink); err != nil {
		t.Fatal(err)
	}
	if _, err := validateAdapter([]string{"--file", path, "--result", resultLink, "--assignment", assignmentPath}); err == nil {
		t.Fatal("symlinked worker result was accepted")
	}
	oversizedAssignment := filepath.Join(t.TempDir(), "oversized-assignment")
	if err := os.WriteFile(oversizedAssignment, make([]byte, 256*1024+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validateAdapter([]string{"--file", path, "--result", resultPath, "--assignment", oversizedAssignment}); err == nil {
		t.Fatal("oversized assignment was accepted")
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	delete(raw, "capabilities")
	omitted, _ := json.Marshal(raw)
	if err := os.WriteFile(path, omitted, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validate(); err == nil {
		t.Fatal("adapter record omitted capabilities")
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	capabilities := raw["capabilities"].(map[string]any)
	for key := range capabilities {
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}
		delete(raw["capabilities"].(map[string]any), key)
		omitted, _ = json.Marshal(raw)
		if err := os.WriteFile(path, omitted, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := validate(); err == nil {
			t.Fatalf("adapter record omitted capability %s", key)
		}
	}
	record.Capabilities.WriteIsolationAttested = true
	b, _ = json.Marshal(record)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validate(); err == nil {
		t.Fatal("self-reported adapter claimed attested isolation")
	}

	// Restore the valid record, then prove each supplied byte stream is bound.
	record.Capabilities.WriteIsolationAttested = false
	b, _ = json.Marshal(record)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, append(resultBytes, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validate(); err == nil {
		t.Fatal("tampered result bytes were accepted")
	}
	if err := os.WriteFile(resultPath, resultBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assignmentPath, append(assignmentBytes, '.'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validate(); err == nil {
		t.Fatal("tampered assignment bytes were accepted")
	}
	if err := os.WriteFile(assignmentPath, assignmentBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	record.ReplayID = strings.Repeat("f", 64)
	b, _ = json.Marshal(record)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validate(); err == nil {
		t.Fatal("incorrect replay identity was accepted")
	}
	for name, mutate := range map[string]func(*adapterRecord){
		"policy":       func(r *adapterRecord) { r.PolicyDigest = strings.Repeat("0", 64) },
		"role":         func(r *adapterRecord) { r.RoleContractDigest = strings.Repeat("0", 64) },
		"contract set": func(r *adapterRecord) { r.ContractSetDigest = strings.Repeat("0", 64) },
		"schema":       func(r *adapterRecord) { r.SchemaDigest = strings.Repeat("0", 64) },
		"workflow":     func(r *adapterRecord) { r.WorkflowVersion = 2 },
		"assignment":   func(r *adapterRecord) { r.AssignmentID = "qa-architecture" },
	} {
		t.Run("reject "+name+" drift", func(t *testing.T) {
			changed := validRecord
			mutate(&changed)
			changed.ReplayID = adapterReplayID(changed)
			changedBytes, _ := json.Marshal(changed)
			if err := os.WriteFile(path, changedBytes, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := validate(); err == nil {
				t.Fatal("drifted adapter contract was accepted")
			}
		})
	}
}

func TestCollisionResolutionPrecedenceAndNoLaunch(t *testing.T) {
	writeInput := func(t *testing.T, input collisionInput) string {
		t.Helper()
		b, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "collision.json")
		if err := os.WriteFile(path, b, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	base := collisionInput{SchemaVersion: 1, ExplicitSelection: "NONE", CompetingBroadOrchestrators: []string{}, DiscoveryEvidence: "RUNTIME_OBSERVED"}
	tests := []struct {
		name      string
		input     collisionInput
		decision  string
		assurance string
		allowed   bool
	}{
		{"no collision", base, "PROCEED_JERRY", "UNCHANGED", true},
		{"explicit Jerry wins", collisionInput{SchemaVersion: 1, ExplicitSelection: "JERRY", ActiveJerryRunID: "old", RequestedJerryRunID: "new", CompetingBroadOrchestrators: []string{"another-agent"}, DiscoveryEvidence: "CALLER_DECLARED"}, "PROCEED_JERRY", "UNCHANGED", true},
		{"explicit other wins", collisionInput{SchemaVersion: 1, ExplicitSelection: "OTHER", ActiveJerryRunID: "run-a", RequestedJerryRunID: "run-a", CompetingBroadOrchestrators: []string{}, DiscoveryEvidence: "RUNTIME_OBSERVED"}, "YIELD_TO_EXPLICIT_SELECTION", "UNCHANGED", false},
		{"same run resumes", collisionInput{SchemaVersion: 1, ExplicitSelection: "NONE", ActiveJerryRunID: "run-a", RequestedJerryRunID: "run-a", CompetingBroadOrchestrators: []string{"another-agent"}, DiscoveryEvidence: "RUNTIME_OBSERVED"}, "RESUME_JERRY", "UNCHANGED", true},
		{"different Jerry run blocks", collisionInput{SchemaVersion: 1, ExplicitSelection: "NONE", ActiveJerryRunID: "run-a", RequestedJerryRunID: "run-b", CompetingBroadOrchestrators: []string{}, DiscoveryEvidence: "RUNTIME_OBSERVED"}, "COLLISION", "OWNER_GATE", false},
		{"competitor blocks", collisionInput{SchemaVersion: 1, ExplicitSelection: "NONE", CompetingBroadOrchestrators: []string{"another-agent"}, DiscoveryEvidence: "CALLER_DECLARED"}, "COLLISION", "OWNER_GATE", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveCollision([]string{"--file", writeInput(t, tc.input)})
			if err != nil {
				t.Fatal(err)
			}
			if got["decision"] != tc.decision || got["assurance"] != tc.assurance || got["launchAllowed"] != tc.allowed || got["launchPerformed"] != false || got["runtimeDiscoveryRule"] != "CALLER_SUPPLIED_EVIDENCE_ONLY" {
				t.Fatalf("unexpected collision decision: %#v", got)
			}
			if !validSHA256(got["inputDigest"].(string)) {
				t.Fatalf("missing input binding: %#v", got)
			}
		})
	}
}

func TestCollisionInputFailsClosed(t *testing.T) {
	write := func(content string) string {
		path := filepath.Join(t.TempDir(), "collision.json")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	invalid := []string{
		`{"schemaVersion":1,"explicitSelection":"NONE","competingBroadOrchestrators":[],"discoveryEvidence":"RUNTIME_OBSERVED","unknown":true}`,
		`{"schemaVersion":1,"explicitSelection":"NONE","activeJerryRunId":"run","competingBroadOrchestrators":[],"discoveryEvidence":"RUNTIME_OBSERVED"}`,
		`{"schemaVersion":1,"explicitSelection":"NONE","competingBroadOrchestrators":["Other Agent"," other agent "],"discoveryEvidence":"RUNTIME_OBSERVED"}`,
		`{"schemaVersion":1,"explicitSelection":"NONE","competingBroadOrchestrators":["jsdlc"],"discoveryEvidence":"RUNTIME_OBSERVED"}`,
		`{"schemaVersion":1,"explicitSelection":"NONE","competingBroadOrchestrators":null,"discoveryEvidence":"RUNTIME_OBSERVED"}`,
		`{"schemaVersion":1,"explicitSelection":"NONE","competingBroadOrchestrators":[],"discoveryEvidence":"AUTOMATIC"}`,
		`{"schemaVersion":1,"explicitSelection":"NONE","competingBroadOrchestrators":[],"discoveryEvidence":"RUNTIME_OBSERVED"} {}`,
	}
	for _, content := range invalid {
		if _, err := resolveCollision([]string{"--file", write(content)}); err == nil {
			t.Fatalf("invalid collision input was accepted: %s", content)
		}
	}
}

func TestShippedWorkflowContractExactlyMatchesRuntime(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	pluginRoot := filepath.Clean(filepath.Join(workingDir, "..", "..", "plugins", "jerry-sdlc"))
	t.Setenv("JSDLC_PLUGIN_ROOT", pluginRoot)
	got, err := validateWorkflow(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["valid"] != true || got["runtimeParity"] != "EXACT" || got["activationAllowed"] != false || !validSHA256(got["contractDigest"].(string)) {
		t.Fatalf("unexpected workflow validation: %#v", got)
	}
	schemaBytes, err := os.ReadFile(filepath.Join(pluginRoot, "schemas", "workflow-contract.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Required   []string `json:"required"`
		Properties map[string]struct {
			Const json.RawMessage `json:"const"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatal(err)
	}
	assignmentProperty, declared := schema.Properties["specialistAssignments"]
	var declaredAssignments []string
	if err := json.Unmarshal(assignmentProperty.Const, &declaredAssignments); err != nil {
		t.Fatal(err)
	}
	if !declared || !contains(schema.Required, "specialistAssignments") || !equalStrings(declaredAssignments, specialistAssignments) {
		t.Fatalf("shipped workflow schema drifted from specialist assignments: %#v", schema)
	}
}

func TestWorkflowContractDriftFailsClosed(t *testing.T) {
	base := workflowContract{
		SchemaVersion: 1, Name: "release-readiness",
		TerminalStates:        append([]string(nil), releaseWorkflowTerminalStates...),
		Roles:                 append([]string(nil), releaseWorkflowRoles...),
		SpecialistAssignments: append([]string(nil), specialistAssignments...),
		States:                append([]string(nil), releaseWorkflowStates...),
		Transitions:           map[string][]string{},
		RequiredDomains:       append([]string(nil), requiredReleaseDomains...),
		ForbiddenActions:      append([]string(nil), releaseWorkflowForbiddenActions...),
	}
	for state, next := range allowedTransitions {
		base.Transitions[state] = append([]string(nil), next...)
	}
	write := func(t *testing.T, value any) string {
		t.Helper()
		b, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "workflow.json")
		if err := os.WriteFile(path, b, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	mutations := map[string]func(*workflowContract){
		"schema":         func(c *workflowContract) { c.SchemaVersion = 2 },
		"name":           func(c *workflowContract) { c.Name = "feature" },
		"terminal state": func(c *workflowContract) { c.TerminalStates[0] = "READY" },
		"role":           func(c *workflowContract) { c.Roles = c.Roles[:len(c.Roles)-1] },
		"specialist": func(c *workflowContract) {
			c.SpecialistAssignments = c.SpecialistAssignments[:len(c.SpecialistAssignments)-1]
		},
		"state":             func(c *workflowContract) { c.States[0] = "NEW" },
		"transition source": func(c *workflowContract) { delete(c.Transitions, "BASELINED") },
		"transition target": func(c *workflowContract) { c.Transitions["VERIFIED"] = append(c.Transitions["VERIFIED"], "READY") },
		"domain":            func(c *workflowContract) { c.RequiredDomains[0] = "invented" },
		"forbidden action":  func(c *workflowContract) { c.ForbiddenActions = c.ForbiddenActions[:len(c.ForbiddenActions)-1] },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			b, _ := json.Marshal(base)
			var changed workflowContract
			if err := json.Unmarshal(b, &changed); err != nil {
				t.Fatal(err)
			}
			mutate(&changed)
			if _, err := validateWorkflow([]string{"--file", write(t, changed)}); err == nil {
				t.Fatal("drifted workflow contract was accepted")
			}
		})
	}
	var raw map[string]any
	b, _ := json.Marshal(base)
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	raw["unknown"] = true
	if _, err := validateWorkflow([]string{"--file", write(t, raw)}); err == nil {
		t.Fatal("unknown workflow field was accepted")
	}
	path := write(t, base)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(content, []byte(" {}")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validateWorkflow([]string{"--file", path}); err == nil {
		t.Fatal("trailing workflow JSON was accepted")
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
	got, err := worker([]string{"--repo", repo, "--candidate", "candidate-a", "--role", "specialist-reviewer", "--assignment", "specialist-security", "--prompt-file", prompt})
	if err != nil {
		t.Fatal(err)
	}
	receipt := got["receipt"].(workerReceipt)
	if receipt.RunID != started["run"].(runState).ID || receipt.Candidate != "candidate-a" || receipt.AssignmentID != "specialist-security" || receipt.ThreadID != "thread-worker-a" || len(receipt.SchemaDigest) != 64 || len(receipt.ReportDigest) != 64 || got["assuranceEffect"] != "EVIDENCE_ONLY" {
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
	if got["assurance"] != "MANAGED_SEPARATE_PASSES" || got["workerObservation"] != "OBSERVED_DISTINCT_SUBPROCESSES" || got["scope"] != "PERSISTED_CANDIDATE_BOUND_EVIDENCE" || got["persistence"] != "REPORTS_AND_RECEIPTS" || got["verdict"] != "INCONCLUSIVE" {
		t.Fatalf("unexpected team result: %#v", got)
	}
	if len(got["roles"].([]result)) != 10 {
		t.Fatalf("expected ten bounded assignment results: %#v", got)
	}
	if digest, ok := got["contractSetDigest"].(string); !ok || len(digest) != 64 {
		t.Fatalf("missing frozen contract-set digest: %#v", got)
	}
	verified, err := verify([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if verified["verdict"] != "INCONCLUSIVE" || verified["reproduced"] != true {
		t.Fatalf("persisted verdict did not reproduce: %#v", verified)
	}
}

func TestTeamLocalCheckedEvidencePersistsButCannotClaimReadiness(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("JSDLC_PLUGIN_ROOT", filepath.Clean(filepath.Join(workingDir, "..", "..", "plugins", "jerry-sdlc")))
	domains := make([]domainResult, 0, len(requiredReleaseDomains))
	for _, domain := range requiredReleaseDomains {
		domains = append(domains, domainResult{Domain: domain, Status: "PASS", Evidence: "checked command and repository evidence", EvidenceIDs: []string{"all-checks"}})
	}
	domains[0] = domainResult{Domain: requiredReleaseDomains[0], Status: "NOT_APPLICABLE", Evidence: "explicit applicability review", EvidenceIDs: []string{"all-checks"}}
	reportBytes, err := json.Marshal(workerReport{Disposition: "CLEAN", Evidence: []string{"checked"}, Findings: []workerFinding{}, Limitations: []string{}, Domains: domains})
	if err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	fakeCodex := filepath.Join(binDir, "codex")
	script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'codex-test 1.0'; exit 0; fi\ncat >/dev/null\nprintf '%%s\\n' \"{\\\"type\\\":\\\"thread.started\\\",\\\"thread_id\\\":\\\"thread-$$\\\"}\" '%s'\n", `{"type":"item.completed","item":{"type":"agent_message","text":`+strconv.Quote(string(reportBytes))+`}}`)
	if err := os.WriteFile(fakeCodex, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	repo := t.TempDir()
	if _, err := start([]string{"--repo", repo, "--candidate", "candidate-a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := check([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "all-checks", "--domains", strings.Join(requiredReleaseDomains, ","), "--authorized", "--", "/bin/true"}); err != nil {
		t.Fatal(err)
	}
	got, err := team([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if got["verdict"] != "INCONCLUSIVE" {
		t.Fatalf("unattested local evidence claimed readiness: %#v", got)
	}
	verified, err := verify([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if verified["verdict"] != "INCONCLUSIVE" || verified["reproduced"] != true {
		t.Fatalf("clean persisted result did not reproduce: %#v", verified)
	}
	decisionInput := adjudicationInput{SchemaVersion: 1, TeamDigest: got["evidenceDigest"].(string), Decisions: []adjudicationDecision{{Kind: "DOMAIN_NOT_APPLICABLE", ID: requiredReleaseDomains[0], Disposition: "ACCEPTED", Rationale: "the checked candidate has no applicable functional surface", EvidenceIDs: []string{"all-checks"}}}}
	decisionBytes, err := json.Marshal(decisionInput)
	if err != nil {
		t.Fatal(err)
	}
	decisionPath := filepath.Join(t.TempDir(), "adjudication.json")
	if err := os.WriteFile(decisionPath, decisionBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	adjudicated, err := adjudicate([]string{"--repo", repo, "--candidate", "candidate-a", "--file", decisionPath, "--authorized"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adjudicate([]string{"--repo", repo, "--candidate", "candidate-a", "--file", decisionPath, "--authorized"}); err == nil {
		t.Fatal("team evidence must not be adjudicated twice")
	}
	verified, err = verify([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if verified["verdict"] != "INCONCLUSIVE" || verified["adjudicated"] != true {
		t.Fatalf("adjudicated result did not reproduce: %#v", verified)
	}
	adjudicationPath := adjudicated["path"].(string)
	originalAdjudication, err := os.ReadFile(adjudicationPath)
	if err != nil {
		t.Fatal(err)
	}
	tamperedAdjudication := bytes.Replace(originalAdjudication, []byte("the checked candidate"), []byte("an edited candidate"), 1)
	if bytes.Equal(tamperedAdjudication, originalAdjudication) {
		t.Fatal("test failed to alter adjudication evidence")
	}
	if err := os.WriteFile(adjudicationPath, tamperedAdjudication, 0o600); err != nil {
		t.Fatal(err)
	}
	tamperedDigest := fmt.Sprintf("%x", sha256.Sum256(tamperedAdjudication))
	tamperedPath := filepath.Join(filepath.Dir(adjudicationPath), tamperedDigest+".json")
	if err := os.Rename(adjudicationPath, tamperedPath); err != nil {
		t.Fatal(err)
	}
	verified, err = verify([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if verified["verdict"] != "BLOCKED" || !strings.Contains(verified["reason"].(string), "authorization anchor") {
		t.Fatalf("edited and rehashed adjudication evidence must block: %#v", verified)
	}
	// LOCAL_USER_AUTHORIZED deliberately does not claim integrity against the
	// same OS user. That user can replace both evidence and its local anchor;
	// a trusted adapter or remote log is required to close this boundary.
	anchorPath := filepath.Join(filepath.Dir(filepath.Dir(adjudicationPath)), got["evidenceDigest"].(string)+".digest")
	if err := os.Remove(anchorPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(anchorPath, []byte(tamperedDigest+"\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	verified, err = verify([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if verified["verdict"] != "INCONCLUSIVE" || verified["adjudicated"] != true {
		t.Fatalf("local-owner rewrite limitation changed; revisit the documented trust boundary: %#v", verified)
	}
	if err := os.Rename(tamperedPath, adjudicationPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adjudicationPath, originalAdjudication, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(anchorPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(anchorPath, []byte(adjudicated["digest"].(string)+"\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	if _, err := check([]string{"--repo", repo, "--candidate", "candidate-a", "--id", "late-check", "--domains", "functional", "--authorized", "--", "/bin/true"}); err != nil {
		t.Fatal(err)
	}
	verified, err = verify([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	if verified["verdict"] != "BLOCKED" || !strings.Contains(verified["reason"].(string), "changed after") {
		t.Fatalf("post-assessment evidence change must block: %#v", verified)
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

func completeAssignedOutputs(architect, executor, specialist, verifier json.RawMessage) []result {
	outputs := []result{
		{"role": "qa-architect", "assignmentId": "qa-architecture", "report": architect},
		{"role": "qa-executor", "assignmentId": "qa-execution", "report": executor},
	}
	for _, assignmentID := range specialistAssignments {
		domain := specialistAssignmentDomains[assignmentID]
		var base workerReport
		_ = json.Unmarshal(specialist, &base)
		base.Domains = []domainResult{{Domain: domain, Status: "BLOCKED", Evidence: "specialist lens not supplied", EvidenceIDs: []string{}}}
		if len(specialist) > 0 {
			var supplied workerReport
			if json.Unmarshal(specialist, &supplied) == nil {
				for _, candidate := range supplied.Domains {
					if candidate.Domain == domain {
						base.Domains = []domainResult{candidate}
					}
				}
			}
		}
		raw, _ := json.Marshal(base)
		outputs = append(outputs, result{"role": "specialist-reviewer", "assignmentId": assignmentID, "report": json.RawMessage(raw)})
	}
	return append(outputs, result{"role": "independent-verifier", "assignmentId": "independent-verification", "report": verifier})
}

func validAssignmentBundle(t *testing.T) teamEvidence {
	t.Helper()
	bundle := teamEvidence{SchemaVersion: 2, RunID: "run", Repository: "/repo", Candidate: "candidate", RepositoryDigest: strings.Repeat("a", 64), ContractSetDigest: strings.Repeat("b", 64), ChecksDigest: strings.Repeat("c", 64), Workflow: "release-readiness", Assurance: "MANAGED_SEPARATE_PASSES", Verdict: "INCONCLUSIVE", Reason: "local passes cannot establish independence", CompletedAt: "2026-01-01T00:00:00Z"}
	for index, assignment := range releaseTeamAssignments() {
		domains := []domainResult{}
		if assignment.lens != "" {
			domains = []domainResult{{Domain: assignment.lens, Status: "BLOCKED", Evidence: "no attested check", EvidenceIDs: []string{}}}
		}
		report, err := json.Marshal(workerReport{Disposition: "CLEAN", Evidence: []string{"reviewed"}, Findings: []workerFinding{}, Limitations: []string{}, Domains: domains})
		if err != nil {
			t.Fatal(err)
		}
		reportDigest, err := canonicalJSONDigest(report)
		if err != nil {
			t.Fatal(err)
		}
		receipt := workerReceipt{SchemaVersion: 2, RunID: bundle.RunID, Repository: bundle.Repository, Candidate: bundle.Candidate, RepositoryDigest: bundle.RepositoryDigest, Role: assignment.role, AssignmentID: assignment.id, RoleContractDigest: strings.Repeat("d", 64), WorkflowDigest: strings.Repeat("e", 64), SchemaDigest: strings.Repeat("f", 64), ThreadID: fmt.Sprintf("thread-%d", index), SandboxModeRequested: "read-only", CodexVersion: "codex-test", PromptDigest: strings.Repeat("1", 64), OutputDigest: strings.Repeat("2", 64), ReportDigest: reportDigest, StartedAt: "2026-01-01T00:00:00Z", CompletedAt: "2026-01-01T00:00:01Z", Command: []string{"codex"}, ExitStatus: 0}
		bundle.Roles = append(bundle.Roles, teamRoleEvidence{Role: assignment.role, AssignmentID: assignment.id, Receipt: receipt, Report: report})
	}
	return bundle
}

func TestSpecialistAssignmentEvidenceFailsClosed(t *testing.T) {
	if err := validateTeamEvidence(validAssignmentBundle(t)); err != nil {
		t.Fatalf("valid assignment bundle failed: %v", err)
	}
	for name, mutate := range map[string]func(*teamEvidence){
		"missing": func(bundle *teamEvidence) { bundle.Roles = bundle.Roles[:len(bundle.Roles)-1] },
		"duplicate": func(bundle *teamEvidence) {
			bundle.Roles[3].AssignmentID, bundle.Roles[3].Receipt.AssignmentID = bundle.Roles[2].AssignmentID, bundle.Roles[2].AssignmentID
		},
		"reused worker":   func(bundle *teamEvidence) { bundle.Roles[3].Receipt.ThreadID = bundle.Roles[2].Receipt.ThreadID },
		"stale candidate": func(bundle *teamEvidence) { bundle.Roles[3].Receipt.Candidate = "older-candidate" },
		"tampered report": func(bundle *teamEvidence) { bundle.Roles[3].Report = bundle.Roles[2].Report },
	} {
		t.Run(name, func(t *testing.T) {
			bundle := validAssignmentBundle(t)
			mutate(&bundle)
			if err := validateTeamEvidence(bundle); err == nil {
				t.Fatal("invalid assignment evidence was accepted")
			}
		})
	}
}

func TestAggregateTeamVerdictRequiresEveryDomain(t *testing.T) {
	domains := make([]domainResult, 0, len(requiredReleaseDomains))
	checks := map[string]checkEvidence{}
	for _, domain := range requiredReleaseDomains {
		id := "check-" + domain
		domains = append(domains, domainResult{Domain: domain, Status: "PASS", Evidence: "verified", EvidenceIDs: []string{id}})
		checks[id] = checkEvidence{ID: id, Domains: []string{domain}, ExitStatus: 0, Trust: "ATTESTED_RUNTIME"}
	}
	report, err := json.Marshal(workerReport{Disposition: "CLEAN", Evidence: []string{"checked"}, Findings: []workerFinding{}, Limitations: []string{}, Domains: domains})
	if err != nil {
		t.Fatal(err)
	}
	fullReport := append(json.RawMessage(nil), report...)
	cleanEmpty, err := json.Marshal(workerReport{Disposition: "CLEAN", Evidence: []string{"checked"}, Findings: []workerFinding{}, Limitations: []string{}, Domains: []domainResult{}})
	if err != nil {
		t.Fatal(err)
	}
	outputs := completeAssignedOutputs(json.RawMessage(cleanEmpty), json.RawMessage(cleanEmpty), json.RawMessage(report), json.RawMessage(report))
	verdict, _ := aggregateTeamVerdict(outputs, checks, true)
	if verdict != "READY" {
		t.Fatalf("expected READY, got %s", verdict)
	}
	if verdict, _ := aggregateTeamVerdict(outputs, checks, false); verdict != "INCONCLUSIVE" {
		t.Fatalf("unattested execution path must not accept even synthetic attested records, got %s", verdict)
	}
	report, err = json.Marshal(workerReport{Disposition: "CLEAN", Evidence: []string{"checked"}, Findings: []workerFinding{}, Limitations: []string{}, Domains: domains[:len(domains)-1]})
	if err != nil {
		t.Fatal(err)
	}
	outputs[len(outputs)-1]["report"] = json.RawMessage(report)
	verdict, _ = aggregateTeamVerdict(outputs, checks, true)
	if verdict != "INCONCLUSIVE" {
		t.Fatalf("missing domain must be inconclusive, got %s", verdict)
	}
	verdict, _ = aggregateTeamVerdict(outputs[:len(outputs)-1], checks, true)
	if verdict != "INCONCLUSIVE" {
		t.Fatalf("missing role must be inconclusive, got %s", verdict)
	}
	missingSpecialist := append([]result{}, outputs[:2]...)
	missingSpecialist = append(missingSpecialist, outputs[3:]...)
	if verdict, _ := aggregateTeamVerdict(missingSpecialist, checks, true); verdict != "INCONCLUSIVE" {
		t.Fatalf("missing specialist lens must be inconclusive, got %s", verdict)
	}
	securityNA, _ := json.Marshal(workerReport{Disposition: "CLEAN", Evidence: []string{"checked"}, Findings: []workerFinding{}, Limitations: []string{}, Domains: []domainResult{{Domain: "security", Status: "NOT_APPLICABLE", Evidence: "no security surface", EvidenceIDs: []string{"check-security"}}}})
	outputs = completeAssignedOutputs(json.RawMessage(cleanEmpty), json.RawMessage(cleanEmpty), fullReport, fullReport)
	outputs[2]["report"] = json.RawMessage(securityNA)
	adjudication := &adjudicationEvidence{Decisions: []adjudicationDecision{{Kind: "DOMAIN_NOT_APPLICABLE", ID: "security", Disposition: "ACCEPTED", Rationale: "the checked candidate has no security surface", EvidenceIDs: []string{"check-security"}}}}
	if verdict, _ := aggregateTeamVerdictWithAdjudication(outputs, checks, true, adjudication); verdict != "READY" {
		t.Fatalf("evidence-backed specialist N/A should aggregate, got %s", verdict)
	}
}

func TestAggregateTeamVerdictFailsClosed(t *testing.T) {
	checks := map[string]checkEvidence{}
	clean := func(disposition string, domains []domainResult) json.RawMessage {
		if domains == nil {
			domains = []domainResult{}
		}
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
		id := "check-" + domain
		passes = append(passes, domainResult{Domain: domain, Status: "PASS", Evidence: "verified", EvidenceIDs: []string{id}})
		checks[id] = checkEvidence{ID: id, Domains: []string{domain}, ExitStatus: 0, Trust: "ATTESTED_RUNTIME"}
	}
	base := func() []result {
		return completeAssignedOutputs(clean("CLEAN", nil), clean("CLEAN", nil), clean("CLEAN", passes), clean("CLEAN", passes))
	}
	duplicate := base()
	duplicate[3]["assignmentId"] = duplicate[2]["assignmentId"]
	finding := base()
	finding[1] = result{"role": "qa-executor", "assignmentId": "qa-execution", "report": clean("FINDINGS", nil)}
	blocked := base()
	blocked[1] = result{"role": "qa-executor", "assignmentId": "qa-execution", "report": clean("BLOCKED", nil)}
	naDomains := append([]domainResult{}, passes...)
	naDomains[0] = domainResult{Domain: requiredReleaseDomains[0], Status: "NOT_APPLICABLE", Evidence: "claimed n/a", EvidenceIDs: []string{}}
	notApplicable := base()
	notApplicable[len(notApplicable)-1] = result{"role": "independent-verifier", "assignmentId": "independent-verification", "report": clean("CLEAN", naDomains)}
	for name, tc := range map[string]struct {
		outputs []result
		want    string
	}{"duplicate": {duplicate, "INCONCLUSIVE"}, "finding": {finding, "NOT_READY"}, "blocked": {blocked, "INCONCLUSIVE"}, "not-applicable": {notApplicable, "INCONCLUSIVE"}} {
		if got, _ := aggregateTeamVerdict(tc.outputs, checks, true); got != tc.want {
			t.Fatalf("%s: got %s want %s", name, got, tc.want)
		}
	}
}

func TestAdjudicationControlsFindingsAndNotApplicable(t *testing.T) {
	checks := map[string]checkEvidence{}
	passes := make([]domainResult, 0, len(requiredReleaseDomains))
	for _, domain := range requiredReleaseDomains {
		id := "check-" + domain
		passes = append(passes, domainResult{Domain: domain, Status: "PASS", Evidence: "verified", EvidenceIDs: []string{id}})
		checks[id] = checkEvidence{ID: id, Domains: []string{domain}, ExitStatus: 0, Trust: "ATTESTED_RUNTIME"}
	}
	passes[0].Status = "NOT_APPLICABLE"
	clean := func(disposition string, findings []workerFinding, domains []domainResult) json.RawMessage {
		if domains == nil {
			domains = []domainResult{}
		}
		b, err := json.Marshal(workerReport{Disposition: disposition, Evidence: []string{"checked"}, Findings: findings, Limitations: []string{}, Domains: domains})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	finding := workerFinding{ID: "F-1", Severity: "HIGH", Confidence: "HIGH", Requirement: "safe", Location: "x", Evidence: "broken", Recommendation: "fix"}
	outputs := completeAssignedOutputs(clean("CLEAN", []workerFinding{}, nil), clean("FINDINGS", []workerFinding{finding}, nil), clean("CLEAN", []workerFinding{}, passes), clean("CLEAN", []workerFinding{}, passes))
	acceptedNA := adjudicationDecision{Kind: "DOMAIN_NOT_APPLICABLE", ID: requiredReleaseDomains[0], Disposition: "ACCEPTED", Rationale: "not present", EvidenceIDs: []string{"check-" + requiredReleaseDomains[0]}}
	rejectedFinding := adjudicationDecision{Kind: "FINDING", ID: "F-1", Disposition: "REJECTED", Rationale: "contradicted by exact evidence", EvidenceIDs: []string{"check-functional"}}
	adj := &adjudicationEvidence{Decisions: []adjudicationDecision{acceptedNA, rejectedFinding}}
	if verdict, _ := aggregateTeamVerdictWithAdjudication(outputs, checks, true, adj); verdict != "READY" {
		t.Fatalf("rejected finding and accepted N/A should allow clean aggregation, got %s", verdict)
	}
	adj.Decisions[1].Disposition = "ACCEPTED"
	if verdict, _ := aggregateTeamVerdictWithAdjudication(outputs, checks, true, adj); verdict != "NOT_READY" {
		t.Fatalf("accepted finding must remain NOT_READY, got %s", verdict)
	}
	adj.Decisions = adj.Decisions[:1]
	if verdict, _ := aggregateTeamVerdictWithAdjudication(outputs, checks, true, adj); verdict != "NOT_READY" {
		t.Fatalf("unadjudicated finding must remain NOT_READY, got %s", verdict)
	}
}

func TestAdjudicationValidatesFindingIdentityAndRejectionEvidence(t *testing.T) {
	report := func(id string) json.RawMessage {
		b, err := json.Marshal(workerReport{Disposition: "FINDINGS", Evidence: []string{"reviewed"}, Findings: []workerFinding{{ID: id, Severity: "HIGH", Confidence: "HIGH", Requirement: "safe", Location: "x", Evidence: "broken", Recommendation: "fix"}}, Limitations: []string{}, Domains: []domainResult{}})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	checks := map[string]checkEvidence{"check-functional": {ID: "check-functional", ExitStatus: 0}}
	bundle := teamEvidence{Roles: []teamRoleEvidence{{Role: "qa-executor", Report: report("F-1")}}}
	decision := adjudicationDecision{Kind: "FINDING", ID: "F-1", Disposition: "REJECTED", Rationale: "disproved", EvidenceIDs: []string{"check-functional"}}
	if err := validateAdjudicationDecisions([]adjudicationDecision{decision}, bundle, checks); err != nil {
		t.Fatalf("valid rejection was refused: %v", err)
	}
	for name, ids := range map[string][]string{"empty": {}, "unknown": {"missing"}, "duplicate": {"check-functional", "check-functional"}, "invalid": {"Bad ID"}} {
		invalid := decision
		invalid.EvidenceIDs = ids
		if err := validateAdjudicationDecisions([]adjudicationDecision{invalid}, bundle, checks); err == nil {
			t.Fatalf("%s rejection evidence was accepted", name)
		}
	}
	bundle.Roles = append(bundle.Roles, teamRoleEvidence{Role: "specialist-reviewer", Report: report("F-1")})
	if err := validateAdjudicationDecisions([]adjudicationDecision{decision}, bundle, checks); err == nil {
		t.Fatal("finding IDs duplicated across roles were accepted")
	}
}

func TestCorrectionManifestEnforcesExactAndDirectoryScopes(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "a.txt"), []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := repositoryManifest(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "a.txt"), []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "outside.txt"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := repositoryManifest(repo)
	if err != nil {
		t.Fatal(err)
	}
	changed := changedManifestPaths(before, after)
	if len(changed) != 2 || changed[0] != "outside.txt" || changed[1] != "src/a.txt" {
		t.Fatalf("unexpected changed paths: %#v", changed)
	}
	if !pathAllowed("src/a.txt", []string{"src/"}) || pathAllowed("src/a.txt", []string{"src"}) || !pathAllowed("src", []string{"src"}) {
		t.Fatal("allowed-path exact/prefix semantics changed")
	}
	for _, invalid := range []string{"", ".", "../x", "/tmp/x", "src/../x"} {
		if validAllowedPath(invalid) {
			t.Fatalf("invalid correction path accepted: %q", invalid)
		}
	}
	external := t.TempDir()
	if err := os.Symlink(filepath.Join(external, "target"), filepath.Join(repo, "link-file")); err != nil {
		t.Fatal(err)
	}
	if _, err := repositoryManifest(repo); err == nil || !strings.Contains(err.Error(), "symlinks") {
		t.Fatalf("file symlink was accepted: %v", err)
	}
	if err := os.Remove(filepath.Join(repo, "link-file")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(repo, "link-dir")); err != nil {
		t.Fatal(err)
	}
	if _, err := repositoryManifest(repo); err == nil || !strings.Contains(err.Error(), "symlinks") {
		t.Fatalf("directory symlink was accepted: %v", err)
	}
	base := runState{SchemaVersion: 2, ID: "run", Workflow: "release-readiness", State: "BASELINED", Assurance: "MANAGED_SEPARATE_PASSES", Repository: "/repo", Candidate: "candidate", ContentDigest: strings.Repeat("a", 64), CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}
	for _, mutate := range []func(*runState){func(s *runState) { s.CorrectionCycle = 1 }, func(s *runState) { s.ParentRunID = "parent" }, func(s *runState) { s.ParentRunID, s.CorrectionCycle = "parent", 3 }} {
		invalid := base
		mutate(&invalid)
		encoded, _ := json.Marshal(invalid)
		if _, err := decodeRunState(encoded); err == nil {
			t.Fatal("invalid correction lineage was accepted")
		}
	}
}

func TestAuthorizedCorrectionCreatesFreshBoundedRun(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	allowedFile := filepath.Join(repo, "src", "a.txt")
	if err := os.WriteFile(allowedFile, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	started, err := start([]string{"--repo", repo, "--candidate", "candidate-a"})
	if err != nil {
		t.Fatal(err)
	}
	state := started["run"].(runState)
	abs, key, root, err := stateLocation(repo)
	if err != nil {
		t.Fatal(err)
	}
	report := func(role string) json.RawMessage {
		findings, disposition := []workerFinding{}, "CLEAN"
		if role == "qa-executor" {
			disposition = "FINDINGS"
			findings = []workerFinding{{ID: "F-1", Severity: "HIGH", Confidence: "HIGH", Requirement: "correct", Location: "src/a.txt", Evidence: "broken", Recommendation: "fix"}}
		}
		b, marshalErr := json.Marshal(workerReport{Disposition: disposition, Evidence: []string{"reviewed"}, Findings: findings, Limitations: []string{}, Domains: []domainResult{}})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		return b
	}
	roles := []teamRoleEvidence{}
	for index, assignment := range releaseTeamAssignments() {
		role := assignment.role
		raw := report(role)
		reportDigest, digestErr := canonicalJSONDigest(raw)
		if digestErr != nil {
			t.Fatal(digestErr)
		}
		receipt := workerReceipt{SchemaVersion: 2, RunID: state.ID, Repository: abs, Candidate: state.Candidate, RepositoryDigest: state.ContentDigest, Role: role, AssignmentID: assignment.id, RoleContractDigest: strings.Repeat("a", 64), WorkflowDigest: strings.Repeat("b", 64), SchemaDigest: strings.Repeat("c", 64), ThreadID: fmt.Sprintf("thread-%d", index), SandboxModeRequested: "read-only", CodexVersion: "codex-test", PromptDigest: strings.Repeat("d", 64), OutputDigest: strings.Repeat("e", 64), ReportDigest: reportDigest, StartedAt: "2026-01-01T00:00:00Z", CompletedAt: "2026-01-01T00:00:01Z", Command: []string{"codex"}, ExitStatus: 0}
		roles = append(roles, teamRoleEvidence{Role: role, AssignmentID: assignment.id, Receipt: receipt, Report: raw})
	}
	checksBytes, _ := json.Marshal(map[string]checkEvidence{})
	checksDigest := sha256.Sum256(checksBytes)
	bundle := teamEvidence{SchemaVersion: 2, RunID: state.ID, Repository: abs, Candidate: state.Candidate, RepositoryDigest: state.ContentDigest, ContractSetDigest: strings.Repeat("f", 64), ChecksDigest: fmt.Sprintf("%x", checksDigest), Workflow: "release-readiness", Roles: roles, Assurance: state.Assurance, Verdict: "NOT_READY", Reason: "finding requires correction", CompletedAt: "2026-01-01T00:00:02Z"}
	_, teamDigest, err := persistTeamEvidence(root, key, bundle)
	if err != nil {
		t.Fatal(err)
	}
	adjudication := adjudicationEvidence{SchemaVersion: 1, RunID: state.ID, Repository: abs, Candidate: state.Candidate, RepositoryDigest: state.ContentDigest, TeamDigest: teamDigest, Trust: "LOCAL_USER_AUTHORIZED", Decisions: []adjudicationDecision{{Kind: "FINDING", ID: "F-1", Disposition: "ACCEPTED", Rationale: "confirmed", EvidenceIDs: []string{}}}, RecordedAt: "2026-01-01T00:00:03Z"}
	adjudicationBytes, _ := json.MarshalIndent(adjudication, "", "  ")
	adjudicationHash := sha256.Sum256(adjudicationBytes)
	adjudicationDigest := fmt.Sprintf("%x", adjudicationHash)
	adjudicationDir := filepath.Join(root, key, "adjudications", state.ID, teamDigest)
	if err := os.MkdirAll(adjudicationDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeExclusiveFile(filepath.Join(root, key, "adjudications", state.ID, teamDigest+".digest"), []byte(adjudicationDigest+"\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(filepath.Join(adjudicationDir, adjudicationDigest+".json"), adjudicationBytes); err != nil {
		t.Fatal(err)
	}
	input := correctionInput{SchemaVersion: 1, TeamDigest: teamDigest, FindingIDs: []string{"F-1"}, AllowedPaths: []string{"src/a.txt"}}
	inputBytes, _ := json.Marshal(input)
	inputPath := filepath.Join(t.TempDir(), "correction.json")
	if err := os.WriteFile(inputPath, inputBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	authorized, err := authorizeCorrection([]string{"--repo", repo, "--candidate", "candidate-a", "--file", inputPath, "--authorized"})
	if err != nil {
		t.Fatal(err)
	}
	recoveredAuthorization, err := authorizeCorrection([]string{"--repo", repo, "--candidate", "candidate-a", "--file", inputPath, "--authorized"})
	if err != nil || recoveredAuthorization["recovered"] != true || recoveredAuthorization["authorizationDigest"] != authorized["authorizationDigest"] {
		t.Fatalf("identical authorization was not recoverable: %#v %v", recoveredAuthorization, err)
	}
	authorizationPath := authorized["path"].(string)
	originalAuthorization, err := os.ReadFile(authorizationPath)
	if err != nil {
		t.Fatal(err)
	}
	tamperedAuthorization := authorized["authorization"].(correctionAuthorization)
	tamperedAuthorization.Candidate = "different-candidate"
	tamperedBytes, _ := json.MarshalIndent(tamperedAuthorization, "", "  ")
	if err := os.Chmod(authorizationPath, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authorizationPath, tamperedBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := authorizeCorrection([]string{"--repo", repo, "--candidate", "candidate-a", "--file", inputPath, "--authorized"}); err == nil || !strings.Contains(err.Error(), "different or invalid") {
		t.Fatalf("mismatched authorization recovered: %v", err)
	}
	if err := os.WriteFile(authorizationPath, originalAuthorization, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(authorizationPath, 0o400); err != nil {
		t.Fatal(err)
	}
	if _, err := finishCorrection([]string{"--repo", repo, "--candidate", "candidate-a", "--new-candidate", "candidate-b", "--authorization", authorized["authorizationDigest"].(string), "--authorized"}); err == nil || !strings.Contains(err.Error(), "no repository changes") {
		t.Fatalf("empty correction was not refused: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "outside.txt"), []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := finishCorrection([]string{"--repo", repo, "--candidate", "candidate-a", "--new-candidate", "candidate-b", "--authorization", authorized["authorizationDigest"].(string), "--authorized"}); err == nil || !strings.Contains(err.Error(), "unauthorized path") {
		t.Fatalf("out-of-scope correction was not refused: %v", err)
	}
	if err := os.Remove(filepath.Join(repo, "outside.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(allowedFile, []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}
	finishCorrectionBeforeStateWriteHook = func() error { return errors.New("injected pre-state-write failure") }
	if _, err := finishCorrection([]string{"--repo", repo, "--candidate", "candidate-a", "--new-candidate", "candidate-b", "--authorization", authorized["authorizationDigest"].(string), "--authorized"}); err == nil || !strings.Contains(err.Error(), "injected") {
		t.Fatalf("completion fault was not injected: %v", err)
	}
	finishCorrectionBeforeStateWriteHook = nil
	t.Cleanup(func() { finishCorrectionBeforeStateWriteHook = nil })
	stillOld, _, err := readState(root, key)
	if err != nil || stillOld.ID != state.ID {
		t.Fatalf("failed completion advanced active state: %#v %v", stillOld, err)
	}
	finished, err := finishCorrection([]string{"--repo", repo, "--candidate", "candidate-a", "--new-candidate", "candidate-b", "--authorization", authorized["authorizationDigest"].(string), "--authorized"})
	if err != nil {
		t.Fatal(err)
	}
	newState := finished["run"].(runState)
	if newState.ParentRunID != state.ID || newState.CorrectionCycle != 1 || newState.Candidate != "candidate-b" || newState.State != "BASELINED" || finished["requiresFreshChecksAndTeam"] != true {
		t.Fatalf("correction did not create a fresh bound run: %#v", finished)
	}
	if _, err := finishCorrection([]string{"--repo", repo, "--candidate", "candidate-a", "--new-candidate", "candidate-b", "--authorization", authorized["authorizationDigest"].(string), "--authorized"}); err == nil {
		t.Fatal("completed correction authorization was replayed")
	}
	newState.CorrectionCycle, newState.ParentRunID = 2, state.ID
	if err := writeState(filepath.Join(root, key, "active.json"), newState); err != nil {
		t.Fatal(err)
	}
	if _, err := authorizeCorrection([]string{"--repo", repo, "--candidate", "candidate-b", "--file", inputPath, "--authorized"}); err == nil || !strings.Contains(err.Error(), "maximum") {
		t.Fatalf("third correction cycle was not refused: %v", err)
	}
}

func TestDoctorIndependentAssuranceFailsClosed(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("JSDLC_PLUGIN_ROOT", filepath.Clean(filepath.Join(workingDir, "..", "..", "plugins", "jerry-sdlc")))
	t.Setenv("PATH", t.TempDir())
	got, err := doctor(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["outcome"] != "UNAVAILABLE" {
		t.Fatalf("unexpected outcome: %#v", got)
	}
	if _, err := start([]string{"--repo", t.TempDir(), "--candidate", "x", "--assurance", "MANAGED_INDEPENDENT"}); err == nil {
		t.Fatal("independent assurance must require unavailable adapter attestation")
	}
}

func TestDoctorReportsSeparatePassesWhenCodexIsAvailable(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("JSDLC_PLUGIN_ROOT", filepath.Clean(filepath.Join(workingDir, "..", "..", "plugins", "jerry-sdlc")))
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "codex"), []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	got, err := doctor(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["outcome"] != "MANAGED_SEPARATE_PASSES" {
		t.Fatalf("unexpected outcome: %#v", got)
	}
	if !validSHA256(got["workflowContractDigest"].(string)) {
		t.Fatalf("doctor omitted workflow validation: %#v", got)
	}
}

func installLifecycleFakeCodex(t *testing.T, binDir string, finding bool) {
	t.Helper()
	disposition := `report='{\"disposition\":\"CLEAN\",\"evidence\":[\"candidate inspected\"],\"findings\":[],\"limitations\":[],\"domains\":[]}'`
	if finding {
		disposition = `report='{\"disposition\":\"FINDINGS\",\"evidence\":[\"defect.txt contains BROKEN\"],\"findings\":[{\"id\":\"QA-KNOWN-DEFECT\",\"severity\":\"HIGH\",\"confidence\":\"HIGH\",\"requirement\":\"release candidate must not contain the known blocking marker\",\"location\":\"defect.txt:1\",\"evidence\":\"observed BROKEN marker\",\"recommendation\":\"replace the marker within defect.txt\"}],\"limitations\":[],\"domains\":[]}'`
	}
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then echo "codex-fixture 1.0"; exit 0; fi
git_check=no
for argument in "$@"; do
  if [ "$argument" = "--skip-git-repo-check" ]; then git_check=yes; fi
done
if [ "$git_check" != yes ]; then
  echo "filesystem-only review requires --skip-git-repo-check" >&2
  exit 92
fi
input=
while IFS= read -r line; do input="$input $line"; done
case "$input" in
  *"Stable assignment ID: specialist-security."*) domain=security ;;
  *"Stable assignment ID: specialist-supply-chain."*) domain=supply-chain ;;
  *"Stable assignment ID: specialist-api-compatibility."*) domain=api-compatibility ;;
  *"Stable assignment ID: specialist-data-migration."*) domain=data-migration ;;
  *"Stable assignment ID: specialist-reliability."*) domain=reliability ;;
  *"Stable assignment ID: specialist-observability."*) domain=observability ;;
  *"Stable assignment ID: specialist-documentation."*) domain=documentation ;;
  *"Stable assignment ID: qa-execution."*) domain=qa-execution ;;
  *"Stable assignment ID: independent-verification."*) domain=independent-verification ;;
  *) domain=qa-architecture ;;
esac
case "$domain" in
  specialist-*) exit 9 ;;
esac
if [ "$domain" = "qa-execution" ]; then
  ` + disposition + `
elif [ "$domain" = "qa-architecture" ]; then
  report='{\"disposition\":\"CLEAN\",\"evidence\":[\"candidate inspected & bounded <read-only>\"],\"findings\":[],\"limitations\":[],\"domains\":[]}'
elif [ "$domain" = "independent-verification" ]; then
  report='{\"disposition\":\"INCONCLUSIVE\",\"evidence\":[],\"findings\":[],\"limitations\":[\"local fake adapter cannot attest checks or independence\"],\"domains\":[]}'
else
  case "$domain" in
    security) report='{\"disposition\":\"CLEAN\",\"evidence\":[\"lens inspected\"],\"findings\":[],\"limitations\":[],\"domains\":[{\"domain\":\"security\",\"status\":\"BLOCKED\",\"evidence\":\"no attested check available\",\"evidenceIds\":[]}]}' ;;
    supply-chain) report='{\"disposition\":\"CLEAN\",\"evidence\":[\"lens inspected\"],\"findings\":[],\"limitations\":[],\"domains\":[{\"domain\":\"supply-chain\",\"status\":\"BLOCKED\",\"evidence\":\"no attested check available\",\"evidenceIds\":[]}]}' ;;
    api-compatibility) report='{\"disposition\":\"CLEAN\",\"evidence\":[\"lens inspected\"],\"findings\":[],\"limitations\":[],\"domains\":[{\"domain\":\"api-compatibility\",\"status\":\"BLOCKED\",\"evidence\":\"no attested check available\",\"evidenceIds\":[]}]}' ;;
    data-migration) report='{\"disposition\":\"CLEAN\",\"evidence\":[\"lens inspected\"],\"findings\":[],\"limitations\":[],\"domains\":[{\"domain\":\"data-migration\",\"status\":\"BLOCKED\",\"evidence\":\"no attested check available\",\"evidenceIds\":[]}]}' ;;
    reliability) report='{\"disposition\":\"CLEAN\",\"evidence\":[\"lens inspected\"],\"findings\":[],\"limitations\":[],\"domains\":[{\"domain\":\"reliability\",\"status\":\"BLOCKED\",\"evidence\":\"no attested check available\",\"evidenceIds\":[]}]}' ;;
    observability) report='{\"disposition\":\"CLEAN\",\"evidence\":[\"lens inspected\"],\"findings\":[],\"limitations\":[],\"domains\":[{\"domain\":\"observability\",\"status\":\"BLOCKED\",\"evidence\":\"no attested check available\",\"evidenceIds\":[]}]}' ;;
    documentation) report='{\"disposition\":\"CLEAN\",\"evidence\":[\"lens inspected\"],\"findings\":[],\"limitations\":[],\"domains\":[{\"domain\":\"documentation\",\"status\":\"BLOCKED\",\"evidence\":\"no attested check available\",\"evidenceIds\":[]}]}' ;;
  esac
fi
printf '%s\n' "{\"type\":\"thread.started\",\"thread_id\":\"thread-$$\"}" "{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"$report\"}}"
`
	if err := os.WriteFile(filepath.Join(binDir, "codex"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestFakeAdapterReleaseReadinessLifecycle(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("JSDLC_PLUGIN_ROOT", filepath.Clean(filepath.Join(workingDir, "..", "..", "plugins", "jerry-sdlc")))
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	repo := t.TempDir()
	defectPath := filepath.Join(repo, "defect.txt")
	if err := os.WriteFile(defectPath, []byte("BROKEN\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := start([]string{"--repo", repo, "--candidate", "fixture-broken"}); err != nil {
		t.Fatal(err)
	}
	installLifecycleFakeCodex(t, binDir, true)
	first, err := team([]string{"--repo", repo, "--candidate", "fixture-broken", "--objective", "Assess the known blocking fixture without release actions"})
	if err != nil {
		t.Fatal(err)
	}
	if first["verdict"] != "NOT_READY" {
		t.Fatalf("known blocking defect did not block: %#v", first)
	}
	decisionPath := filepath.Join(t.TempDir(), "adjudication.json")
	decision, _ := json.Marshal(adjudicationInput{SchemaVersion: 1, TeamDigest: first["evidenceDigest"].(string), Decisions: []adjudicationDecision{{Kind: "FINDING", ID: "QA-KNOWN-DEFECT", Disposition: "ACCEPTED", Rationale: "fixture intentionally confirms this defect", EvidenceIDs: []string{}}}})
	if err := os.WriteFile(decisionPath, decision, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := adjudicate([]string{"--repo", repo, "--candidate", "fixture-broken", "--file", decisionPath, "--authorized"}); err != nil {
		t.Fatal(err)
	}
	correctionPath := filepath.Join(t.TempDir(), "correction.json")
	correction, _ := json.Marshal(correctionInput{SchemaVersion: 1, TeamDigest: first["evidenceDigest"].(string), FindingIDs: []string{"QA-KNOWN-DEFECT"}, AllowedPaths: []string{"defect.txt"}})
	if err := os.WriteFile(correctionPath, correction, 0o600); err != nil {
		t.Fatal(err)
	}
	authorized, err := authorizeCorrection([]string{"--repo", repo, "--candidate", "fixture-broken", "--file", correctionPath, "--authorized"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(defectPath, []byte("FIXED\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := finishCorrection([]string{"--repo", repo, "--candidate", "fixture-broken", "--new-candidate", "fixture-corrected", "--authorization", authorized["authorizationDigest"].(string), "--authorized"}); err != nil {
		t.Fatal(err)
	}
	installLifecycleFakeCodex(t, binDir, false)
	second, err := team([]string{"--repo", repo, "--candidate", "fixture-corrected", "--objective", "Re-review the corrected fixture without release actions"})
	if err != nil {
		t.Fatal(err)
	}
	if second["verdict"] != "INCONCLUSIVE" || second["assurance"] != "MANAGED_SEPARATE_PASSES" {
		t.Fatalf("local re-review exceeded its assurance: %#v", second)
	}
	verified, err := verify([]string{"--repo", repo, "--candidate", "fixture-corrected"})
	if err != nil {
		t.Fatal(err)
	}
	if verified["verdict"] != "INCONCLUSIVE" || verified["reproduced"] != true {
		t.Fatalf("corrected lifecycle did not reproduce safely: %#v", verified)
	}
	t.Logf("fixture outcomes: initial=%s corrected=%s verified=%s assurance=%s", first["verdict"], second["verdict"], verified["verdict"], second["assurance"])
}

func TestDoctorFailsClosedOnWorkflowDrift(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "workflows"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "workflows", "release-readiness.json"), []byte(`{"schemaVersion":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JSDLC_PLUGIN_ROOT", root)
	got, err := doctor(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["outcome"] != "UNAVAILABLE" || !strings.Contains(got["reason"].(string), "canonical workflow validation failed") {
		t.Fatalf("doctor accepted workflow drift: %#v", got)
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
