#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
checker="$script_dir/check-release-binaries.sh"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

case "$(uname -s)" in Linux) os=linux ;; Darwin) os=darwin ;; *) echo "OK: skipped (unsupported host)"; exit 0 ;; esac
case "$(uname -m)" in x86_64|amd64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) echo "OK: skipped (unsupported arch)"; exit 0 ;; esac
host="strategist-$os-$arch"

# build_fixture <dir> <version-line> [runtime-line]
build_fixture() {
  local dir=$1 version=$2 runtime=${3:-"runtime: embedded OpenSpec 1.13.0"}
  mkdir -p "$dir"
  printf '#!/bin/sh\nprintf "%%s\\n" "%s" "commit: abc" "%s"\n' "$version" "$runtime" > "$dir/$host"
  chmod +x "$dir/$host"
  printf 'other-binary' > "$dir/strategist-other"
  printf '%s/%s\t%s\n%s/strategist-other\tstrategist-other\n' "$dir" "$host" "$host" "$dir" > "$dir/published.tsv"
  (cd "$dir" && sha256sum "$host" strategist-other > SHA256SUMS)
}

run() { bash "$checker" "$1/published.tsv" "$1/SHA256SUMS" "${2:-}" >/dev/null 2>&1; }

build_fixture "$tmp/ok" "V1.2.3"
run "$tmp/ok" || { echo "FAIL: valid release should pass" >&2; exit 1; }
run "$tmp/ok" "1.2.3" || { echo "FAIL: matching expected version should pass" >&2; exit 1; }
if run "$tmp/ok" "1.2.4"; then echo "FAIL: version mismatch should fail" >&2; exit 1; fi

build_fixture "$tmp/snapshot" "Vdev"
run "$tmp/snapshot" || { echo "FAIL: a snapshot 'Vdev' should pass without an expected version" >&2; exit 1; }

build_fixture "$tmp/badsum" "V1.2.3"
printf 'tampered' >> "$tmp/badsum/strategist-other"
if run "$tmp/badsum"; then echo "FAIL: digest mismatch should fail" >&2; exit 1; fi

build_fixture "$tmp/missing" "V1.2.3"
grep -v strategist-other "$tmp/missing/SHA256SUMS" > "$tmp/missing/S" && mv "$tmp/missing/S" "$tmp/missing/SHA256SUMS"
if run "$tmp/missing"; then echo "FAIL: asset without a checksum line should fail" >&2; exit 1; fi

build_fixture "$tmp/extra" "V1.2.3"
printf '%s  strategist-ghost\n' "$(printf x | sha256sum | awk '{print $1}')" >> "$tmp/extra/SHA256SUMS"
if run "$tmp/extra"; then echo "FAIL: checksum for an unpublished asset should fail" >&2; exit 1; fi

build_fixture "$tmp/noruntime" "V1.2.3" "runtime: none"
if run "$tmp/noruntime"; then echo "FAIL: a binary without the embedded runtime should fail" >&2; exit 1; fi

build_fixture "$tmp/badversion" "not-a-version"
if run "$tmp/badversion"; then echo "FAIL: a non-version first line should fail" >&2; exit 1; fi

echo "OK: release binaries check"
