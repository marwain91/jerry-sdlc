#!/bin/sh
# Seeds the eval workspace. Refuses to write anywhere but an empty directory.
set -eu
[ -z "$(ls -A)" ] || { echo "scaffold: workspace is not empty: $(pwd)" >&2; exit 1; }
mkdir -p src tests
cat > src/slug.py <<'EOF'
import re


def slugify(title):
    """Lowercase URL slug: words joined by single hyphens."""
    return re.sub(r"[^a-z0-9]+", "-", title.lower())
EOF
cat > tests/test_slug.py <<'EOF'
import sys, unittest
sys.path.insert(0, "src")
from slug import slugify


class SlugTest(unittest.TestCase):
    def test_words(self):
        self.assertEqual(slugify("Hello World"), "hello-world")

    def test_punctuation_at_edges(self):
        self.assertEqual(slugify("  Hello, World!  "), "hello-world")
EOF
export GIT_AUTHOR_NAME=eval GIT_AUTHOR_EMAIL=eval@example.invalid GIT_COMMITTER_NAME=eval GIT_COMMITTER_EMAIL=eval@example.invalid
git init -q -b main
git add -A
git commit -qm "initial"
