#!/usr/bin/env bash
# test-style-report.sh — prints one status row per test style (unit, spec,
# integration, eval, eval-promptfoo, web), using the metric that actually fits
# each style: line coverage % where suites exercise code paths directly,
# scenario pass counts where they validate Gherkin/eval scenarios instead.
# See .analysis/refined/20260804-test-coverage-visibility-by-style/design.md.
set -euo pipefail

coverage_dir="${1:-coverage}"
go_cache="${2:-/tmp/go-build-cache}"

mkdir -p "$coverage_dir"

row() {
  printf "%-15s %-20s %-14s %-10s %s\n" "$1" "$2" "$3" "$4" "$5"
}

row "STYLE" "COMMAND" "METRIC" "VALUE" "STATUS"

# --- unit -------------------------------------------------------------
profile="$coverage_dir/unit.out"
GOCACHE="$go_cache" go test -race -coverprofile="$profile" -coverpkg=./... \
  $(GOCACHE="$go_cache" go list ./... | grep -v '/testutil') >/dev/null 2>&1 || true
pct="$(GOCACHE="$go_cache" go tool cover -func="$profile" 2>/dev/null | tail -1 | grep -o '[0-9.]*%' | tail -1)"
if [[ -n "${pct:-}" ]]; then
  row "unit" "make test" "line coverage" "$pct" "ok"
else
  row "unit" "make test" "line coverage" "n/a" "FAIL"
fi

# --- spec (Gherkin) -----------------------------------------------------
output="$(GOCACHE="$go_cache" go test -race -tags=spec -v ./tests/spec/... 2>&1 || true)"
pass=$(printf '%s\n' "$output" | grep -c '^--- PASS' || true)
fail=$(printf '%s\n' "$output" | grep -c '^--- FAIL' || true)
total=$((pass + fail))
status="ok"
[[ "$fail" -gt 0 ]] && status="FAIL"
[[ "$total" -eq 0 ]] && status="n/a"
row "spec" "make spec" "scenarios" "${pass}/${total}" "$status"

# --- integration ----------------------------------------------------------
profile="$coverage_dir/integration.out"
GOCACHE="$go_cache" go test -race -tags=integration -coverprofile="$profile" \
  -coverpkg=./internal/... ./tests/integration/... >/dev/null 2>&1 || true
pct="$(GOCACHE="$go_cache" go tool cover -func="$profile" 2>/dev/null | tail -1 | grep -o '[0-9.]*%' | tail -1)"
if [[ -n "${pct:-}" ]]; then
  row "integration" "make integration" "line coverage" "$pct" "ok"
else
  row "integration" "make integration" "line coverage" "n/a" "FAIL"
fi

# --- eval -------------------------------------------------------------
output="$(GOCACHE="$go_cache" go test -race -tags=eval -v ./tests/evals/... 2>&1 || true)"
pass=$(printf '%s\n' "$output" | grep -c '^--- PASS' || true)
fail=$(printf '%s\n' "$output" | grep -c '^--- FAIL' || true)
total=$((pass + fail))
status="ok"
[[ "$fail" -gt 0 ]] && status="FAIL"
[[ "$total" -eq 0 ]] && status="n/a"
row "eval" "make eval" "scenarios" "${pass}/${total}" "$status"

# --- eval-promptfoo -----------------------------------------------------
# Deliberately not run: standalone by design, requires a local LM Studio
# endpoint. See .analysis/archived/20260804-promptfoo-ci-adapter-adr.md.
row "eval-promptfoo" "make eval-promptfoo" "-" "-" "excluded (manual)"

# --- web ----------------------------------------------------------------
if [[ -d web/landing/node_modules ]]; then
  output="$(cd web/landing && npm run cover 2>&1 || true)"
  line="$(printf '%s\n' "$output" | grep -m1 '^All files')"
  pct="$(printf '%s\n' "$line" | awk -F'|' '{gsub(/ /,"",$5); print $5}')"
  if [[ -n "${pct:-}" ]]; then
    row "web" "make cover-web" "line coverage" "${pct}%" "ok"
  else
    row "web" "make cover-web" "line coverage" "n/a" "FAIL"
  fi
else
  row "web" "make cover-web" "line coverage" "n/a" "skipped (run make install-web)"
fi
