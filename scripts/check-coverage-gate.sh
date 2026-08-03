#!/usr/bin/env bash
set -euo pipefail

manifest="${1:-scripts/coverage-packages.tsv}"
coverage_dir="${2:-coverage}"
go_cache="${3:-/tmp/go-build-cache}"

if [[ ! -f "$manifest" ]]; then
  echo "FAIL: coverage manifest not found: $manifest" >&2
  exit 1
fi

mkdir -p "$coverage_dir"
fail=0

while IFS=$'\t' read -r pkg minimum reason; do
  [[ -n "${pkg:-}" ]] || continue
  [[ "$pkg" != \#* ]] || continue
  if [[ -z "${minimum:-}" || ! "$minimum" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
    echo "FAIL: invalid minimum for $pkg in $manifest" >&2
    fail=1
    continue
  fi

  profile="$coverage_dir/$(echo "$pkg" | tr '/:' '__').out"
  output="$(GOCACHE="$go_cache" go test -coverprofile="$profile" -coverpkg="./$pkg/..." "./$pkg/..." 2>&1 || true)"
  pct="$(printf '%s\n' "$output" | grep -o '[0-9.]*%' | tail -1 | tr -d '%')"
  if [[ -z "$pct" ]]; then
    printf "%-30s FAIL no coverage result (%s)\n" "$pkg" "${reason:-no reason}"
    printf '%s\n' "$output"
    fail=1
    continue
  fi

  printf "%-30s %s%% >= %s%%  %s\n" "$pkg" "$pct" "$minimum" "${reason:-}"
  ok="$(awk -v p="$pct" -v m="$minimum" 'BEGIN{print (p+0 >= m+0)}')"
  if [[ "$ok" != "1" ]]; then
    echo "  FAIL: $pct% < $minimum%"
    fail=1
  fi
done < "$manifest"

exit "$fail"
