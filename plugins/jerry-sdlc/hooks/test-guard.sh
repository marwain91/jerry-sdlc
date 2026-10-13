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

# Plain reads only: no textconv, external diff, fsmonitor, or optional index writes.
export GIT_OPTIONAL_LOCKS=0
g() { git -c core.fsmonitor=false -c diff.external= "$@"; }
top=$(g rev-parse --show-toplevel 2>/dev/null) || exit 0

# State lives outside the repository, at an absolute, physically resolved path.
state_dir=${CLAUDE_PLUGIN_DATA:-${XDG_STATE_HOME:-$HOME/.local/state}/jerry-sdlc}
case "$state_dir" in /*) ;; *) exit 0 ;; esac
case "$state_dir/" in "$top"/*) exit 0 ;; esac
(umask 077 && mkdir -p "$state_dir") 2>/dev/null || exit 0
state_dir=$(cd -P "$state_dir" 2>/dev/null && pwd -P) || exit 0
top=$(cd -P "$top" && pwd -P) || exit 0
case "$state_dir/" in "$top"/*) exit 0 ;; esac
base_file=$state_dir/session-$session.base
seen_file=$state_dir/session-$session.seen
[ ! -L "$base_file" ] && [ ! -L "$seen_file" ] || exit 0
find "$state_dir" -name 'session-*' -mtime +30 -exec rm -f {} + 2>/dev/null

if [ "$mode" = start ]; then
  [ -s "$base_file" ] || g rev-parse -q --verify HEAD > "$base_file" 2>/dev/null
  exit 0
fi

# A stop hook already asked once this turn; let the turn end.
printf '%s' "$input" | tr -d '\n' | grep -q '"stop_hook_active"[[:space:]]*:[[:space:]]*true' && exit 0

base=$(cat "$base_file" 2>/dev/null)
[ -n "$base" ] || base=$(g rev-parse -q --verify HEAD 2>/dev/null) || exit 0
g cat-file -e "$base^{commit}" 2>/dev/null || exit 0
touch "$base_file" "$seen_file" 2>/dev/null

set -- ':(glob)**/test/**' ':(glob)**/tests/**' ':(glob)**/__tests__/**' \
  ':(glob)**/spec/**' ':(glob)**/test_*' ':(glob)**/*_test.*' \
  ':(glob)**/*.test.*' ':(glob)**/*.spec.*' ':(glob)**/*Test.*' ':(glob)**/*Tests.*'

diff=$(g diff --no-color --no-ext-diff --no-textconv -U0 "$base" -- "$@" 2>/dev/null) || exit 0
deleted=$(g diff --no-ext-diff --no-textconv --name-only --diff-filter=D "$base" -- "$@" 2>/dev/null)
[ -n "$diff$deleted" ] || exit 0

added=$(printf '%s\n' "$diff" | grep '^+' | grep -v '^+++')
removed=$(printf '%s\n' "$diff" | grep '^-' | grep -v '^---')

w='(^|[^A-Za-z0-9_])'
skip_re="\\.(skip|only)[.(]|${w}x(it|describe|test)\\(|pytest\\.mark\\.(skip|xfail)|pytest\\.(skip|xfail)\\(|unittest\\.skip|expectedFailure|${w}t\\.Skip(Now|f)?\\(|@Disabled|@Ignore|#\\[ignore\\]|markTestSkipped|this\\.skip\\(|\\[Ignore"
assert_re="${w}(assert[A-Za-z_]*!?|expect|verify|XCTAssert[A-Za-z]*)[[:space:]]*\\(|${w}(assert|require)\\.[A-Za-z]+\\(|${w}assert[[:space:]]|\\.should[A-Z.(]|${w}t\\.(Error|Fatal)f?\\("

# Exact checks pin a value; loose ones accept a range of outcomes. Removing an
# exact check while adding a loose one is the classic way to "fix" a flaky test.
exact_re="${w}(assert(Equals?|ListEqual|DictEqual|TupleEqual|SetEqual|MultiLineEqual)|assert_eq!|XCTAssertEqual)[[:space:]]*\\(|\\.(toBe|toEqual|toStrictEqual|isEqualTo)\\(|${w}(assert|require)\\.Equal(Values)?\\(|${w}assert[[:space:]][^#]*=="
loose_re="${w}(assert(In|True|IsNotNone|Greater|GreaterEqual|Less|LessEqual|Regex|AlmostEqual|IsInstance)|XCTAssert(True|NotNil))[[:space:]]*\\(|\\.(toBeTruthy|toBeDefined|toBeGreaterThan|toBeLessThan|toContain|toMatch|toBeCloseTo)\\(|expect\\.any(thing)?\\(|${w}(assert|require)\\.(True|NotNil|NotEmpty|Contains|Greater|Less|InDelta)\\(|${w}assert[[:space:]][^=#]*[[:space:]]in[[:space:]]|${w}assert[[:space:]]+True"

count() { printf '%s\n' "$1" | grep -Eo "$2" | wc -l | tr -d ' '; }
skip_lines=$(printf '%s\n' "$added" | grep -E "$skip_re")
skips=$(printf '%s' "$skip_lines" | grep -c .)
asserts_added=$(count "$added" "$assert_re")
asserts_removed=$(count "$removed" "$assert_re")
exact_removed=$(count "$removed" "$exact_re")
exact_added=$(count "$added" "$exact_re")
loose_added=$(count "$added" "$loose_re")
loose_removed=$(count "$removed" "$loose_re")

findings=
[ "$skips" -gt 0 ] && findings="$findings
- $skips added line(s) that skip, focus, or expect failure of tests"
[ "$asserts_removed" -gt "$asserts_added" ] && findings="$findings
- assertions removed: $asserts_removed, added: $asserts_added"
[ "$exact_removed" -gt "$exact_added" ] && [ "$loose_added" -gt "$loose_removed" ] && findings="$findings
- exact assertions replaced by looser ones (for example assertEqual by assertIn): exact removed $exact_removed, added $exact_added; loose added $loose_added"
[ -n "$deleted" ] && findings="$findings
- deleted test files: $(printf '%s' "$deleted" | tr '\n' ' ')"
[ -n "$findings" ] || exit 0

# Fingerprint the suspicious lines with their file names, not just the counts,
# and remember every fingerprint already raised in this session.
digest=$({ printf '%s\n' "$diff" | grep -E "^\\+\\+\\+ |$skip_re|$assert_re|$loose_re"; printf '%s\n' "$deleted"; } |
  g hash-object --stdin 2>/dev/null) || exit 0
[ -n "$digest" ] || exit 0
grep -qx "$digest" "$seen_file" 2>/dev/null && exit 0
printf '%s\n' "$digest" >> "$seen_file" 2>/dev/null || exit 0

cat >&2 <<EOF
Jerry SDLC test guard: since this session started (base ${base%"${base#???????}"}), the diff in test files shows:$findings

If this is intended (for example, a test was moved, replaced, or deleted with its feature), say so and why in your reply to the user. Otherwise restore the coverage. Never weaken a test to get a pass. This guard asks once per change.
EOF
exit 2
