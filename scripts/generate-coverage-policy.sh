#!/usr/bin/env bash
# Generates docs/generated/coverage-policy.md as a readable view of
# scripts/coverage-packages.tsv. The TSV remains the single source of truth —
# scripts/check-coverage-gate.sh keeps reading it directly, unchanged. This
# generator only renders it; see docs/adr/0025-generated-documentation-anti-drift.md
# (UNC-02: TSV stays source, .md is a generated view, never the reverse).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
source scripts/lib-provenance.sh

TSV="scripts/coverage-packages.tsv"
OUT="docs/generated/coverage-policy.md"

{
  provenance_header "${TSV} (source of truth — unchanged by this generator)" "scripts/generate-coverage-policy.sh"
  echo
  echo "# Coverage Policy"
  echo
  echo "Per-package minimum coverage thresholds enforced by \`make cover-gate\`"
  echo "(\`scripts/check-coverage-gate.sh\`). This file is a generated view of"
  echo "\`${TSV}\` — edit the TSV, not this file."
  echo
  echo "| Package | Minimum Coverage | Reason |"
  echo "|---|---:|---|"
  awk -F'\t' 'NF && $1 !~ /^#/ {
    gsub(/\|/, "\\|", $1); gsub(/\|/, "\\|", $3)
    printf "| `%s` | %s%% | %s |\n", $1, $2, $3
  }' "$TSV"
} >"$OUT"

echo "generate-coverage-policy: wrote $OUT"
