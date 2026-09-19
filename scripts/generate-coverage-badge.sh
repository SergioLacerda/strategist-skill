#!/usr/bin/env bash
# generate-coverage-badge.sh — generates coverage/coverage-badge.json (Shields.io endpoint schema)
# and coverage/badge.svg from Go coverage data.
set -euo pipefail

coverage_dir="${1:-coverage}"
go_cache="${2:-/tmp/go-build-cache}"

mkdir -p "$coverage_dir"

profile="$coverage_dir/coverage.out"
if [[ ! -f "$profile" ]]; then
  GOCACHE="$go_cache" go test -coverprofile="$profile" -coverpkg=./internal/... ./internal/... >/dev/null 2>&1 || true
fi

pct="$(GOCACHE="$go_cache" go tool cover -func="$profile" 2>/dev/null | tail -1 | grep -o '[0-9.]*%' | tail -1 | tr -d '%' || echo "0.0")"
if [[ -z "$pct" ]]; then
  pct="0.0"
fi

# Determine color based on coverage percentage
color="brightgreen"
pct_num="$(awk -v p="$pct" 'BEGIN{print int(p)}')"
if [[ "$pct_num" -ge 95 ]]; then
  color="brightgreen"
  color_hex="#4c1"
elif [[ "$pct_num" -ge 90 ]]; then
  color="green"
  color_hex="#97ca00"
elif [[ "$pct_num" -ge 80 ]]; then
  color="yellow"
  color_hex="#dfb317"
else
  color="red"
  color_hex="#e05d44"
fi

# 1. Output JSON endpoint (Shields.io schema)
cat <<EOF > "$coverage_dir/coverage-badge.json"
{
  "schemaVersion": 1,
  "label": "coverage",
  "message": "${pct}%",
  "color": "${color}"
}
EOF

# 2. Output standalone SVG badge
cat <<EOF > "$coverage_dir/badge.svg"
<svg xmlns="http://www.w3.org/2000/svg" width="104" height="20" role="img" aria-label="coverage: ${pct}%">
  <title>coverage: ${pct}%</title>
  <linearGradient id="s" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="r">
    <rect width="104" height="20" rx="3" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#r)">
    <rect width="61" height="20" fill="#555"/>
    <rect x="61" width="43" height="20" fill="${color_hex}"/>
    <rect width="104" height="20" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="110">
    <text aria-hidden="true" x="315" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="510">coverage</text>
    <text x="315" y="140" transform="scale(.1)" fill="#fff" textLength="510">coverage</text>
    <text aria-hidden="true" x="815" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="330">${pct}%</text>
    <text x="815" y="140" transform="scale(.1)" fill="#fff" textLength="330">${pct}%</text>
  </g>
</svg>
EOF

echo "generate-coverage-badge: generated $coverage_dir/coverage-badge.json and $coverage_dir/badge.svg (${pct}%)"
