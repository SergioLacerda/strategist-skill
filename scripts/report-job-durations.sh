#!/usr/bin/env bash
# Writes a per-job duration/outcome report for one workflow run, plus the
# outcome of the non-blocking Windows full-suite step, so flakiness and the
# promotion criteria in docs/adr/0052-cicd-enforcement-policy.md (decision 5:
# 30 consecutive green runs, zero unexplained failures, duration in budget)
# are measured rather than guessed.
#
# Usage: report-job-durations.sh <run-id> [out-dir]
# Env:   GITHUB_REPOSITORY, GH_TOKEN (gh), GITHUB_STEP_SUMMARY (optional).
# Output: <out-dir>/jobs.json (raw jobs API) and <out-dir>/jobs.md (table).
set -euo pipefail

run_id="${1:-}"
out="${2:-ci-metrics}"
repo="${GITHUB_REPOSITORY:-SergioLacerda/strategist-skill}"
full_suite_step="Whole test tree (evidence, not yet blocking)"

[[ -n "$run_id" ]] || { echo "::error::usage: report-job-durations.sh <run-id> [out-dir]" >&2; exit 2; }
mkdir -p "$out"

gh api "repos/$repo/actions/runs/$run_id/jobs" --paginate > "$out/jobs.json"

python3 - "$out/jobs.json" "$out/jobs.md" "$full_suite_step" "$run_id" <<'PY'
import json, sys
from datetime import datetime

src, dst, step_name, run_id = sys.argv[1:5]

def parse(ts):
    return datetime.strptime(ts, "%Y-%m-%dT%H:%M:%SZ") if ts else None

# --paginate concatenates one JSON document per page; read them all.
raw = open(src, encoding="utf-8").read()
dec, idx, jobs = json.JSONDecoder(), 0, []
while idx < len(raw):
    while idx < len(raw) and raw[idx].isspace():
        idx += 1
    if idx >= len(raw):
        break
    doc, idx = dec.raw_decode(raw, idx)
    jobs.extend(doc.get("jobs", []))

rows, full_suite = [], "not-run"
for job in jobs:
    start, end = parse(job.get("started_at")), parse(job.get("completed_at"))
    secs = int((end - start).total_seconds()) if start and end else None
    rows.append((job["name"], job.get("conclusion") or job.get("status"), secs))
    for step in job.get("steps", []):
        if step.get("name") == step_name:
            full_suite = step.get("conclusion") or step.get("status")

lines = [f"### Job durations (run {run_id})", "", "| job | conclusion | seconds |", "|---|---|---:|"]
for name, concl, secs in sorted(rows):
    lines.append(f"| {name} | {concl} | {'' if secs is None else secs} |")
lines += ["", f"Windows full-suite step (continue-on-error): **{full_suite}**"]
open(dst, "w", encoding="utf-8").write("\n".join(lines) + "\n")
PY

cat "$out/jobs.md"
if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  cat "$out/jobs.md" >> "$GITHUB_STEP_SUMMARY"
fi
