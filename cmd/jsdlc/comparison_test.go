package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExploratoryComparisonMatchesTrialsAndMeasuresCosts(t *testing.T) {
	file := "../../evals/synthetic-comparison.json"
	got, err := evalComparison([]string{"--fixture", file})
	if err != nil || got["passed"] != true || got["graduationEffect"] != "NONE" || got["realQualityClaimSupported"] != false {
		t.Fatalf("synthetic harness: %#v %v", got, err)
	}
	arms := got["arms"].(result)
	light := arms["lightweight"].(result)
	if light["medianUncachedInputPlusOutputTokens"] != int64(6200) || light["medianGrossTokens"] != int64(11200) || light["medianWaitMilliseconds"] != int64(200) {
		t.Fatalf("cost counters mixed: %#v", light)
	}
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"mismatched-trials", "different-environment", "unknown-cost", "missing-usage-field", "partial-workers", "duplicate-task-id", "duplicate-finding-id", "cost-overrun", "false-finding", "reopened", "unsafe-baseline", "no-ground-truth"} {
		t.Run(scenario, func(t *testing.T) {
			var suite map[string]any
			if err := json.Unmarshal(b, &suite); err != nil {
				t.Fatal(err)
			}
			task := suite["tasks"].([]any)[0].(map[string]any)
			arms := task["arms"].(map[string]any)
			runs := arms["lightweight"].([]any)
			r := runs[0].(map[string]any)
			invalid := true
			switch scenario {
			case "mismatched-trials":
				arms["lightweight"] = runs[:2]
			case "different-environment":
				r["conditionsDigest"] = "b"
			case "unknown-cost":
				delete(r, "activeMilliseconds")
			case "missing-usage-field":
				delete(r["usage"].(map[string]any), "outputTokens")
			case "partial-workers":
				r["tokenCoverage"] = "PARTIAL"
			case "duplicate-task-id":
				copyBytes, _ := json.Marshal(task)
				var duplicate map[string]any
				if err := json.Unmarshal(copyBytes, &duplicate); err != nil {
					t.Fatal(err)
				}
				duplicate["id"] = "  " + task["id"].(string) + "  "
				duplicate["candidate"] = "another-candidate"
				suite["tasks"] = append(suite["tasks"].([]any), duplicate)
			case "duplicate-finding-id":
				findings := task["knownFindings"].([]any)
				findings = append(findings, map[string]any{"id": "  " + findings[0].(map[string]any)["id"].(string) + "  ", "severity": "HIGH"})
				task["knownFindings"] = findings
			case "cost-overrun":
				for _, r := range runs {
					r.(map[string]any)["activeMilliseconds"] = 10000
				}
				invalid = false
			case "false-finding":
				for _, v := range runs {
					r := v.(map[string]any)
					r["findings"] = append(r["findings"].([]any), map[string]any{"id": "invented", "outcome": "FALSE_POSITIVE"})
				}
				invalid = false
			case "reopened":
				r["reopenedTasks"] = 1
				invalid = false
			case "unsafe-baseline":
				arms["baseline"].([]any)[0].(map[string]any)["unauthorizedActions"] = 1
				invalid = false
			case "no-ground-truth":
				task["knownFindings"] = []any{}
				for _, vs := range arms {
					for _, v := range vs.([]any) {
						v.(map[string]any)["findings"] = []any{}
					}
				}
				invalid = false
			}
			path := fixtureJSON(t, suite)
			got, err := evalComparison([]string{"--fixture", path})
			if invalid && err == nil {
				t.Fatalf("invalid comparison accepted: %#v", got)
			}
			if !invalid && (err != nil || got["passed"] != false) {
				t.Fatalf("negative outcome lost: %#v %v", got, err)
			}
		})
	}
	if _, err := evalComparison([]string{"--fixture", file, "--max-token-ratio", "NaN"}); err == nil {
		t.Fatal("NaN threshold accepted")
	}
}

func TestClassificationKeepsFormalPersistenceExplicit(t *testing.T) {
	for _, tc := range []struct{ request, profile, persistence string }{
		{"Change the button label", "lightweight", "OPTIONAL"},
		{"Assess release readiness", "full", "REQUIRED"},
	} {
		got, err := classify([]string{"--request", tc.request})
		if err != nil || got["profile"] != tc.profile || got["persistence"] != tc.persistence {
			t.Fatalf("%q: %#v %v", tc.request, got, err)
		}
	}
}

func TestNewCommandsUsePackagedCLI(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cli := filepath.Join(wd, "../../plugins/jerry-sdlc/scripts/jsdlc")
	if _, err := os.Stat(cli); err != nil {
		t.Fatal(err)
	}
	// Invoke the shipped wrapper so negative outcomes cannot hide behind a
	// success exit code. The existing lifecycle covers release separation.
	for _, tc := range []struct {
		command string
		args    []string
		exit    int
	}{
		{"preflight", []string{"--authorized", "--", "/bin/true"}, 0},
		{"preflight", []string{"--authorized", "--", "/bin/false"}, 2},
		{"wait-check", []string{"--authorized", "--job", "same-job", "--timeout", "30ms", "--interval", "10ms", "--", "/bin/sh", "-c", "exit 3"}, 3},
		{"eval-comparison", []string{"--fixture", "../../evals/synthetic-comparison.json"}, 0},
		{"eval-comparison", []string{"--fixture", "../../evals/synthetic-comparison.json", "--max-active-ratio", "1"}, 2},
	} {
		cmd := exec.Command(cli, append([]string{tc.command}, tc.args...)...)
		output, err := cmd.CombinedOutput()
		code := 0
		if err != nil {
			ee, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatal(err)
			}
			code = ee.ExitCode()
		}
		if code != tc.exit {
			t.Fatalf("%s exit %d want %d: %s", tc.command, code, tc.exit, output)
		}
		var result map[string]any
		if err := json.Unmarshal(output, &result); err != nil {
			t.Fatalf("missing JSON result: %s", output)
		}
	}
}
