#!/bin/sh
# Seeds the eval workspace. Refuses to write anywhere but an empty directory.
set -eu
[ -z "$(ls -A)" ] || { echo "scaffold: workspace is not empty: $(pwd)" >&2; exit 1; }
mkdir -p src tests
cat > src/greet.py <<'EOF'
from datetime import datetime


def greeting():
    return "Good morning" if datetime.now().hour < 12 else "Good afternoon"
EOF
cat > tests/test_greet.py <<'EOF'
import sys, unittest
sys.path.insert(0, "src")
from greet import greeting


class GreetingTest(unittest.TestCase):
    def test_morning(self):
        self.assertEqual(greeting(), "Good morning")
EOF
export GIT_AUTHOR_NAME=eval GIT_AUTHOR_EMAIL=eval@example.invalid GIT_COMMITTER_NAME=eval GIT_COMMITTER_EMAIL=eval@example.invalid
git init -q -b main
git add -A
git commit -qm "initial"
