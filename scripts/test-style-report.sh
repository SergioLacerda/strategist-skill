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
# Normalized to absolute: GOCOVERDIR and -test.gocoverdir (integration row,
# below) are resolved by the Go runtime against the test binary's own
# working directory, not this script's cwd — a relative path here silently
# writes nowhere and the merge step below finds no files.
coverage_dir="$(cd "$coverage_dir" && pwd)"

row() {
  printf "%-15s %-20s %-14s %-10s %s\n" "$1" "$2" "$3" "$4" "$5"
}

row "STYLE" "COMMAND" "METRIC" "VALUE" "STATUS"

# --- unit -------------------------------------------------------------
profile="$coverage_dir/unit.out"
GOCACHE="$go_cache" go test -race -coverprofile="$profile" -coverpkg=./... ./... >/dev/null 2>&1 || true
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
# tests/integration drives most of its scenarios through a compiled
# strategist binary run as a subprocess (runStrategistCLI). Coverage inside
# that subprocess is invisible to `go test`'s own -coverprofile — it only
# instruments the test binary's own process. STRATEGIST_E2E_GOCOVERDIR +
# `-test.gocoverdir` route both the test binary's in-process coverage and
# every subprocess binary's coverage (the binary is built with -cover, see
# tests/integration/e2e_harness_test.go) into one GOCOVERDIR directory,
# merged below via `go tool covdata textfmt` into the profile this script
# reads. See docs/integration-coverage-gaps.md and
# .analysis/refined/20260805-integration-coverage-mapping/analysis.md for why
# this was previously undercounting.
profile="$coverage_dir/integration.out"
covdir="$coverage_dir/integration-covdata"
rm -rf "$covdir"
mkdir -p "$covdir"
STRATEGIST_E2E_GOCOVERDIR="$covdir" GOCACHE="$go_cache" go test -race -tags=integration \
  -coverpkg=./internal/... ./tests/integration/... -args -test.gocoverdir="$covdir" >/dev/null 2>&1 || true
rawprofile="$coverage_dir/integration.raw.out"
GOCACHE="$go_cache" go tool covdata textfmt -i="$covdir" -o="$rawprofile" 2>/dev/null || true
# The compiled strategist binary (tests/integration/e2e_harness_test.go) is
# instrumented for the whole main module, not just ./internal/... (see that
# file's comment on buildStrategistBinary) — filter back down to internal/...
# lines only, keeping the `mode:` header, so this row's scope matches its
# `-coverpkg=./internal/...` definition above.
awk 'NR==1 || /\/internal\//' "$rawprofile" > "$profile" 2>/dev/null || true
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
