package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDeliveryVerificationIncludesUncitedChecks(t *testing.T) {
	for _, scenario := range []string{"failed", "pending", "added", "removed", "malformed"} {
		t.Run(scenario, func(t *testing.T) {
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("JSDLC_PLUGIN_ROOT", filepath.Join(wd, "..", "..", "plugins", "jerry-sdlc"))
			repo := t.TempDir()
			started, err := deliveryStart([]string{"--repo", repo, "--candidate", "c", "--workflow", "bug-diagnosis"})
			if err != nil {
				t.Fatal(err)
			}
			run := started["run"].(deliveryRun)
			_, key, root, err := stateLocation(repo)
			if err != nil {
				t.Fatal(err)
			}
			report := filepath.Join(t.TempDir(), "report.json")
			if err := os.WriteFile(report, []byte(`{"disposition":"CLEAN","summary":"diagnosed","evidence":["source inspected"],"checkIds":[],"findings":[],"limitations":[]}`), 0600); err != nil {
				t.Fatal(err)
			}
			for _, role := range run.RequiredRoles {
				if _, err := deliveryRecordPass([]string{"--repo", repo, "--candidate", "c", "--role", role, "--file", report}); err != nil {
					t.Fatal(err)
				}
			}
			command := "/bin/true"
			if scenario == "failed" {
				command = "/bin/false"
			}
			checked, err := deliveryCheck([]string{"--repo", repo, "--candidate", "c", "--id", "uncited", "--authorized", "--", command})
			if err != nil {
				t.Fatal(err)
			}
			checkPath := checked["path"].(string)
			verify := func() result {
				t.Helper()
				got, err := deliveryVerify([]string{"--repo", repo, "--candidate", "c"})
				if err != nil {
					t.Fatal(err)
				}
				return got
			}
			if scenario == "added" || scenario == "removed" {
				if got := verify(); got["outcome"] != "COMPLETE" {
					t.Fatalf("expected complete: %#v", got)
				}
			}
			switch scenario {
			case "pending":
				path := filepath.Join(root, key, "delivery", "check-reservations", run.ID, "pending.json")
				if err := createCheckReservation(path, checkReservation{SchemaVersion: 1, RunID: run.ID, ID: "pending", OwnerPID: os.Getpid(), CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}); err != nil {
					t.Fatal(err)
				}
			case "added":
				e := checked["check"].(checkEvidence)
				e.ID = "extra"
				if err := persistCheckEvidence(filepath.Join(filepath.Dir(checkPath), "extra.json"), e); err != nil {
					t.Fatal(err)
				}
			case "removed":
				if err := os.Remove(checkPath); err != nil {
					t.Fatal(err)
				}
			case "malformed":
				if err := os.WriteFile(checkPath, []byte(`{}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if got := verify(); got["outcome"] != "BLOCKED" || got["releaseReadinessEffect"] != "NONE" {
				t.Fatalf("%s escaped verification: %#v", scenario, got)
			}
		})
	}
}
