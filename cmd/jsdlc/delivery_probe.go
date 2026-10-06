package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"os/exec"
	"syscall"
	"time"
)

// Probes execute only explicitly authorized commands in the caller's project
// environment. They are bounded local observations, not a sandbox or verdict.
func deliveryProbe(name string, args []string) (result, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	authorized := fs.Bool("authorized", false, "exact command is authorized in this environment")
	timeout := fs.Duration("timeout", 30*time.Second, "total timeout, at most 60s")
	interval := fs.Duration("interval", 2*time.Second, "wait-check polling interval")
	job := fs.String("job", "", "stable job/process identity for wait-check")
	pending := fs.Int("pending-exit", 3, "exit code indicating the same job is queued or running")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	command := fs.Args()
	if !*authorized || len(command) == 0 || *timeout <= 0 || *timeout > 60*time.Second || *interval < 10*time.Millisecond || *interval > 15*time.Second || *pending <= 0 || *pending > 125 || name == "wait-check" && (*job == "" || len(*job) > 256) {
		return nil, errors.New("authorized command, bounded timeout/interval, and wait-check job identity are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	started := time.Now()
	attempts := 0
	exit := -1
	stdoutDigest, stderrDigest := "", ""
	outcome := "TIMED_OUT"
	for {
		attempts++
		cmd := exec.Command(command[0], command[1:]...)
		// A status command's descendant can retain its stdout/stderr pipes.
		// Bound pipe draining too, after the poll process has stopped.
		cmd.WaitDelay = min(*timeout, 100*time.Millisecond)
		cmd.Env, cmd.SysProcAttr = workerEnvironment(), &syscall.SysProcAttr{Setpgid: true}
		var stdout, stderr cappedBuffer
		stdout.limit, stderr.limit = 1024*1024, 1024*1024
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		waited := make(chan error, 1)
		go func() { waited <- cmd.Wait() }()
		var err error
		select {
		case err = <-waited:
		case <-ctx.Done():
			_ = terminateProcessGroup(cmd.Process.Pid)
			err = <-waited
		}
		if cleanupErr := terminateProcessGroup(cmd.Process.Pid); cleanupErr != nil {
			return nil, cleanupErr
		}
		h1, h2 := sha256.Sum256(stdout.Bytes()), sha256.Sum256(stderr.Bytes())
		stdoutDigest, stderrDigest = hex.EncodeToString(h1[:]), hex.EncodeToString(h2[:])
		exit = 0
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				exit = ee.ExitCode()
			} else {
				return nil, err
			}
		}
		if ctx.Err() != nil {
			break
		}
		if exit == 0 {
			outcome = "PASSED"
			break
		}
		if name == "preflight" || exit != *pending {
			outcome = "FAILED"
			break
		}
		timer := time.NewTimer(*interval)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
		}
		if ctx.Err() != nil {
			break
		}
	}
	return result{"outcome": outcome, "attempts": attempts, "job": *job, "lastExitStatus": exit, "elapsedMilliseconds": time.Since(started).Milliseconds(), "stdoutDigest": stdoutDigest, "stderrDigest": stderrDigest, "execution": "FULLY_PRIVILEGED_LOCAL_COMMAND", "releaseReadinessEffect": "NONE", "resumeSameJob": name == "wait-check" && outcome == "TIMED_OUT"}, nil
}
