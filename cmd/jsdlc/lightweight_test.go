package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func lightweightFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("JSDLC_PLUGIN_ROOT", filepath.Join(wd, "..", "..", "plugins", "jerry-sdlc"))
	return t.TempDir()
}
func fixtureJSON(t *testing.T, value any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.json")
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
func fixtureWrite(t *testing.T, repo, path, value string) {
	t.Helper()
	full := filepath.Join(repo, path)
	if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLightweightCompletionUsesOneCheckAndCombinedRoles(t *testing.T) {
	repo := lightweightFixture(t)
	fixtureWrite(t, repo, "code.txt", "candidate")
	started, err := deliveryStart([]string{"--repo", repo, "--candidate", "c", "--workflow", "feature", "--experience", "full", "--add-roles", "qa-executor,verifier", "--role-reason", "Check record integrity"})
	if err != nil {
		t.Fatal(err)
	}
	run := started["run"].(deliveryRun)
	if len(run.RequiredRoles) != 4 || contains(run.RequiredRoles, "product-designer") {
		t.Fatalf("unexpected light roles: %#v", run.RequiredRoles)
	}
	if _, err := deliveryCheck([]string{"--repo", repo, "--candidate", "c", "--id", "one", "--authorized", "--", "/bin/true"}); err != nil {
		t.Fatal(err)
	}
	for _, role := range run.RequiredRoles {
		report := fixtureJSON(t, deliveryReportInput{"CLEAN", "Observed criteria", []string{"Actual check"}, []string{"one"}, []deliveryFinding{}, []string{}})
		if _, err := deliveryRecordPass([]string{"--repo", repo, "--candidate", "c", "--role", role, "--file", report}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := deliveryVerify([]string{"--repo", repo, "--candidate", "c"})
	if err != nil || got["outcome"] != "COMPLETE" || got["releaseReadinessEffect"] != "NONE" {
		t.Fatalf("one-check delivery: %#v %v", got, err)
	}
	if _, err := roles([]string{"--workflow", "feature", "--add-roles", "ux-reviewer"}); err == nil {
		t.Fatal("unjustified role accepted")
	}
}

func TestLightweightOwnerRequiresObservedCheck(t *testing.T) {
	repo := lightweightFixture(t)
	_, err := deliveryStart([]string{"--repo", repo, "--candidate", "c", "--workflow", "trivial-change"})
	if err != nil {
		t.Fatal(err)
	}
	report := fixtureJSON(t, deliveryReportInput{"CLEAN", "Edited", []string{"Inspected source"}, []string{}, []deliveryFinding{}, []string{}})
	if _, err := deliveryRecordPass([]string{"--repo", repo, "--candidate", "c", "--role", "implementer", "--file", report}); err != nil {
		t.Fatal(err)
	}
	got, err := deliveryVerify([]string{"--repo", repo, "--candidate", "c"})
	if err != nil || got["outcome"] != "BLOCKED" {
		t.Fatalf("unobserved check accepted: %#v %v", got, err)
	}
}

func TestLightweightAdditionalContractDriftFailsClosed(t *testing.T) {
	repo := lightweightFixture(t)
	original := os.Getenv("JSDLC_PLUGIN_ROOT")
	plugin := t.TempDir()
	paths := []string{"workflows/everyday-delivery.json", "schemas/delivery-report-input.schema.json"}
	for _, role := range append(append([]string{}, deliveryWorkflowRoles["feature"]...), "ux-reviewer") {
		paths = append(paths, "roles/"+role+".md")
	}
	for _, path := range paths {
		b, err := os.ReadFile(filepath.Join(original, path))
		if err != nil {
			t.Fatal(err)
		}
		fixtureWrite(t, plugin, path, string(b))
	}
	t.Setenv("JSDLC_PLUGIN_ROOT", plugin)
	got, err := deliveryStart([]string{"--repo", repo, "--candidate", "c", "--workflow", "feature", "--add-roles", "ux-reviewer", "--role-reason", "Review keyboard recovery"})
	if err != nil {
		t.Fatal(err)
	}
	run := got["run"].(deliveryRun)
	if err := validateDeliveryBinding(run, repo); err != nil {
		t.Fatal(err)
	}
	fixtureWrite(t, plugin, "roles/ux-reviewer.md", "Changed specialist requirements")
	if err := validateDeliveryBinding(run, repo); err == nil {
		t.Fatal("added role contract drift did not invalidate the candidate")
	}
	if _, err := deliveryStart([]string{"--repo", repo, "--candidate", "next", "--workflow", "feature", "--profile", ""}); err == nil {
		t.Fatal("explicit empty profile bypassed the new-record policy")
	}
}

func TestScopedCandidateIgnoresUnrelatedEditsButBindsMembership(t *testing.T) {
	repo := lightweightFixture(t)
	fixtureWrite(t, repo, "src/code.txt", "candidate")
	fixtureWrite(t, repo, "notes.txt", "other work")
	inputs := fixtureJSON(t, []string{"src"})
	got, err := deliveryStart([]string{"--repo", repo, "--candidate", "c", "--workflow", "feature", "--inputs", inputs})
	if err != nil || got["candidateScope"] != "CALLER_DECLARED_INPUTS" {
		t.Fatalf("scoped start: %#v %v", got, err)
	}
	fixtureWrite(t, repo, "notes.txt", "concurrent unrelated edit")
	status, err := deliveryStatus([]string{"--repo", repo, "--candidate", "c"})
	if err != nil || status["stale"] != false {
		t.Fatalf("unrelated edit invalidated scope: %#v %v", status, err)
	}
	fixtureWrite(t, repo, "src/new-consumer.txt", "new consumer")
	status, err = deliveryStatus([]string{"--repo", repo, "--candidate", "c"})
	if err != nil || status["stale"] != true {
		t.Fatalf("directory membership ignored: %#v %v", status, err)
	}
}

func TestReusableCheckProvenanceAndInvalidation(t *testing.T) {
	for _, scenario := range []string{"reuse", "changed-input", "new-file", "different-runtime", "failed", "removed-sidecar", "tampered-source", "escape"} {
		t.Run(scenario, func(t *testing.T) {
			repo := lightweightFixture(t)
			fixtureWrite(t, repo, "src/code.txt", "candidate")
			fixtureWrite(t, repo, "notes.txt", "documentation")
			inputs := fixtureJSON(t, []string{"src"})
			first, err := deliveryStart([]string{"--repo", repo, "--candidate", "one", "--workflow", "feature"})
			if err != nil {
				t.Fatal(err)
			}
			old := first["run"].(deliveryRun)
			command := "/bin/true"
			if scenario == "failed" {
				command = "/bin/false"
			}
			checked, err := deliveryCheck([]string{"--repo", repo, "--candidate", "one", "--id", "unit", "--inputs", inputs, "--environment", "image@sha256:fixture", "--authorized", "--", command})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := deliveryCancel([]string{"--repo", repo, "--candidate", "one"}); err != nil {
				t.Fatal(err)
			}
			fixtureWrite(t, repo, "notes.txt", "corrected docs")
			if scenario == "changed-input" {
				fixtureWrite(t, repo, "src/code.txt", "changed implementation")
			}
			if scenario == "new-file" {
				fixtureWrite(t, repo, "src/new.txt", "new dependency")
			}
			if scenario == "escape" {
				if err := os.Symlink(t.TempDir(), filepath.Join(repo, "src/link")); err != nil {
					t.Fatal(err)
				}
			}
			second, err := deliveryStart([]string{"--repo", repo, "--candidate", "two", "--workflow", "feature"})
			if err != nil {
				t.Fatal(err)
			}
			current := second["run"].(deliveryRun)
			env := "image@sha256:fixture"
			if scenario == "different-runtime" {
				env = "image@sha256:other"
			}
			got, err := deliveryReuseCheck([]string{"--repo", repo, "--candidate", "two", "--id", "reused", "--source-run", old.ID, "--source-check", "unit", "--environment", env})
			if contains([]string{"changed-input", "new-file", "different-runtime", "failed", "escape"}, scenario) {
				if err == nil {
					t.Fatalf("unsafe reuse accepted: %#v", got)
				}
				return
			}
			if err != nil || got["execution"] != "REUSED_LOCAL_CHECK" {
				t.Fatalf("compatible reuse: %#v %v", got, err)
			}
			_, key, root, _ := stateLocation(repo)
			current, _, err = readDeliveryRun(root, key)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "removed-sidecar" {
				if err := os.Remove(deliveryDependenciesPath(root, key, current.ID, "reused")); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "tampered-source" {
				e := checked["check"].(checkEvidence)
				e.StdoutDigest = strings.Repeat("a", 64)
				if err := persistCheckEvidence(checked["path"].(string), e); err != nil {
					t.Fatal(err)
				}
			}
			_, err = validateDeliveryCheck(root, key, current, "reused")
			if scenario == "reuse" && err != nil {
				t.Fatal(err)
			}
			if scenario != "reuse" && err == nil {
				t.Fatal("missing or tampered provenance accepted")
			}
		})
	}
}

func TestDependencyPathsRejectTraversalAndIntermediateSymlinks(t *testing.T) {
	repo := t.TempDir()
	outside := t.TempDir()
	fixtureWrite(t, outside, "secret.txt", "not a dependency")
	if err := os.Symlink(outside, filepath.Join(repo, "escape")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../secret.txt", outside, "escape/secret.txt", ".git/config", "."} {
		if _, err := digestDeliveryInputs(repo, []string{path}); err == nil {
			t.Fatalf("unsafe dependency path %q", path)
		}
	}
}

func TestLegacyProfileOmissionRetainsFullRoles(t *testing.T) {
	repo := lightweightFixture(t)
	started, err := deliveryStart([]string{"--repo", repo, "--candidate", "c", "--workflow", "feature", "--profile", "full"})
	if err != nil {
		t.Fatal(err)
	}
	run := started["run"].(deliveryRun)
	run.Profile = ""
	if err := writeDeliveryJSON(started["path"].(string), run); err != nil {
		t.Fatal(err)
	}
	_, key, root, _ := stateLocation(repo)
	read, _, err := readDeliveryRun(root, key)
	if err != nil || !equalStrings(read.RequiredRoles, deliveryWorkflowRoles["feature"]) {
		t.Fatalf("legacy reinterpretation: %#v %v", read, err)
	}
}

func TestSliceBudgetsUnknownsRoundsAndRecordedExhaustion(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "ledger.json")
	got, err := sliceBudget("slice-start", []string{"--file", ledger, "--id", "slice"})
	if err != nil {
		t.Fatal(err)
	}
	if got["grossTokens"] != nil || got["activeSeconds"] != nil {
		t.Fatal("unknown measurements became zero")
	}
	record := func(e sliceEvent) (result, error) {
		return sliceBudget("slice-record", []string{"--file", ledger, "--event", fixtureJSON(t, e)})
	}
	for _, id := range []string{"initial", "correction"} {
		if _, err := record(sliceEvent{ID: id, Kind: "review", Evidence: "Candidate inspected"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := record(sliceEvent{ID: "extra", Kind: "review", Evidence: "Optional polish"}); err == nil {
		t.Fatal("unbounded review admitted")
	}
	if _, err := record(sliceEvent{ID: "blocker", Kind: "review", Evidence: "Race reproduced", MaterialBlocker: "Duplicate billing race"}); err != nil {
		t.Fatal(err)
	}
	if _, err := record(sliceEvent{ID: "again", Kind: "review", Evidence: "More review", MaterialBlocker: "Another review"}); err == nil {
		t.Fatal("second extension admitted")
	}
	u := modelUsage{Input: 600000, Cached: 1000, Output: 5000}
	got, err = record(sliceEvent{ID: "usage", Kind: "measurement", Evidence: "All worker deltas", Usage: &u, Coverage: "ALL_WORKERS"})
	if err != nil || got["outcome"] != "CHECKPOINT_REQUIRED" || got["uncachedInputPlusOutputTokens"] != int64(604000) {
		t.Fatalf("overrun lost: %#v %v", got, err)
	}
	if _, err := record(sliceEvent{ID: "checkpoint", Kind: "checkpoint", Evidence: "Candidate and remaining criteria saved"}); err != nil {
		t.Fatal(err)
	}
	if _, err := record(sliceEvent{ID: "usage", Kind: "measurement", Evidence: "Duplicated counters", Usage: &u, Coverage: "ALL_WORKERS"}); err == nil {
		t.Fatal("duplicate usage admitted")
	}
	bad := fixtureJSON(t, map[string]any{"id": "unknown", "kind": "measurement", "evidence": "missing counters", "usage": map[string]int{"inputTokens": 20}, "tokenCoverage": "ALL_WORKERS"})
	if _, err := sliceBudget("slice-record", []string{"--file", ledger, "--event", bad}); err == nil {
		t.Fatal("missing counters became zero")
	}
}

func TestBoundedPreflightAndWaitPreserveJobIdentity(t *testing.T) {
	if _, err := deliveryProbe("preflight", []string{"--", "/bin/true"}); err == nil {
		t.Fatal("unauthorized probe")
	}
	got, err := deliveryProbe("preflight", []string{"--authorized", "--", "/bin/false"})
	if err != nil || got["outcome"] != "FAILED" || got["attempts"] != 1 {
		t.Fatalf("failed preflight: %#v %v", got, err)
	}
	got, err = deliveryProbe("wait-check", []string{"--authorized", "--job", "existing-job", "--timeout", "50ms", "--interval", "10ms", "--", "/bin/sh", "-c", "exit 3"})
	if err != nil || got["outcome"] != "TIMED_OUT" || got["job"] != "existing-job" || got["resumeSameJob"] != true {
		t.Fatalf("bounded wait: %#v %v", got, err)
	}
	got, err = deliveryProbe("preflight", []string{"--authorized", "--timeout", "50ms", "--", "/bin/sleep", "5"})
	if err != nil || got["outcome"] != "TIMED_OUT" {
		t.Fatalf("hung probe escaped bound: %#v %v", got, err)
	}
	started := time.Now()
	got, err = deliveryProbe("preflight", []string{"--authorized", "--timeout", "50ms", "--", "/bin/sh", "-c", "sleep 5 & exit 0"})
	if err == nil && got["outcome"] != "TIMED_OUT" || time.Since(started) >= 3*time.Second {
		t.Fatalf("descendant-held pipes escaped the probe bound: %v", err)
	}
	pidFile := filepath.Join(t.TempDir(), "child-pid")
	// The shell's nonzero exit takes precedence over a pipe-drain timeout.
	// Its status-only child must still be cleaned up before another poll.
	got, err = deliveryProbe("wait-check", []string{"--authorized", "--job", "pending-child", "--timeout", "50ms", "--interval", "10ms", "--", "/bin/sh", "-c", "sleep 5 & echo $! > \"$1\"; exit 3", "probe", pidFile})
	if err != nil || got["outcome"] != "TIMED_OUT" {
		t.Fatalf("pending-child wait: %#v %v", got, err)
	}
	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	if processRunning(pid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Fatalf("pending probe left its child alive: %d", pid)
	}
}
