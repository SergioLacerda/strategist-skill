#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
checker="$script_dir/check-goreleaser-pin.sh"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

printf 'GORELEASER_VERSION  ?= v2.12.2\n' > "$tmp/Makefile"
workflow() {
  printf '      - name: Run goreleaser\n        uses: goreleaser/goreleaser-action@abc # v7\n        with:\n          version: "%s"\n' "$1" > "$tmp/release.yml"
}

workflow v2.12.2
bash "$checker" "$tmp/Makefile" "$tmp/release.yml" >/dev/null 2>&1 || { echo "FAIL: matching pins should pass" >&2; exit 1; }

workflow "~> v2"
if bash "$checker" "$tmp/Makefile" "$tmp/release.yml" >/dev/null 2>&1; then
  echo "FAIL: a floating release version should fail" >&2
  exit 1
fi

workflow v2.13.0
if bash "$checker" "$tmp/Makefile" "$tmp/release.yml" >/dev/null 2>&1; then
  echo "FAIL: drifted pins should fail" >&2
  exit 1
fi

echo "OK: goreleaser pin check"
