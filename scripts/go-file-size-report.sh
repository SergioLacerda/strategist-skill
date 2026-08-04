#!/usr/bin/env bash
set -euo pipefail

echo "=== Go Files > 200 Lines ==="

files="$(find cmd internal -type f -name '*.go' \
  ! -name '*_test.go' \
  ! -path 'internal/embed/defaults/*' | sort)"

results="$(for f in $files; do
  lines="$(wc -l < "$f" | tr -d ' ')"
  if [[ "$lines" -gt 200 ]]; then printf "%s %s\n" "$f" "$lines"; fi
done | sort -k2,2nr -k1,1)"

if [[ -n "$results" ]]; then printf "%s\n" "$results"; else echo "none"; fi
