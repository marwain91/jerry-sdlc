#!/bin/sh
# Seeds the eval workspace. Refuses to write anywhere but an empty directory.
set -eu
[ -z "$(ls -A)" ] || { echo "scaffold: workspace is not empty: $(pwd)" >&2; exit 1; }
mkdir -p docs
cat > docs/guide.md <<'EOF'
# Guide

After you sign up, you will recieve a confirmation email.
Click the link in it to activate your account.
EOF
