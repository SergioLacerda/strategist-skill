#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
reporter="$script_dir/report-job-durations.sh"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir "$tmp/bin"

# Two pages, as `gh api --paginate` prints them.
cat > "$tmp/pages.json" <<'JSON'
{"total_count":2,"jobs":[{"name":"lint","status":"completed","conclusion":"success","started_at":"2026-09-24T10:00:00Z","completed_at":"2026-09-24T10:03:30Z","steps":[]}]}
{"total_count":2,"jobs":[{"name":"test-windows","status":"completed","conclusion":"success","started_at":"2026-09-24T10:00:00Z","completed_at":"2026-09-24T10:10:00Z","steps":[{"name":"Whole test tree (evidence, not yet blocking)","status":"completed","conclusion":"failure"}]}]}
JSON
printf '#!/bin/sh\ncat "%s/pages.json"\n' "$tmp" > "$tmp/bin/gh"
chmod +x "$tmp/bin/gh"

PATH="$tmp/bin:$PATH" bash "$reporter" 123 "$tmp/out" >/dev/null

grep -q '| lint | success | 210 |' "$tmp/out/jobs.md" || { echo "FAIL: lint row missing or wrong" >&2; exit 1; }
grep -q '| test-windows | success | 600 |' "$tmp/out/jobs.md" || { echo "FAIL: windows row missing (pagination?)" >&2; exit 1; }
grep -q 'Windows full-suite step (continue-on-error): \*\*failure\*\*' "$tmp/out/jobs.md" || { echo "FAIL: full-suite outcome missing" >&2; exit 1; }
[[ -s "$tmp/out/jobs.json" ]] || { echo "FAIL: raw jobs.json not written" >&2; exit 1; }

if bash "$reporter" >/dev/null 2>&1; then echo "FAIL: a missing run id should fail" >&2; exit 1; fi

echo "OK: job duration report"
