#!/usr/bin/env bash
# Generates docs/generated/eval-scenarios.md from eval.Scenario{ID, Description}
# struct literals in tests/evals/{scenarios,contracts}/*_test.go.
#
# Deterministic source investigated as part of this task (was open at refinement
# time, see design.md § Decisões de Implementação task 7.1): scenario definitions
# are Go struct literals, not a CLI-loadable data format (docs/adr/0021-eval-cli-subcommand.md),
# but every eval.Scenario{} literal declares an explicit ID and Description field
# in that order — a reliable, if informal, extraction target.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
source scripts/lib-provenance.sh

OUT="docs/generated/eval-scenarios.md"

{
  provenance_header "tests/evals/{scenarios,contracts}/*_test.go (eval.Scenario{ID, Description} literals)" "scripts/generate-eval-scenarios.sh"
  echo
  echo "# Eval Scenarios"
  echo
  echo "Scenario battery run by \`strategist eval run\` (\`go test -tags=eval\`)."
  echo "Extracted from \`eval.Scenario{}\` struct literals — see"
  echo "\`docs/adr/0021-eval-cli-subcommand.md\` for why these stay Go test"
  echo "files rather than a CLI-loadable format."
  echo
  echo "| Group | File | Scenario ID | Description |"
  echo "|---|---|---|---|"
  PYTHONIOENCODING=utf-8 python3 - <<'PY'
import glob, os, re

pattern = re.compile(
    r'ID:\s*"([^"]+)"\s*,\s*\n?\s*Description:\s*"([^"]+)"',
)

files = sorted(
    p.replace(os.sep, "/")
    for p in glob.glob("tests/evals/scenarios/*_test.go")
    + glob.glob("tests/evals/contracts/*_test.go")
)

for path in files:
    with open(path, "r", encoding="utf-8") as f:
        content = f.read()
    group_match = re.search(r"^package\s+(\w+)_test", content, re.MULTILINE)
    group = group_match.group(1) if group_match else "?"
    for scenario_id, desc in pattern.findall(content):
        desc = " ".join(desc.split()).replace("|", "\\|")
        print(f"| `{group}` | `{path}` | `{scenario_id}` | {desc} |")
PY
} >"$OUT"

echo "generate-eval-scenarios: wrote $OUT"
