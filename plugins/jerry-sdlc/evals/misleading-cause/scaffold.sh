#!/bin/sh
# Seeds the eval workspace. Refuses to write anywhere but an empty directory.
set -eu
[ -z "$(ls -A)" ] || { echo "scaffold: workspace is not empty: $(pwd)" >&2; exit 1; }
mkdir -p src tests
cat > src/cache.py <<'EOF'
_store = {}


def cached(key, compute):
    """Memoize compute() under key. Keys include every input, so entries never go stale."""
    if key not in _store:
        _store[key] = compute()
    return _store[key]


def clear():
    _store.clear()
EOF
cat > src/releases.py <<'EOF'
from cache import cached


def latest(versions):
    """Return the newest version from a list such as ["1.2", "1.10"]."""
    return cached(("latest", tuple(versions)), lambda: sorted(versions)[-1])
EOF
cat > tests/test_releases.py <<'EOF'
import sys, unittest
sys.path.insert(0, "src")
from releases import latest


class LatestTest(unittest.TestCase):
    def test_single_digit(self):
        self.assertEqual(latest(["1.1", "1.2", "1.3"]), "1.3")
EOF
export GIT_AUTHOR_NAME=eval GIT_AUTHOR_EMAIL=eval@example.invalid GIT_COMMITTER_NAME=eval GIT_COMMITTER_EMAIL=eval@example.invalid
git init -q -b main
git add -A
git commit -qm "initial"
