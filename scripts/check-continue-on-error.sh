#!/usr/bin/env bash
# Every `continue-on-error: true` in a workflow must carry a catalog comment
# within the 12 lines above it:
#
#   # continue-on-error-catalog: owner=<who> reason=<why> remove-when=<condition>
#
# so no failure is masked without an owner, a reason and a removal condition.
# Usage: check-continue-on-error.sh [workflows-dir]   (default .github/workflows)
set -euo pipefail

dir="${1:-.github/workflows}"
window=12
failures=0
found=0

shopt -s nullglob
for file in "$dir"/*.yml "$dir"/*.yaml; do
  lineno=0
  catalog_line=0
  while IFS= read -r line || [[ -n "$line" ]]; do
    lineno=$((lineno + 1))
    if [[ "$line" =~ ^[[:space:]]*#[[:space:]]*continue-on-error-catalog:[[:space:]]*owner=[^[:space:]]+[[:space:]]+reason=.+[[:space:]]remove-when=.+ ]]; then
      catalog_line=$lineno
      continue
    fi
    if [[ "$line" =~ ^[[:space:]]*(-[[:space:]]+)?continue-on-error:[[:space:]]*true([[:space:]]|$) ]]; then
      found=$((found + 1))
      if [[ "$catalog_line" -eq 0 || $((lineno - catalog_line)) -gt "$window" ]]; then
        echo "::error file=$file,line=$lineno::continue-on-error: true without a 'continue-on-error-catalog: owner=... reason=... remove-when=...' comment in the $window lines above" >&2
        failures=$((failures + 1))
      fi
    fi
  done < "$file"
done

if [[ "$failures" -gt 0 ]]; then
  exit 1
fi
echo "continue-on-error catalog ok ($found entry/entries)"
