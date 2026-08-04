#!/usr/bin/env bash
set -euo pipefail

gocognit="${1:?gocognit binary path required}"
curdir="${2:?CURDIR required}"
threshold="${3:-15}"

echo "=== Cognitive Complexity > ${threshold} ==="
"$gocognit" -over "$threshold" ./cmd ./internal \
  | awk '{split($NF,a,":"); print a[1]}' \
  | sort -u \
  | sed "s|${curdir}/||" \
  || true

echo ""
"$gocognit" -over "$threshold" ./cmd ./internal \
  | sort -t' ' -k1 -rn \
  | sed "s|${curdir}/||" \
  || true
