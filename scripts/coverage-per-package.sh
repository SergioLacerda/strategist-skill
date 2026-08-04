#!/usr/bin/env bash
set -euo pipefail

pkgs="${1:?space-separated package list required}"
profile="${2:?coverage profile path required}"
go_cache="${3:?GOCACHE required}"

for pkg in $pkgs; do
  echo "=== $pkg ==="
  GOCACHE="$go_cache" go test -race -coverprofile="$profile" -coverpkg="./$pkg/..." "./$pkg/..." 2>/dev/null
  go tool cover -func="$profile" | tail -1
done
