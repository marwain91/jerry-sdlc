#!/bin/sh
# Tests for plugins/jerry-sdlc/hooks/test-guard.sh. Needs git. Run from the repository root.
set -eu

guard=$(pwd)/plugins/jerry-sdlc/hooks/test-guard.sh
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export CLAUDE_PLUGIN_DATA="$work/state" CLAUDE_PROJECT_DIR="$work/repo"
export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@example.invalid GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@example.invalid

fail=0
n=0
run() { # run <mode> <session> [extra json] -> sets rc
  rc=0
  printf '{"session_id":"%s","cwd":"%s"%s}' "$2" "$work/repo" "${3:-}" | sh "$guard" "$1" 2>"$work/err" || rc=$?
}
expect() { # expect <rc> <name>
  n=$((n + 1))
  if [ "$rc" -eq "$1" ]; then echo "ok $n - $2"; else echo "not ok $n - $2 (exit $rc, want $1)"; cat "$work/err"; fail=1; fi
}
fresh() {
  rm -rf "$work/repo" "$work/state"
  mkdir -p "$work/repo/tests" "$work/repo/src"
  cd "$work/repo"
  git init -q
  printf 'def add(a, b):\n    return a + b\n' > src/calc.py
  printf 'def test_add():\n    assert add(1, 2) == 3\n    assert add(0, 0) == 0\n' > tests/test_calc.py
  printf 'def test_other():\n    assert True\n' > tests/test_other.py
  git add -A && git commit -qm init
  run start s1
}

fresh
printf 'def add(a, b):\n    return a - b\n' > src/calc.py
run stop s1; expect 0 "source-only change passes"

fresh
printf 'def test_add():\n    assert add(1, 2) == 3\n    assert add(0, 0) == 0\n    assert add(2, 2) == 4\n' > tests/test_calc.py
run stop s1; expect 0 "added assertion passes"

fresh
printf 'import pytest\n@pytest.mark.skip\ndef test_add():\n    assert add(1, 2) == 3\n    assert add(0, 0) == 0\n' > tests/test_calc.py
run stop s1; expect 2 "added skip marker blocks"
grep -q 'skip, focus' "$work/err" || { echo "not ok - reason names the skip"; fail=1; }
run stop s1 ',"stop_hook_active":true'; expect 0 "stop_hook_active allows the stop"
run stop s1; expect 0 "same finding does not block twice"

fresh
printf 'def test_add():\n    assert add(1, 2) == 3\n' > tests/test_calc.py
git commit -qam "drop assertion"
run stop s1; expect 2 "committed assertion removal blocks"

fresh
git rm -q tests/test_other.py
run stop s1; expect 2 "deleted test file blocks"

fresh
git mv tests/test_other.py tests/test_misc.py
run stop s1; expect 0 "renamed test file passes"

fresh
printf 'def test_add():\n    assert add(1, 2) == 3\n' > tests/test_calc.py
run stop s2; expect 2 "session without recorded base compares with HEAD"

fresh
printf 'def test_add():\n    assert add(1, 2) == 3\n' > tests/test_calc.py
git commit -qam "drop assertion"
run start s1
run stop s1; expect 2 "resumed session keeps its original base"

fresh
sed -i.bak 's/def test_add/def test_asserts_add/' tests/test_calc.py && rm -f tests/test_calc.py.bak
run stop s1; expect 0 "renaming a test that contains 'assert' in its name passes"

fresh
printf 'def test_add():\n    assert add(1, 2) == 3; assert add(0, 0) == 0\n' > tests/test_calc.py
git commit -qam "join" && run start s3
printf 'def test_add():\n    assert add(1, 2) == 3\n' > tests/test_calc.py
run stop s3; expect 2 "dropping one of two assertions on a line blocks"

fresh
printf 'test.each([1])("one", (n) => {\n  expect(n).toBe(1);\n});\n' > tests/one.test.js
git add -A && git commit -qm js && run start s4
sed -i.bak 's/test.each/test.skip.each/' tests/one.test.js && rm -f tests/one.test.js.bak
run stop s4; expect 2 "chained test.skip.each blocks"

fresh
printf 'import pytest\npytestmark = pytest.mark.skip(reason="later")\n' >> tests/test_other.py
run stop s1; expect 2 "module-level pytestmark skip blocks"
printf 'import unittest\n@unittest.skip("flaky")\ndef test_add():\n    assert add(1, 2) == 3\n    assert add(0, 0) == 0\n' > tests/test_calc.py
run stop s1; expect 2 "a different skip after an acknowledged one blocks again"
git checkout -q -- tests/test_calc.py
run stop s1; expect 0 "returning to an acknowledged finding does not block again"

rm -rf "$work/repo"; mkdir -p "$work/repo"
run stop s1; expect 0 "non-git directory passes"

[ "$fail" -eq 0 ] && echo "hook tests passed"
exit "$fail"
