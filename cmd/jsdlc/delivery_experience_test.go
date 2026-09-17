package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExperienceClassification(t *testing.T) {
	for _, tc := range []struct{ request, files, experience, risk string }{
		{"Build an onboarding flow", "", "full", "NORMAL"},
		{"Fix keyboard navigation", "", "focused", "NORMAL"},
		{"Update layout", "src/app.tsx", "full", "NORMAL"},
		{"Fix authentication form", "", "focused", "HIGH"},
		{"Refactor queue worker", "worker.go", "none", "NORMAL"},
		{"Build squirrel database", "", "none", "NORMAL"},
		{"Refactor queue worker", "src/notcomponents/worker.go", "none", "NORMAL"},
		{"Refactor queue worker", "src/worker.tsxcache", "none", "NORMAL"},
		{"Update component", "src/components/Control.js", "full", "NORMAL"},
		{"Add CLI help text", "", "full", "NORMAL"},
		{"Fix confirmation prompt", "", "focused", "NORMAL"},
		{"Create a transactional email", "", "full", "NORMAL"},
		{"Build a new screen", "", "full", "NORMAL"},
		{"Refactor client transport", "client.go", "none", "NORMAL"},
	} {
		got, err := classify([]string{"--request", tc.request, "--files", tc.files})
		if err != nil || got["experience"] != tc.experience || got["risk"] != tc.risk {
			t.Errorf("%q: %#v %v", tc.request, got, err)
		}
	}
}

func TestDesignReviewRouting(t *testing.T) {
	for _, tc := range []struct{ request, workflow string }{
		{"Review this CLI deletion confirmation for clarity; do not change code", "pr-review"},
		{"Critique the existing interface", "pr-review"},
		{"Give me feedback on the onboarding flow", "pr-review"},
		{"Review this design", "pr-review"},
		{"Assess this flow", "pr-review"},
		{"Review this copy", "pr-review"},
		{"Review the onboarding flow and implement the improvements", "feature"},
		{"Critique the dashboard then redesign it", "feature"},
		{"Build a design review dashboard", "feature"},
		{"Review payment release readiness", "release-readiness"},
		{"Review the checkout interface before production release", "release-readiness"},
		{"Review the interface; production is down", "incident"},
	} {
		got, err := classify([]string{"--request", tc.request})
		if err != nil || got["workflow"] != tc.workflow {
			t.Errorf("%q: %#v %v", tc.request, got, err)
		}
		if tc.workflow == "pr-review" && got["experience"] != "focused" {
			t.Errorf("read-only design review lacks UX participation: %#v", got)
		}
		uxTriggers := 0
		for _, trigger := range got["triggers"].([]string) {
			if trigger == "ux-accessibility" {
				uxTriggers++
			}
		}
		if tc.workflow == "pr-review" && uxTriggers != 1 {
			t.Errorf("expected one UX trigger: %#v", got)
		}
	}
}

