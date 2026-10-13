#!/bin/sh
# Jerry SDLC test guard (Claude Code hook).
#
#   start  record HEAD as this session's base commit
#   stop   compare the base with the working tree; if the diff adds skip
#          markers, removes more assertions than it adds, or deletes test
#          files, block the stop once and ask for a justification
#
# Read-only for the repository: it runs git diff and never runs tests.
# It blocks once per distinct finding, so a justified change does not
# repeat the request on every turn. Any internal error allows the stop.

mode=${1:-stop}
input=$(cat 2>/dev/null) || exit 0

field() {
  printf '%s' "$input" | tr -d '\n' | sed -n "s/.*\"$1\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p"
}

session=$(field session_id | tr -cd 'A-Za-z0-9_-')
[ -n "$session" ] || exit 0
repo=${CLAUDE_PROJECT_DIR:-$(field cwd)}
[ -n "$repo" ] && cd "$repo" 2>/dev/null || exit 0
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || exit 0

state_dir=${CLAUDE_PLUGIN_DATA:-${TMPDIR:-/tmp}/jerry-sdlc}
mkdir -p "$state_dir" 2>/dev/null || exit 0
base_file=$state_dir/session-$session.base
seen_file=$state_dir/session-$session.seen
find "$state_dir" -name 'session-*' -mtime +7 -exec rm -f {} + 2>/dev/null

if [ "$mode" = start ]; then
  [ -s "$base_file" ] || git rev-parse -q --verify HEAD > "$base_file" 2>/dev/null
  exit 0
fi

# A stop hook already asked once this turn; let the turn end.
printf '%s' "$input" | tr -d '\n' | grep -q '"stop_hook_active"[[:space:]]*:[[:space:]]*true' && exit 0

base=$(cat "$base_file" 2>/dev/null)
[ -n "$base" ] || base=$(git rev-parse -q --verify HEAD 2>/dev/null) || exit 0
git cat-file -e "$base^{commit}" 2>/dev/null || exit 0

set -- ':(glob)**/test/**' ':(glob)**/tests/**' ':(glob)**/__tests__/**' \
  ':(glob)**/spec/**' ':(glob)**/test_*' ':(glob)**/*_test.*' \
  ':(glob)**/*.test.*' ':(glob)**/*.spec.*' ':(glob)**/*Test.*' ':(glob)**/*Tests.*'

diff=$(git diff --no-color --no-ext-diff -U0 "$base" -- "$@" 2>/dev/null) || exit 0
deleted=$(git diff --no-ext-diff --name-only --diff-filter=D "$base" -- "$@" 2>/dev/null)
[ -n "$diff$deleted" ] || exit 0

added=$(printf '%s\n' "$diff" | grep '^+' | grep -v '^+++')
removed=$(printf '%s\n' "$diff" | grep '^-' | grep -v '^---')

skip_re='\.skip\(|(^|[^A-Za-z_])x(it|describe|test)\(|\.only\(|@pytest\.mark\.(skip|xfail)|pytest\.(skip|xfail)\(|unittest\.skip|expectedFailure|(^|[^A-Za-z_])t\.Skip(Now|f)?\(|@Disabled|@Ignore|#\[ignore\]|markTestSkipped|this\.skip\(|\[Ignore'
assert_re='assert|expect\(|\.should|(^|[^A-Za-z_])t\.(Error|Fatal)|require\.|verify\('

skips=$(printf '%s\n' "$added" | grep -Ec "$skip_re")
asserts_added=$(printf '%s\n' "$added" | grep -Eic "$assert_re")
asserts_removed=$(printf '%s\n' "$removed" | grep -Eic "$assert_re")

findings=
[ "$skips" -gt 0 ] && findings="$findings
- $skips added line(s) that skip, focus, or expect failure of tests"
[ "$asserts_removed" -gt "$asserts_added" ] && findings="$findings
- assertions removed: $asserts_removed, added: $asserts_added"
[ -n "$deleted" ] && findings="$findings
- deleted test files: $(printf '%s' "$deleted" | tr '\n' ' ')"
[ -n "$findings" ] || exit 0

digest=$(printf '%s' "$findings" | git hash-object --stdin 2>/dev/null)
[ "$(cat "$seen_file" 2>/dev/null)" = "$digest" ] && exit 0
printf '%s\n' "$digest" > "$seen_file" 2>/dev/null

cat >&2 <<EOF
Jerry SDLC test guard: since this session started (base ${base%"${base#???????}"}), the diff in test files shows:$findings

If this is intended (for example, a test was moved, replaced, or deleted with its feature), say so and why in your reply to the user. Otherwise restore the coverage. Never weaken a test to get a pass. This guard asks once per change.
EOF
exit 2
