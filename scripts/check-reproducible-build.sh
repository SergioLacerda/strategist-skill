#!/usr/bin/env bash
set -euo pipefail

gocache="${1:-/tmp/go-build-cache}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

build_once() {
  local out="$1"
  GOCACHE="$gocache" CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X main.Version=reproducible-check" \
    -o "$out" \
    ./cmd/strategist
}

build_once "$tmpdir/strategist-1"
build_once "$tmpdir/strategist-2"

sha1="$(sha256sum "$tmpdir/strategist-1" | awk '{print $1}')"
sha2="$(sha256sum "$tmpdir/strategist-2" | awk '{print $1}')"

if [[ "$sha1" != "$sha2" ]]; then
  echo "::error::repeated deterministic builds produced different checksums" >&2
  echo "first:  $sha1" >&2
  echo "second: $sha2" >&2
  exit 1
fi

echo "OK: repeated deterministic builds match ($sha1)"
