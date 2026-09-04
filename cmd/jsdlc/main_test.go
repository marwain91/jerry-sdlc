package main

import (
	"encoding/json"
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

func TestReleaseRoles(t *testing.T) {
	got, err := roles([]string{"--workflow", "release-readiness"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got["roles"].([]string)) != 5 {
		t.Fatalf("unexpected roles: %#v", got)
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
