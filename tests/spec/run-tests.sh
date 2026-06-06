#!/usr/bin/env bash
# Validates Strategist test fixtures and schemas.
#
# Checks:
#   1. Each fixture in fixtures/ is valid YAML with required fields
#   2. Each schema in .strategist/schemas/ is valid YAML
#   3. shellcheck passes on bootstrap.sh
#
# Exit code: 0 if all pass, 1 if any fail.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
FIXTURES_DIR="$SCRIPT_DIR/fixtures"

pass=0
fail=0

# ── helpers ───────────────────────────────────────────────────────────────────

ok()   { echo "  ✓ $1"; ((pass++)) || true; }
fail() { echo "  ✗ $1"; ((fail++)) || true; }

check_yaml() {
  local file="$1"
  if python3 -c "import sys, yaml; yaml.safe_load(open('$file'))" 2>/dev/null; then
    ok "valid YAML: $file"
  else
    fail "invalid YAML: $file"
  fi
}

# ── fixture validation ────────────────────────────────────────────────────────

echo ""
echo "=== Fixture validation ==="

REQUIRED_FIELDS="scenario expected_event"

for fixture in "$FIXTURES_DIR"/*.yaml; do
  [ -f "$fixture" ] || continue
  name="$(basename "$fixture")"

  # YAML validity
  if ! python3 -c "import yaml; yaml.safe_load(open('$fixture'))" 2>/dev/null; then
    fail "invalid YAML: $name"
    continue
  fi

  # Required fields
  for field in $REQUIRED_FIELDS; do
    value=$(python3 -c "import yaml; d=yaml.safe_load(open('$fixture')); print(d.get('$field',''))" 2>/dev/null)
    if [ -z "$value" ]; then
      fail "missing field '$field' in $name"
    else
      ok "$name: $field present"
    fi
  done
done

# ── schema validation ─────────────────────────────────────────────────────────

echo ""
echo "=== Schema validation ==="

SCHEMAS_DIR="$REPO_ROOT/.strategist/schemas"

if [ -d "$SCHEMAS_DIR" ]; then
  for schema in "$SCHEMAS_DIR"/*.yaml; do
    [ -f "$schema" ] || continue
    check_yaml "$schema"
  done
else
  echo "  (no schemas dir found at $SCHEMAS_DIR — skipping)"
fi

# ── shellcheck ────────────────────────────────────────────────────────────────

echo ""
echo "=== Shell script linting ==="

if command -v shellcheck >/dev/null 2>&1; then
  for script in "$REPO_ROOT/bootstrap.sh"; do
    if shellcheck "$script" 2>/dev/null; then
      ok "shellcheck: $(basename "$script")"
    else
      fail "shellcheck: $(basename "$script")"
    fi
  done
else
  echo "  (shellcheck not found — skipping)"
fi

# ── summary ───────────────────────────────────────────────────────────────────

echo ""
echo "=== Results: $pass passed, $fail failed ==="

[ "$fail" -eq 0 ]
