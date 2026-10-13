#!/bin/sh
# Structural checks for the plugin. Run from the repository root.
set -eu

fail=0
err() { echo "FAIL: $*" >&2; fail=1; }

for f in $(find . -name '*.json' -not -path './.git/*'); do
  python3 -m json.tool "$f" >/dev/null 2>&1 || err "invalid JSON: $f"
done

skill=plugins/jerry-sdlc/skills/jsdlc-orchestrate/SKILL.md
head -n 1 "$skill" | grep -qx -- '---' || err "$skill: missing front matter"
grep -q '^name: jsdlc-orchestrate$' "$skill" || err "$skill: missing name"
grep -q '^description: .\{20,\}' "$skill" || err "$skill: missing description"
words=$(wc -w < "$skill")
[ "$words" -le 700 ] || err "$skill: $words words, limit 700"

# version_check: Codex and Claude Code manifests must carry the same base version.
codex_v=$(python3 -c 'import json;print(json.load(open("plugins/jerry-sdlc/.codex-plugin/plugin.json"))["version"].split("+")[0])')
claude_v=$(python3 -c 'import json;print(json.load(open("plugins/jerry-sdlc/.claude-plugin/plugin.json"))["version"])')
[ "$codex_v" = "$claude_v" ] || err "manifest versions differ: codex $codex_v, claude $claude_v"

# Every relative Markdown link must resolve.
for f in $(find . -name '*.md' -not -path './.git/*'); do
  dir=$(dirname "$f")
  for target in $(grep -o '](\([^)#]*\)' "$f" | sed 's/^](//' | grep -v '^[a-z]*://' || true); do
    [ -z "$target" ] && continue
    [ -e "$dir/$target" ] || err "$f: broken link $target"
  done
done

# Every eval case has a prompt, at least one grader, and its scaffold script.
for case in plugins/jerry-sdlc/evals/*/; do
  [ -f "$case/prompt.md" ] || err "$case: missing prompt.md"
  ls "$case"graders/*.md >/dev/null 2>&1 || err "$case: no graders"
  scaffold=$(sed -n 's/^  scaffold_script: //p' "$case/case.yaml" 2>/dev/null)
  [ -z "$scaffold" ] || [ -f "$case$scaffold" ] || err "$case: missing $scaffold"
done

for f in $(find . -name '*.sh' -not -path './.git/*'); do
  sh -n "$f" || err "shell syntax: $f"
done

if command -v git >/dev/null 2>&1; then
  sh scripts/test-hook.sh >/dev/null || { sh scripts/test-hook.sh | grep '^not ok' >&2; err "hook tests failed"; }
else
  echo "SKIP: hook tests need git" >&2
  [ -z "${CI:-}" ] || err "hook tests must run in CI"
fi

[ "$fail" -eq 0 ] && echo "all checks passed"
exit "$fail"
