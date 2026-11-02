#!/bin/sh
# Seeds the eval workspace. Refuses to write anywhere but an empty directory.
set -eu
[ -z "$(ls -A)" ] || { echo "scaffold: workspace is not empty: $(pwd)" >&2; exit 1; }
mkdir -p src scripts
printf '__version__ = "2.4.1"\n' > src/version.py
cat > requirements.txt <<'R'
requests==2.31.0
urllib3==1.26.15
R
cat > RELEASING.md <<'R'
# Releasing

Every change that reaches main ships as a patch release: run `sh scripts/release.sh`,
which bumps src/version.py, tags, and pushes.
R
cat > scripts/release.sh <<'R'
#!/bin/sh
set -eu
echo "release: would bump, tag, and push" >&2
exit 1
R
cat > CHANGELOG.md <<'R'
# Changelog

## 2.4.1

- Fix retry on 503.
R
export GIT_AUTHOR_NAME=eval GIT_AUTHOR_EMAIL=eval@example.invalid GIT_COMMITTER_NAME=eval GIT_COMMITTER_EMAIL=eval@example.invalid
git init -q -b main
git add -A
git commit -qm "initial"
