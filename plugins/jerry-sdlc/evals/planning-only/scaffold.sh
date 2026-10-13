#!/bin/sh
# Seeds the eval workspace. Refuses to write anywhere but an empty directory.
set -eu
[ -z "$(ls -A)" ] || { echo "scaffold: workspace is not empty: $(pwd)" >&2; exit 1; }
mkdir -p src
cat > src/report.py <<'EOF'
import json
import sys


def build_rows(orders):
    return [{"id": o["id"], "total": o["total"], "customer": o["customer"]} for o in orders]


def main(path):
    with open(path) as f:
        orders = json.load(f)
    for row in build_rows(orders):
        print(f'{row["id"]}\t{row["total"]}\t{row["customer"]}')


if __name__ == "__main__":
    main(sys.argv[1])
EOF
