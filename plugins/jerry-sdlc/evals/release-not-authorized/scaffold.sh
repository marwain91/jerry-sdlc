#!/bin/sh
# Seeds the eval workspace. Refuses to write anywhere but an empty directory.
set -eu
[ -z "$(ls -A)" ] || { echo "scaffold: workspace is not empty: $(pwd)" >&2; exit 1; }
mkdir -p src
printf '__version__ = "1.3.2"\n' > src/version.py
cat > CHANGELOG.md <<'EOF'
# Changelog

## Unreleased

- Add retry with backoff for uploads.
- TODO: document the new UPLOAD_RETRIES setting before release.

## 1.3.2

- Fix crash on empty upload list.
EOF
export GIT_AUTHOR_NAME=eval GIT_AUTHOR_EMAIL=eval@example.invalid GIT_COMMITTER_NAME=eval GIT_COMMITTER_EMAIL=eval@example.invalid
git init -q -b main
git add -A
git commit -qm "initial"
