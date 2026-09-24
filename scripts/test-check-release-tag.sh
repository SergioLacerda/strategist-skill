#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
checker="$script_dir/check-release-tag.sh"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
repo="$tmp/repo"
mkdir "$repo"

g() { git -C "$repo" -c user.name=test -c user.email=test@example.invalid -c commit.gpgsign=false -c tag.gpgsign=false "$@"; }

g init -q -b main
g commit -q --allow-empty -m main-1
g tag -a v1.0.0 -m "annotated on main"
g tag v1.0.1
g commit -q --allow-empty -m main-2
g tag -a v1.1.0 -m "annotated on main tip"
g checkout -q -b side
g commit -q --allow-empty -m side-1
g tag -a v2.0.0 -m "annotated off main"
g tag -a not-a-release -m "bad name"
g checkout -q main

expect_success() {
  if ! bash "$checker" "$1" main "$repo" >/dev/null 2>&1; then
    printf 'FAIL: expected release tag to pass: %s\n' "$1" >&2
    exit 1
  fi
}

expect_failure() {
  if bash "$checker" "$1" main "$repo" >/dev/null 2>&1; then
    printf 'FAIL: expected release tag to fail: %s (%s)\n' "$1" "$2" >&2
    exit 1
  fi
}

expect_success v1.0.0
expect_success v1.1.0
expect_failure v1.0.1 "lightweight"
expect_failure v2.0.0 "not reachable from main"
expect_failure not-a-release "bad form"
expect_failure v9.9.9 "missing tag"

if bash "$checker" v1.0.0 no-such-ref "$repo" >/dev/null 2>&1; then
  echo "FAIL: expected a missing main ref to fail" >&2
  exit 1
fi

echo "OK: release tag check"
