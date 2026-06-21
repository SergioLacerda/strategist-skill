#!/usr/bin/env sh
set -eu

ROOT=".analysis/refined"

if [ ! -d "$ROOT" ]; then
  echo "OK: $ROOT not found"
  exit 0
fi

violations=0

# Rule 1: no standalone markdown files directly under refined/
while IFS= read -r f; do
  [ -n "$f" ] || continue
  echo "FAIL: standalone file under refined/: $f"
  violations=1
done <<EOF
$(find "$ROOT" -maxdepth 1 -type f -name '*.md' | sort)
EOF

# Rule 2: each mission directory must contain analysis.md, proposal.md, design.md, tasks.md
for d in "$ROOT"/*; do
  [ -d "$d" ] || continue
  for required in analysis.md proposal.md design.md tasks.md; do
    if [ ! -f "$d/$required" ]; then
      echo "FAIL: missing $required in $d"
      violations=1
    fi
  done
done

if [ "$violations" -ne 0 ]; then
  exit 1
fi

echo "OK: refined structure valid"