func TestExperienceRolesAndEvidence(t *testing.T) {
	for _, experience := range []string{"none", "focused", "full"} {
		t.Run(experience, func(t *testing.T) {
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("JSDLC_PLUGIN_ROOT", filepath.Join(wd, "..", "..", "plugins", "jerry-sdlc"))
			repo := t.TempDir()
			started, err := deliveryStart([]string{"--repo", repo, "--candidate", "c", "--workflow", "bug-diagnosis", "--experience", experience})
			if err != nil {
				t.Fatal(err)
			}
			run := started["run"].(deliveryRun)
			selected, err := roles([]string{"--workflow", "bug-diagnosis", "--experience", experience})
			if err != nil || !equalStrings(run.RequiredRoles, selected["roles"].([]string)) {
				t.Fatalf("roles/start mismatch: %#v %v", selected, err)
			}
			reportPath := filepath.Join(t.TempDir(), "report.json")
			record := func(role string, checks []string) {
				t.Helper()
				report := deliveryReportInput{Disposition: "CLEAN", Summary: "observed", Evidence: []string{"inspected candidate"}, CheckIDs: checks, Findings: []deliveryFinding{}, Limitations: []string{}}
				b, err := json.Marshal(report)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(reportPath, b, 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := deliveryRecordPass([]string{"--repo", repo, "--candidate", "c", "--role", role, "--file", reportPath}); err != nil {
					t.Fatal(err)
				}
			}
			verify := func() result {
				t.Helper()
				got, err := deliveryVerify([]string{"--repo", repo, "--candidate", "c"})
				if err != nil {
					t.Fatal(err)
				}
				return got
			}
			for _, role := range deliveryWorkflowRoles["bug-diagnosis"] {
				record(role, []string{})
			}
			if experience == "none" {
				if got := verify(); got["outcome"] != "COMPLETE" {
					t.Fatalf("backend completion changed: %#v", got)
				}
				return
			}
			if got := verify(); got["outcome"] != "INCOMPLETE" {
				t.Fatalf("missing UX evidence completed: %#v", got)
			}
			if experience == "full" {
				record("product-designer", []string{})
			}
			if _, err := deliveryCheck([]string{"--repo", repo, "--candidate", "c", "--id", "rendered", "--authorized", "--", "/bin/true"}); err != nil {
				t.Fatal(err)
			}
			record("ux-reviewer", []string{"rendered"})
			if got := verify(); got["outcome"] != "COMPLETE" || got["releaseReadinessEffect"] != "NONE" {
				t.Fatalf("UX contract completion failed: %#v", got)
			}
		})
	}
	for _, value := range []string{"", "ui", "FULL", "unknown"} {
		if _, err := roles([]string{"--workflow", "feature", "--experience", value}); err == nil {
			t.Errorf("invalid experience accepted: %s", value)
		}
		if _, err := deliveryStart([]string{"--candidate", "c", "--workflow", "feature", "--experience", value}); err == nil {
			t.Errorf("invalid start experience accepted: %s", value)
		}
	}
	if _, err := roles([]string{"--workflow", "release-readiness", "--experience", "full"}); err == nil {
		t.Fatal("everyday extension changed formal release contract")
	}
}

func TestExperienceContractBinding(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("JSDLC_PLUGIN_ROOT", filepath.Join(wd, "..", "..", "plugins", "jerry-sdlc"))
	repo := t.TempDir()
	started, err := deliveryStart([]string{"--repo", repo, "--candidate", "c", "--workflow", "feature", "--experience", "full"})
	if err != nil {
		t.Fatal(err)
	}
	run := started["run"].(deliveryRun)
	_, key, root, err := stateLocation(repo)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, key, "delivery", "active.json")
	run.RequiredRoles = deliveryWorkflowRoles["feature"]
	if err := writeDeliveryJSON(path, run); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readDeliveryRun(root, key); err == nil {
		t.Fatal("removing required experience roles bypassed state validation")
	}
	run.Experience = "none"
	if err := writeDeliveryJSON(path, run); err != nil {
		t.Fatal(err)
	}
	if err := validateDeliveryBinding(run, repo); err == nil {
		t.Fatal("downgrading experience bypassed contract digest")
	}
	_, noneDigest, err := loadDeliveryCatalog("feature", "none")
	if err != nil {
		t.Fatal(err)
	}
	_, focusedDigest, err := loadDeliveryCatalog("feature", "focused")
	if err != nil {
		t.Fatal(err)
	}
	_, fullDigest, err := loadDeliveryCatalog("feature", "full")
	if err != nil {
		t.Fatal(err)
	}
	if noneDigest == focusedDigest || focusedDigest == fullDigest || noneDigest == fullDigest {
		t.Fatal("experience not bound to contract digest")
	}
}

func TestCleanUXReportRequiresCheck(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("JSDLC_PLUGIN_ROOT", filepath.Join(wd, "..", "..", "plugins", "jerry-sdlc"))
	repo := t.TempDir()
	started, err := deliveryStart([]string{"--repo", repo, "--candidate", "c", "--workflow", "bug-diagnosis", "--experience", "focused"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "report.json")
	report := `{"disposition":"CLEAN","summary":"observed","evidence":["source only"],"checkIds":[],"findings":[],"limitations":[]}`
	if err := os.WriteFile(path, []byte(report), 0600); err != nil {
		t.Fatal(err)
	}
	for _, role := range started["run"].(deliveryRun).RequiredRoles {
		if _, err := deliveryRecordPass([]string{"--repo", repo, "--candidate", "c", "--role", role, "--file", path}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := deliveryVerify([]string{"--repo", repo, "--candidate", "c"})
	if err != nil || got["outcome"] != "BLOCKED" {
		t.Fatalf("unobserved UX claimed complete: %#v %v", got, err)
	}
}
