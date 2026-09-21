#!/usr/bin/env bash
# check-coverage-docs-drift.sh — fails when the coverage numbers written in
# README.md (badge) and docs/test-styles.md differ from the measured values by
# more than COVERAGE_DOCS_TOLERANCE percentage points (default 1.0). Never
# rewrites anything; the fix is `make sync-test-styles-docs sync-readme-badge`.
set -euo pipefail

readme="${README_FILE:-README.md}"
doc_file="${TEST_STYLES_DOC:-docs/test-styles.md}"
coverage_dir="${1:-coverage}"
go_cache="${2:-/tmp/go-build-cache}"
tolerance="${COVERAGE_DOCS_TOLERANCE:-1.0}"
here="$(dirname "$0")"

pkg_cov() {
  local pkg="$1" profile
  profile="$coverage_dir/$(echo "$pkg" | tr '/:' '__').out"
  if [[ ! -f "$profile" ]]; then
    mkdir -p "$coverage_dir"
    GOCACHE="$go_cache" go test -coverprofile="$profile" -coverpkg="./$pkg" "./$pkg" >/dev/null 2>&1 || true
  fi
  GOCACHE="$go_cache" go tool cover -func="$profile" 2>/dev/null | tail -1 | grep -o '[0-9.]*%' | tail -1 | tr -d '%'
}

[[ -f "$coverage_dir/coverage-badge.json" ]] || bash "$here/generate-coverage-badge.sh" "$coverage_dir" "$go_cache" >/dev/null
badge_measured="$(grep -o '"message": *"[0-9.]*%' "$coverage_dir/coverage-badge.json" | grep -o '[0-9.]*' | head -1)"

fail=0
check() { # label documented measured
  local label="$1" documented="$2" measured="$3"
  if [[ -z "$documented" || -z "$measured" ]]; then
    echo "COVERAGE_DOCS_UNREADABLE $label: documented='${documented}' measured='${measured}'" >&2
    fail=1
    return
  fi
  if ! awk -v d="$documented" -v m="$measured" -v t="$tolerance" 'BEGIN{x=d-m; if(x<0)x=-x; exit !(x<=t)}'; then
    echo "COVERAGE_DOCS_DRIFT $label: documented ${documented}% vs measured ${measured}% (tolerance ${tolerance})" >&2
    fail=1
  fi
}

readme_pct="$(grep -o 'img\.shields\.io/badge/coverage-[0-9.]*%25' "$readme" | grep -o '[0-9.]*%25' | grep -o '^[0-9.]*' | head -1 || true)"
check "README coverage badge" "$readme_pct" "$badge_measured"

doc_triplet="$(grep -o 'measure [0-9.]*%/[0-9.]*%/[0-9.]*%*' "$doc_file" | head -1 | grep -o '[0-9.]*%' | tr -d '%' | tr '\n' ' ' || true)"
read -r d_eval d_integrity d_runtimefs <<<"$doc_triplet"
d_treasure="$(grep -o 'and measures [0-9.]*%' "$doc_file" | head -1 | grep -o '[0-9.]*' | head -1 || true)"
check "test-styles internal/eval" "${d_eval:-}" "$(pkg_cov internal/eval)"
check "test-styles internal/integrity" "${d_integrity:-}" "$(pkg_cov internal/integrity)"
check "test-styles internal/runtimefs" "${d_runtimefs:-}" "$(pkg_cov internal/runtimefs)"
check "test-styles treasure-chest" "${d_treasure:-}" "$(pkg_cov treasure-chest)"

if (( fail != 0 )); then
  echo "coverage docs drifted — run: make sync-test-styles-docs sync-readme-badge" >&2
  exit 1
fi
echo "coverage docs in sync (tolerance ${tolerance} points)"
