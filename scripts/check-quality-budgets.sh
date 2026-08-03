#!/usr/bin/env bash
set -euo pipefail

manifest="${1:-scripts/quality-budgets.tsv}"
gocognit_bin="${2:-gocognit}"
complexity_threshold="${3:-15}"

if [[ ! -f "$manifest" ]]; then
  echo "::error::$manifest not found" >&2
  exit 1
fi

declare -A max_lines
declare -A seen

while IFS=$'\t' read -r path limit _reason; do
  [[ -z "${path:-}" || "${path:0:1}" == "#" ]] && continue
  if [[ ! "$limit" =~ ^[0-9]+$ ]]; then
    echo "::error::$manifest has invalid max_lines for $path: $limit" >&2
    exit 1
  fi
  max_lines["$path"]="$limit"
done < "$manifest"

fail=0

while IFS= read -r file; do
  lines=$(wc -l < "$file" | tr -d ' ')
  limit="${max_lines[$file]:-200}"
  if (( lines > limit )); then
    echo "::error::$file has $lines lines; budget is $limit" >&2
    fail=1
  fi
  seen["$file"]=1
done < <(find cmd internal -type f -name '*.go' \
  ! -name '*_test.go' \
  ! -path 'internal/embed/defaults/*' | sort)

for path in "${!max_lines[@]}"; do
  if [[ -z "${seen[$path]:-}" ]]; then
    echo "::error::$manifest contains stale budget for missing file: $path" >&2
    fail=1
  fi
done

if [[ ! -x "$gocognit_bin" ]]; then
  echo "::error::gocognit not found at $gocognit_bin" >&2
  exit 1
fi

complexity_output="$("$gocognit_bin" -over "$complexity_threshold" ./cmd ./internal || true)"
if [[ -n "$complexity_output" ]]; then
  echo "::error::cognitive complexity exceeds threshold $complexity_threshold" >&2
  printf '%s\n' "$complexity_output" >&2
  fail=1
fi

if (( fail != 0 )); then
  exit 1
fi

echo "OK: quality budgets valid"
