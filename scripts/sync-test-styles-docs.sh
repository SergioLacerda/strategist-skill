#!/usr/bin/env bash
# sync-test-styles-docs.sh — synchronizes live gated package coverage metrics into docs/test-styles.md
set -euo pipefail

doc_file="${TEST_STYLES_DOC:-docs/test-styles.md}"
coverage_dir="${1:-coverage}"
go_cache="${2:-/tmp/go-build-cache}"

if [[ ! -f "$doc_file" ]]; then
  echo "FAIL: $doc_file not found" >&2
  exit 1
fi

pkg_cov() {
  local pkg="$1"
  local profile
  profile="$coverage_dir/$(echo "$pkg" | tr '/:' '__').out"
  if [[ ! -f "$profile" ]]; then
    GOCACHE="$go_cache" go test -coverprofile="$profile" -coverpkg="./$pkg" "./$pkg" >/dev/null 2>&1 || true
  fi
  local pct
  pct="$(GOCACHE="$go_cache" go tool cover -func="$profile" 2>/dev/null | tail -1 | grep -o '[0-9.]*%' | tail -1 || echo "0.0%")"
  echo "$pct"
}

eval_cov="$(pkg_cov "internal/eval")"
integrity_cov="$(pkg_cov "internal/integrity")"
runtimefs_cov="$(pkg_cov "internal/runtimefs")"
treasure_cov="$(pkg_cov "treasure-chest")"

# Replace the numbers in docs/test-styles.md if they differ. Each pattern consumes
# the full previously-written value (including any trailing "%") so re-running the
# sync is idempotent instead of appending another "%" every time.
today="$(date +%Y-%m-%d)"
sed -i -E "s/measure [0-9.]+%\\/[0-9.]+%\\/[0-9.]+%*/measure ${eval_cov}\\/${integrity_cov}\\/${runtimefs_cov}/g" "$doc_file"
sed -i -E "s/and measures [0-9.]+%+ —/and measures ${treasure_cov} —/g" "$doc_file"
sed -i -E "s/\\*\\*Last Updated:\\*\\* [0-9]{4}-[0-9]{2}-[0-9]{2}/**Last Updated:** ${today}/g" "$doc_file"

echo "sync-test-styles-docs: updated $doc_file (eval: $eval_cov, integrity: $integrity_cov, runtimefs: $runtimefs_cov, treasure: $treasure_cov)"
