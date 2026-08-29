#!/usr/bin/env bash
set -euo pipefail

pkgs="${1:?space-separated package list required}"
profile="${2:?coverage profile path required}"
go_cache="${3:?GOCACHE required}"
manifest="${4:-scripts/coverage-packages.tsv}"

# minimum_for looks up pkg's gate threshold from manifest (tab-separated:
# package<TAB>minimum<TAB>reason). Empty when pkg has no gate entry.
minimum_for() {
  awk -F'\t' -v p="$1" '$1 == p { print $2; exit }' "$manifest" 2>/dev/null
}

for pkg in $pkgs; do
  echo "=== $pkg ==="
  GOCACHE="$go_cache" go test -race -coverprofile="$profile" -coverpkg="./$pkg/..." "./$pkg/..." 2>/dev/null
  total_line="$(go tool cover -func="$profile" | tail -1)"
  pct="$(printf '%s\n' "$total_line" | grep -o '[0-9.]*%' | tail -1 | tr -d '%')"
  minimum="$(minimum_for "$pkg")"
  if [[ -n "$minimum" && -n "$pct" ]]; then
    marker="✅"
    ok="$(awk -v p="$pct" -v m="$minimum" 'BEGIN{print (p+0 >= m+0)}')"
    [[ "$ok" == "1" ]] || marker="❌"
    printf '%s  (gate: %s%% %s)\n' "$total_line" "$minimum" "$marker"
  else
    printf '%s\n' "$total_line"
  fi
done
