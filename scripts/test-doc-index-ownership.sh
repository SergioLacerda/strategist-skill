#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
checker="$repo_root/scripts/check-doc-index-ownership.sh"
fixtures="$repo_root/tests/fixtures/docs-index-ownership"

expect_success() {
  local name=$1
  if ! bash "$checker" "$fixtures/$name" >/dev/null; then
    printf 'FAIL: expected docs index fixture to pass: %s\n' "$name" >&2
    exit 1
  fi
}

expect_failure() {
  local name=$1
  if bash "$checker" "$fixtures/$name" >/dev/null 2>&1; then
    printf 'FAIL: expected docs index fixture to fail: %s\n' "$name" >&2
    exit 1
  fi
}

expect_success valid
expect_success excluded-indexes
expect_failure unindexed
expect_failure duplicate-purpose

echo "OK: docs index ownership fixtures"
