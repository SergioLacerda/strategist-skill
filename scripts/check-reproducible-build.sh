#!/usr/bin/env bash
# Builds the CLI twice and fails if the bytes differ. Covers the plain build and,
# with REPRODUCIBLE_PAYLOAD=1, the standalone build that embeds the private
# runtime (fetches the pinned Node for the host by digest first).
set -euo pipefail

gocache="${1:-/tmp/go-build-cache}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

build_once() {
  local out="$1" tags="$2"
  GOCACHE="$gocache" CGO_ENABLED=0 go build \
    ${tags:+-tags "$tags"} \
    -trimpath \
    -ldflags="-s -w -X main.Version=reproducible-check" \
    -o "$out" \
    ./cmd/strategist
}

check_variant() {
  local label="$1" tags="$2"
  build_once "$tmpdir/$label-1" "$tags"
  build_once "$tmpdir/$label-2" "$tags"
  local sha1 sha2
  sha1="$(sha256sum "$tmpdir/$label-1" | awk '{print $1}')"
  sha2="$(sha256sum "$tmpdir/$label-2" | awk '{print $1}')"
  if [[ "$sha1" != "$sha2" ]]; then
    echo "::error::repeated deterministic builds produced different checksums ($label build)" >&2
    echo "first:  $sha1" >&2
    echo "second: $sha2" >&2
    exit 1
  fi
  echo "OK: repeated deterministic $label builds match ($sha1)"
}

check_variant plain ""

if [[ "${REPRODUCIBLE_PAYLOAD:-0}" == "1" ]]; then
  cd "$(dirname "$0")/.."
  py="$(command -v python3 || command -v python)"
  "$py" scripts/fetch-node-runtime.py --host >/dev/null
  check_variant payload "strategist_payload"
fi
