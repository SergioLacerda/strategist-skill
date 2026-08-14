#!/usr/bin/env bash
# Generates docs/generated/contract-index.md — a navigable table of every
# Strategist machine contract (contrato | origem | tipo | testes | lifecycle),
# per .analysis/strategist-ai-first-analysis/05-documentacao-gerada.md §8
# ("contract provenance"). Complements, and does not replace,
# scripts/check-contract-consistency.sh, which stays untouched and keeps
# gating skill.yaml field presence — this script only renders an index.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
source scripts/lib-provenance.sh

CONTRACTS_DIR=".strategist/contracts/machine"
OUT="docs/generated/contract-index.md"

python3 -c "import yaml" 2>/dev/null || python3 -m pip install --user pyyaml >/dev/null

{
  provenance_header "${CONTRACTS_DIR}/*.yaml (module/type/description fields)" "scripts/generate-contract-index.sh"
  echo
  echo "# Contract Index"
  echo
  echo "Machine contracts governing the Strategist mission pipeline. See"
  echo "\`.strategist/contracts/index.yaml\` for load order and"
  echo "\`.strategist/contracts/narrative/\` for the human-readable narrative"
  echo "counterpart of each phase."
  echo
  echo "| Contract | Type | Origin | Description | Tests (best-effort) |"
  echo "|---|---|---|---|---|"
  PYTHONIOENCODING=utf-8 python3 - "$CONTRACTS_DIR" <<'PY'
import sys, os, glob, re, subprocess, yaml

contracts_dir = sys.argv[1]

def best_effort_tests(module):
    """Grep tests/spec/specs and tests/evals/contracts for a mention of the
    contract's module name. Best-effort only — absence does not mean
    untested, only that no naming match was found."""
    hits = []
    for base in ("tests/spec/specs", "tests/evals/contracts"):
        if not os.path.isdir(base):
            continue
        try:
            out = subprocess.run(
                ["grep", "-rl", module, base],
                capture_output=True, text=True, check=False,
            ).stdout.strip()
        except FileNotFoundError:
            continue
        if out:
            hits.extend(out.splitlines())
    return hits

for path in sorted(glob.glob(os.path.join(contracts_dir, "*.yaml").replace(os.sep, "/"))):
    path = path.replace(os.sep, "/")
    name = os.path.basename(path)
    try:
        with open(path, "r", encoding="utf-8") as f:
            doc = yaml.safe_load(f) or {}
    except Exception as exc:  # noqa: BLE001 - one bad file must not abort the whole index
        # Collapse to one line — a raw newline inside a `| cell |` breaks the
        # markdown table. A row that fails to parse is itself useful anti-drift
        # signal (a real YAML syntax bug in the contract file), not something
        # to hide — but it must render as a valid table row.
        msg = " ".join(str(exc).split())
        print(f"| `{name}` | — | `{path}` | **UNPARSEABLE**: {msg} | — |")
        continue
    module = str(doc.get("module", doc.get("id", name))).strip()
    ctype = str(doc.get("type", "—")).strip()
    desc = str(doc.get("description", "")).strip()
    desc = " ".join(desc.split())
    if len(desc) > 160:
        desc = desc[:157] + "..."
    desc = desc.replace("|", "\\|")
    tests = best_effort_tests(module)
    tests_cell = "; ".join(f"`{t}`" for t in tests[:3]) if tests else "—"
    print(f"| `{module}` | {ctype} | `{path}` | {desc} | {tests_cell} |")
PY
  echo
  echo "Lifecycle: all contracts listed above are \`active\` unless superseded"
  echo "by a newer ADR — see \`docs/adr/README.md\` for the ADR index."
} >"$OUT"

echo "generate-contract-index: wrote $OUT"
