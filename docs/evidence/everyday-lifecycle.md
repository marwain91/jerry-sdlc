# Packaged everyday lifecycle

The Linux amd64 integration test launches the shipped `scripts/jsdlc` wrapper for each command, including its binary checksum verification. It executes real shell checks and supplies explicitly synthetic role reports. It requires no Codex credentials, model calls, network access, or consuming-project initialization.

Run from the repository root in a disposable container:

```sh
podman run --rm -v "$PWD:/src:ro,z" -w /src \
  docker.io/library/golang:1.23 \
  go test ./cmd/jsdlc -run '^TestPackagedEverydayLifecycle$' -v -count=1
```

The test also runs through the existing CI `go test ./...` step. Other OS/architecture combinations skip it explicitly; it does not provide macOS or IDE/app evidence. All test repositories, state, and role inputs live in temporary directories. Role inputs stay outside the frozen candidates.

For feature, bug-fix, bug-diagnosis, PR-review, trivial-change, and incident workflows it checks:

- Missing reports give INCOMPLETE, and subsequent recording can continue.
- Real successful command executions support synthetic clean role reports.
- Duplicate role records are rejected specifically as append-only evidence.
- Completion has a nonempty evidence digest, local unattested assurance, and no release-readiness effect.
- Reverification reproduces the same digest.
- A separate fresh release run remains INCONCLUSIVE despite completed everyday evidence.
- Candidate edits become stale and BLOCKED; the next delivery run gets a fresh baseline.
- An uncited failed check blocks the new run, which can then be cancelled.

A separate interruption scenario starts a long-running check, observes its durable process-group identity, proves live recovery is refused, kills the exact launcher and recorded group, recovers the reservation, and successfully reuses its ID. Cleanup is restricted to fixture-owned processes and temporary directories.

Observed on 2026-09-07: all seven scenarios passed in approximately 4.2 seconds (excluding Go compilation), with marker `PACKAGED_EVERYDAY_LIFECYCLE_PASS` and Linux binary SHA-256 `9fd226e768d1f2eeb968899b502b741244fe7753f6b7e2d6b7ea25fa960f16ff`.

This validates packaged command integration. Synthetic reports do not establish review quality, agent independence, adequate project coverage, or incident recovery in production. The deferred model benchmark remains deferred.
