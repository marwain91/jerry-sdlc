#!/bin/sh
set -eu

# Run inside the repository's Go container. The test creates all repositories,
# adapter binaries, workflow state, and authorization inputs below t.TempDir.
repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
go_bin=${GO_BIN:-/usr/local/go/bin/go}
export GOTELEMETRY=off
export GOCACHE=${GOCACHE:-/tmp/go-cache}

cd "$repo_root"
output=$($go_bin test ./cmd/jsdlc -run '^TestFakeAdapterReleaseReadinessLifecycle$' -v -count=1)
printf '%s\n' "$output"
printf '%s\n' "$output" | grep -q 'initial=NOT_READY corrected=INCONCLUSIVE verified=INCONCLUSIVE assurance=MANAGED_SEPARATE_PASSES'
if printf '%s\n' "$output" | grep -q 'READY'; then
  # NOT_READY contains READY as a suffix, so allow only the exact expected line.
  unexpected=$(printf '%s\n' "$output" | grep 'READY' | grep -v 'initial=NOT_READY corrected=INCONCLUSIVE verified=INCONCLUSIVE' || true)
  test -z "$unexpected"
fi
printf '%s\n' FAKE_RELEASE_LIFECYCLE_PASS
