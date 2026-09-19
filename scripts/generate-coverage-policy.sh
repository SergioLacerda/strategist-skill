#!/usr/bin/env bash
# Generates docs/generated/coverage-policy.md as a readable view of
# scripts/coverage-packages.tsv and the reviewed exemption allowlist remain the
# policy sources; this generator only renders them.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
source scripts/lib-provenance.sh

TSV="scripts/coverage-packages.tsv"
EXEMPTIONS="scripts/coverage-exemptions.tsv"
OUT="docs/generated/coverage-policy.md"

{
  provenance_header "${TSV} and ${EXEMPTIONS} (policy sources — unchanged by this generator)" "scripts/generate-coverage-policy.sh"
  echo
  echo "# Coverage Policy"
  echo
  echo "Per-package minimum coverage thresholds enforced by \`make cover-gate\`"
  echo "(\`scripts/check-coverage-gate.sh\`). Production inventory is discovered"
  echo "from \`cmd/...\`, \`internal/...\`, and \`treasure-chest/...\`; test packages"
  echo "are outside this gate. Thresholds remain in \`${TSV}\`; reviewed exceptions"
  echo "are recorded in \`${EXEMPTIONS}\`. Edit those sources, not this file."
  echo
  echo "| Package | Minimum Coverage | Reason |"
  echo "|---|---:|---|"
  awk -F'\t' 'NF && $1 !~ /^#/ {
    gsub(/\|/, "\\|", $1); gsub(/\|/, "\\|", $3)
    printf "| `%s` | %s%% | %s |\n", $1, $2, $3
  }' "$TSV"
  echo
  echo "## Reviewed Exemptions"
  echo
  echo "These production packages are present in the inventory but intentionally have no threshold row."
  echo
  echo "| Package | Owner | Reason |"
  echo "|---|---|---|"
  awk -F'\t' 'NF && $1 !~ /^#/ {
    gsub(/\|/, "\\|", $1); gsub(/\|/, "\\|", $2); gsub(/\|/, "\\|", $3)
    printf "| `%s` | %s | %s |\n", $1, $2, $3
  }' "$EXEMPTIONS"
} >"$OUT"

echo "generate-coverage-policy: wrote $OUT"
