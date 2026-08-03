#!/usr/bin/env bash
set -euo pipefail

violations=0

fail() {
  echo "FAIL: $*" >&2
  violations=1
}

require_contains() {
  local file="$1"
  local needle="$2"
  [[ -f "$file" ]] || {
    fail "$file is missing"
    return
  }
  grep -Fq "$needle" "$file" || fail "$file missing: $needle"
}

require_absent_regex() {
  local file="$1"
  local regex="$2"
  [[ -f "$file" ]] || {
    fail "$file is missing"
    return
  }
  if grep -Eq "$regex" "$file"; then
    fail "$file contains forbidden pattern: $regex"
  fi
}

for file in internal/embed/defaults/skill.yaml .strategist/skill.yaml; do
  [[ -f "$file" ]] || continue
  require_contains "$file" "discovery:"
  require_contains "$file" "contract: write_analysis"
  require_contains "$file" "refinement:"
  require_contains "$file" "execution:"
  require_contains "$file" "contract: controlled"
done

require_contains docs/adr/0005-slot-write-contracts.md 'The `write_pending` contract was discontinued.'
require_contains docs/configuration.md '| `discovery` | `write_analysis` |'
require_contains docs/configuration.md '| `refinement` | `write_analysis` |'
require_contains docs/configuration.md '| `execution` | `controlled` |'
require_contains docs/c4-diagrams.md '| Slot `discovery` (Ranger) | pluggable | `write_analysis` |'
require_contains tests/spec/specs/slot-contracts.feature "Roles: Ranger=write_analysis, Archivist=write_analysis, Sniper=controlled"

require_absent_regex docs/onboarding/readme-en.md 'CI-passing|version-1\.0'
require_contains docs/onboarding/readme-en.md 'actions/workflows/test.yml/badge.svg'
require_contains docs/onboarding/readme-en.md 'img.shields.io/github/v/release/SergioLacerda/strategist-skill'

for pkg in dojo governance integrity i18n runtimefs treasure validate; do
  require_contains docs/architecture.md "  ${pkg}/"
done

if [[ "$violations" -ne 0 ]]; then
  exit 1
fi

echo "OK: contract consistency valid"
