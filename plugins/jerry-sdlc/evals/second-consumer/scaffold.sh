#!/bin/sh
# Seeds the eval workspace. Refuses to write anywhere but an empty directory.
set -eu
[ -z "$(ls -A)" ] || { echo "scaffold: workspace is not empty: $(pwd)" >&2; exit 1; }
mkdir -p server web mobile tests
cat > README.md <<'EOF'
# Profiles

`server/api.py` serves the profile API. Two clients consume it: the web client in `web/` and the mobile app in `mobile/`.
EOF
cat > server/api.py <<'EOF'
def profile(user):
    """Profile payload returned by GET /api/profile."""
    return {"id": user["id"], "full_name": user["first"] + " " + user["last"]}
EOF
cat > web/profile.js <<'EOF'
export function render(data) {
  return `<h1>${data.full_name}</h1>`;
}
EOF
cat > mobile/profile.js <<'EOF'
export function title(data) {
  return data.name;
}
EOF
cat > tests/test_api.py <<'EOF'
import sys, unittest
sys.path.insert(0, "server")
from api import profile


class ProfileTest(unittest.TestCase):
    def test_full_name(self):
        self.assertEqual(profile({"id": 1, "first": "Ada", "last": "Lovelace"})["full_name"], "Ada Lovelace")
EOF
export GIT_AUTHOR_NAME=eval GIT_AUTHOR_EMAIL=eval@example.invalid GIT_COMMITTER_NAME=eval GIT_COMMITTER_EMAIL=eval@example.invalid
git init -q -b main
git add -A
git commit -qm "initial"
