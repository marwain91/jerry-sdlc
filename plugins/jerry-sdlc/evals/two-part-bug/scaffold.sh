#!/bin/sh
# Seeds the eval workspace. Refuses to write anywhere but an empty directory.
set -eu
[ -z "$(ls -A)" ] || { echo "scaffold: workspace is not empty: $(pwd)" >&2; exit 1; }
mkdir -p app tests data
: > app/__init__.py
cat > data/customers.json <<'EOF'
[
  {"id": 1, "name": "Ada Lovelace", "city": "London"},
  {"id": 2, "name": "Acme, Inc.", "city": "Prague"},
  {"id": 3, "name": "Grace Hopper", "city": "New York"}
]
EOF
cat > app/export.py <<'EOF'
def to_csv(customers):
    lines = ["id,name,city"]
    for c in customers:
        lines.append(f'{c["id"]},{c["name"]},{c["city"]}')
    return "\n".join(lines) + "\n"
EOF
cat > app/importer.py <<'EOF'
def from_csv(text):
    rows = []
    for line in text.strip().split("\n")[1:]:
        parts = line.split(",")
        if len(parts) != 3:
            continue  # malformed row
        rows.append({"id": int(parts[0]), "name": parts[1], "city": parts[2]})
    return rows
EOF
cat > app/roundtrip.py <<'EOF'
import json

from app.export import to_csv
from app.importer import from_csv

with open("data/customers.json") as f:
    customers = json.load(f)
print(f"{len(from_csv(to_csv(customers)))} customers")
EOF
cat > tests/test_export.py <<'EOF'
import unittest
from app.export import to_csv


class ExportTest(unittest.TestCase):
    def test_header_and_row(self):
        self.assertEqual(to_csv([{"id": 1, "name": "Ada", "city": "London"}]), "id,name,city\n1,Ada,London\n")
EOF
export GIT_AUTHOR_NAME=eval GIT_AUTHOR_EMAIL=eval@example.invalid GIT_COMMITTER_NAME=eval GIT_COMMITTER_EMAIL=eval@example.invalid
git init -q -b main
git add -A
git commit -qm "initial"
