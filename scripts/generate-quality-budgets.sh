#!/usr/bin/env bash
# Generates docs/generated/quality-budgets.md as a readable view of
# scripts/quality-budgets.tsv. The TSV remains the single source of truth —
# scripts/check-quality-budgets.sh keeps reading it directly, unchanged. This
# generator only renders it; see docs/adr/0025-generated-documentation-anti-drift.md
# (UNC-02: TSV stays source, .md is a generated view, never the reverse).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
source scripts/lib-provenance.sh

TSV="scripts/quality-budgets.tsv"
OUT="docs/generated/quality-budgets.md"

{
  provenance_header "${TSV} (source of truth — unchanged by this generator)" "scripts/generate-quality-budgets.sh"
  echo
  echo "# Quality Budgets"
  echo
  echo "Per-file line-count exceptions enforced by \`make quality-budget-gate\`"
  echo "(\`scripts/check-quality-budgets.sh\`). This file is a generated view of"
  echo "\`${TSV}\` — edit the TSV, not this file."
  echo
  echo "| Path | Max Lines | Reason |"
  echo "|---|---:|---|"
  awk -F'\t' 'NF && $1 !~ /^#/ {
    gsub(/\|/, "\\|", $1); gsub(/\|/, "\\|", $3)
    printf "| `%s` | %s | %s |\n", $1, $2, $3
  }' "$TSV"
} >"$OUT"

echo "generate-quality-budgets: wrote $OUT"
