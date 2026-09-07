package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// This exercises the shipped wrapper/binary, not calls into the Go functions.
// Reports are synthetic; check commands execute real subprocesses in temp dirs.
func TestPackagedEverydayLifecycle(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("packaged lifecycle currently targets Linux amd64 only")
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	plugin := filepath.Clean(filepath.Join(wd, "..", "..", "plugins", "jerry-sdlc"))
	cli := filepath.Join(plugin, "scripts", "jsdlc")
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	callFor := func(t *testing.T, wantError bool, args ...string) map[string]any {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, cli, args...)
		output, err := cmd.CombinedOutput()
		if ctx.Err() != nil {
			t.Fatalf("CLI timed out: %v", args)
		}
		if (err != nil) != wantError {
			t.Fatalf("CLI %v: err=%v output=%s", args, err, output)
		}
		var got map[string]any
		if err := json.Unmarshal(output, &got); err != nil {
			t.Fatalf("CLI JSON: %v: %s", err, output)
		}
		return got
	}
	expectFor := func(t *testing.T, got map[string]any, key string, want any) {
		t.Helper()
		if got[key] != want {
			t.Fatalf("%s: want %v, got %#v", key, want, got)
		}
	}
	reportDir := t.TempDir()
	for _, workflow := range []string{"feature", "bug-fix", "bug-diagnosis", "pr-review", "trivial-change", "incident"} {
		t.Run(workflow, func(t *testing.T) {
			call := func(wantError bool, args ...string) map[string]any { t.Helper(); return callFor(t, wantError, args...) }
			expect := func(got map[string]any, key string, want any) { t.Helper(); expectFor(t, got, key, want) }
			repo := t.TempDir()
			candidateFile := filepath.Join(repo, "candidate.txt")
			if err := os.WriteFile(candidateFile, []byte("candidate\n"), 0600); err != nil {
				t.Fatal(err)
			}
			risk := "NORMAL"
			if workflow == "incident" {
				risk = "HIGH"
			}
			identity := []string{"--repo", repo, "--candidate", "fixture-1"}
			command := func(name string, extra ...string) map[string]any {
				t.Helper()
				args := append([]string{name}, identity...)
				return call(false, append(args, extra...)...)
			}
			command("delivery-start", "--workflow", workflow, "--risk", risk)
			expect(command("delivery-verify"), "outcome", "INCOMPLETE")
			expect(command("delivery-status"), "stale", false)
			// Actual command evidence. Two executions let QA and verifier cite
			// different receipts without pretending there are independent agents.
			for _, id := range []string{"qa", "verification"} {
				expect(command("delivery-check", "--id", id, "--authorized", "--", "/bin/sh", "-c", "test -s candidate.txt"), "reportedPass", true)
			}
			roles := call(false, "roles", "--workflow", workflow)["roles"].([]any)
			for _, roleValue := range roles {
				role := roleValue.(string)
				ids := []string{}
				if role == "qa-executor" {
					ids = []string{"qa"}
				}
				if role == "verifier" {
					ids = []string{"verification"}
				}
				report := map[string]any{"disposition": "CLEAN", "summary": "Synthetic lifecycle report", "evidence": []string{"Fixture candidate exists; shell checks executed"}, "checkIds": ids, "findings": []any{}, "limitations": []string{"Synthetic report, no model review"}}
				b, err := json.Marshal(report)
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(reportDir, workflow+"-"+role+".json")
				if err := os.WriteFile(path, b, 0600); err != nil {
					t.Fatal(err)
				}
				command("delivery-record", "--role", role, "--file", path, "--producer-mode", "SELF_REVIEW")
				duplicate := append([]string{"delivery-record"}, identity...)
				failure := call(true, append(duplicate, "--role", role, "--file", path, "--producer-mode", "SELF_REVIEW")...)
				message, _ := failure["error"].(string)
				if !strings.Contains(message, "append-only and already exists") {
					t.Fatalf("wrong duplicate failure: %#v", failure)
				}
			}
			complete := command("delivery-verify")
			expect(complete, "outcome", "COMPLETE")
			expect(complete, "releaseReadinessEffect", "NONE")
			expect(complete, "assurance", "LOCAL_UNATTESTED")
			digest, _ := complete["evidenceDigest"].(string)
			if !validSHA256(digest) {
				t.Fatalf("missing finalized digest: %#v", complete)
			}
			expect(command("delivery-verify"), "evidenceDigest", complete["evidenceDigest"])
			// Release state uses its own namespace and cannot consume these reports.
			command("start", "--workflow", "release-readiness")
			formal := command("verify")
			expect(formal, "verdict", "INCONCLUSIVE")
			if err := os.WriteFile(candidateFile, []byte("changed\n"), 0600); err != nil {
				t.Fatal(err)
			}
			expect(command("delivery-status"), "stale", true)
			expect(command("delivery-verify"), "outcome", "BLOCKED")
			// Terminal run is archived; new evidence must bind the new candidate.
			identity = []string{"--repo", repo, "--candidate", "fixture-2"}
			command("delivery-start", "--workflow", workflow, "--risk", risk)
			expect(command("delivery-check", "--id", "failure", "--authorized", "--", "/bin/false"), "reportedPass", false)
			expect(command("delivery-verify"), "outcome", "BLOCKED")
			command("delivery-cancel")
			t.Logf("%s: incomplete -> complete -> stable reverify -> stale blocked -> fresh uncited failure blocked", workflow)
		})
	}
	t.Run("interrupted-check", func(t *testing.T) {
		repo := t.TempDir()
		call := func(wantError bool, name string, extra ...string) map[string]any {
			t.Helper()
			args := []string{name, "--repo", repo, "--candidate", "recovery"}
			return callFor(t, wantError, append(args, extra...)...)
		}
		started := call(false, "delivery-start", "--workflow", "trivial-change")
		runID := started["run"].(map[string]any)["id"].(string)
		reservationPath := filepath.Join(filepath.Dir(started["path"].(string)), "check-reservations", runID, "interrupted.json")
		cmd := exec.Command(cli, "delivery-check", "--repo", repo, "--candidate", "recovery", "--id", "interrupted", "--authorized", "--", "/bin/sleep", "60")
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		var pgid int
		waited := false
		defer func() {
			if !waited {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
			if pgid > 0 {
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
			}
		}()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			b, err := os.ReadFile(reservationPath)
			if err == nil {
				var reservation struct {
					ProcessGroupID int `json:"processGroupId"`
				}
				if json.Unmarshal(b, &reservation) == nil && reservation.ProcessGroupID > 0 {
					pgid = reservation.ProcessGroupID
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
		if pgid == 0 {
			t.Fatal("command process identity did not become durable")
		}
		call(true, "delivery-recover-check", "--id", "interrupted", "--authorized")
		expectFor(t, call(false, "delivery-verify"), "outcome", "BLOCKED")
		// Simulate abrupt launcher loss, then stop its exact recorded process group.
		if err := cmd.Process.Kill(); err != nil {
			t.Fatal(err)
		}
		_ = cmd.Wait()
		waited = true
		if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
			t.Fatal(err)
		}
		deadline = time.Now().Add(3 * time.Second)
		for processGroupRunning(pgid) && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		if processGroupRunning(pgid) {
			t.Fatal("interrupted command group did not stop")
		}
		expectFor(t, call(false, "delivery-recover-check", "--id", "interrupted", "--authorized"), "recovered", true)
		expectFor(t, call(false, "delivery-check", "--id", "interrupted", "--authorized", "--", "/bin/true"), "reportedPass", true)
		call(false, "delivery-cancel")
		t.Log("live recovery refused; interrupted reservation recovered; same check ID reused successfully")
	})
	binary, err := os.ReadFile(filepath.Join(plugin, "assets", "bin", "linux-amd64", "jsdlc"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("PACKAGED_EVERYDAY_LIFECYCLE_PASS binary_sha256=%x reports=SYNTHETIC", sha256.Sum256(binary))
}
