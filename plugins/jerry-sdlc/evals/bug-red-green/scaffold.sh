#!/bin/sh
# Seeds the eval workspace. Refuses to write anywhere but an empty directory.
set -eu
[ -z "$(ls -A)" ] || { echo "scaffold: workspace is not empty: $(pwd)" >&2; exit 1; }
mkdir -p src tests
cat > src/pages.py <<'EOF'
def page_count(total, size):
    """Number of pages needed to show `total` items, `size` per page."""
    if size <= 0:
        raise ValueError("size must be positive")
    return total // size
EOF
cat > tests/test_pages.py <<'EOF'
import sys, unittest
sys.path.insert(0, "src")
from pages import page_count


class PageCountTest(unittest.TestCase):
    def test_exact_multiple(self):
        self.assertEqual(page_count(30, 10), 3)

    def test_rejects_zero_size(self):
        with self.assertRaises(ValueError):
            page_count(5, 0)
EOF
export GIT_AUTHOR_NAME=eval GIT_AUTHOR_EMAIL=eval@example.invalid GIT_COMMITTER_NAME=eval GIT_COMMITTER_EMAIL=eval@example.invalid
git init -q -b main
git add -A
git commit -qm "initial"
