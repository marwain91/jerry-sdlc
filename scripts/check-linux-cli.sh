#!/bin/sh
set -eu

# Run inside the repository's Go container. All state is disposable.
case "$(uname -s):$(uname -m)" in
  Linux:x86_64) ;;
  *) echo "requires Linux x86-64" >&2; exit 1 ;;
esac

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cli="$repo_root/plugins/jerry-sdlc/scripts/jsdlc"
go_bin=${GO_BIN:-/usr/local/go/bin/go}
test -x "$go_bin"
scratch=$(mktemp -d /tmp/jsdlc-linux-matrix.XXXXXX)
trap 'rm -rf "$scratch"' EXIT HUP INT TERM
fixture_repo="$scratch/repository"
state_home="$scratch/state"
mkdir -p "$fixture_repo" "$state_home"
printf '%s\n' fixture >"$fixture_repo/README.txt"
export XDG_STATE_HOME="$state_home"

test -r "$repo_root/plugins/jerry-sdlc/.codex-plugin/plugin.json"
test -r "$repo_root/plugins/jerry-sdlc/skills/jsdlc-orchestrate/SKILL.md"

# Success proves platform selection and checksum verification by the wrapper.
roles=$($cli roles --workflow release-readiness)
printf '%s\n' "$roles" | grep -q '"release-readiness"'

# Doctor must create writable external state and must not claim independent
# assurance. Some development containers expose a Codex executable; others do
# not, so either truthful local outcome is accepted.
doctor=$($cli doctor)
printf '%s\n' "$doctor" | grep -Eq '"outcome": "(MANAGED_SEPARATE_PASSES|UNAVAILABLE)"'
if printf '%s\n' "$doctor" | grep -q '"outcome": "MANAGED_SEPARATE_PASSES"'; then
  printf '%s\n' "$doctor" | grep -q '"independentWorkers": false'
  printf '%s\n' "$doctor" | grep -q '"readOnlyIsolation": false'
fi
test -d "$state_home/jsdlc"

started=$($cli start --repo "$fixture_repo" --candidate linux-matrix-a)
printf '%s\n' "$started" | grep -q '"state": "BASELINED"'
status=$($cli status --repo "$fixture_repo" --candidate linux-matrix-a)
printf '%s\n' "$status" | grep -q '"stale": false'
printf '%s\n' "$status" | grep -q '"candidate": "linux-matrix-a"'

collision="$scratch/collision.json"
printf '%s\n' '{"schemaVersion":1,"explicitSelection":"NONE","competingBroadOrchestrators":["fixture-orchestrator"],"discoveryEvidence":"CALLER_DECLARED"}' >"$collision"
resolved=$($go_bin run ./cmd/jsdlc resolve-collision --file "$collision")
printf '%s\n' "$resolved" | grep -q '"decision": "COLLISION"'
printf '%s\n' "$resolved" | grep -q '"assurance": "OWNER_GATE"'
printf '%s\n' "$resolved" | grep -q '"launchAllowed": false'

# Controlled legacy-state and interruption fixtures belong in focused tests;
# the public CLI intentionally cannot manufacture them.
cd "$repo_root"
$go_bin test ./cmd/jsdlc -run 'Test(StateUpgradeAndRollbackAreReversible|StateRollbackRefusesPostMigrationTransition|StateRollbackRejectsTamperedBackup|StateRollbackRejectsRepositoryMutation|InterruptedAtomicWriteLeavesActiveStateReadable)$' -count=1

printf '%s\n' LINUX_CLI_MATRIX_PASS
