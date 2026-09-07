package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Run the actual CLI entrypoint in a child process so os.Exit is observable.
func TestEvaluationCLIProcess(t *testing.T) {
	if os.Getenv("JSDLC_TEST_EVAL_PROCESS") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"jsdlc"}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	os.Exit(99)
}

func TestEvaluationCLIExitCodes(t *testing.T) {
	// Rotate labels while preserving the valid balanced suite envelope, yielding
	// a structurally valid workflow evaluation which must miss its thresholds.
	b, err := os.ReadFile("../../evals/workflow-routing-suite.json")
	if err != nil {
		t.Fatal(err)
	}
	var suite map[string]any
	if err := json.Unmarshal(b, &suite); err != nil {
		t.Fatal(err)
	}
	labels := []string{"release-readiness", "feature", "bug-fix", "bug-diagnosis", "pr-review", "trivial-change", "incident"}
	for _, item := range suite["cases"].([]any) {
		entry := item.(map[string]any)
		for i, label := range labels {
			if entry["workflow"] == label {
				entry["workflow"] = labels[(i+1)%len(labels)]
				break
			}
		}
	}
	badWorkflow := filepath.Join(t.TempDir(), "failed-workflows.json")
	b, err = json.Marshal(suite)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badWorkflow, b, 0600); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile("../../evals/holdout-3-trigger-suite.json")
	if err != nil {
		t.Fatal(err)
	}
	var triggers triggerSuite
	if err := json.Unmarshal(b, &triggers); err != nil {
		t.Fatal(err)
	}
	for i := range triggers.Cases {
		triggers.Cases[i].Release = !triggers.Cases[i].Release
	}
	b, err = json.Marshal(triggers)
	if err != nil {
		t.Fatal(err)
	}
	badTriggers := filepath.Join(t.TempDir(), "failed-triggers.json")
	if err := os.WriteFile(badTriggers, b, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, command, fixture string
		exit                   int
		passed                 bool
	}{
		{"workflows pass", "eval-workflows", "../../evals/workflow-routing-suite.json", 0, true},
		{"workflows fail", "eval-workflows", badWorkflow, 2, false},
		{"quality pass", "eval-quality", "../../evals/synthetic-quality-pass.json", 0, true},
		{"quality fail", "eval-quality", "../../evals/synthetic-quality-fail.json", 2, false},
		{"triggers pass", "eval-triggers", "../../evals/holdout-3-trigger-suite.json", 0, true},
		{"triggers fail", "eval-triggers", badTriggers, 2, false},
		{"input error", "eval-workflows", filepath.Join(t.TempDir(), "absent.json"), 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestEvaluationCLIProcess$", "--", tc.command, "--fixture", tc.fixture)
			cmd.Env = append(os.Environ(), "JSDLC_TEST_EVAL_PROCESS=1")
			output, err := cmd.CombinedOutput()
			exit := 0
			if err != nil {
				ee, ok := err.(*exec.ExitError)
				if !ok {
					t.Fatal(err)
				}
				exit = ee.ExitCode()
			}
			if exit != tc.exit {
				t.Fatalf("exit=%d want=%d: %s", exit, tc.exit, output)
			}
			var report map[string]any
			if err := json.Unmarshal(output, &report); err != nil {
				t.Fatalf("lost JSON report: %v: %s", err, output)
			}
			if tc.exit == 1 {
				if report["error"] == nil {
					t.Fatal("missing input diagnostic")
				}
				return
			}
			if report["passed"] != tc.passed {
				t.Fatalf("wrong evaluation result: %#v", report)
			}
			if digest, ok := report["fixtureDigest"].(string); !ok || !validSHA256(digest) {
				t.Fatal("missing report provenance")
			}
		})
	}
}
