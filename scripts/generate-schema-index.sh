#!/usr/bin/env bash
# Generates docs/generated/schema-index.md from .strategist/schemas/*.yaml
# description fields. Uses python3+pyyaml (already a dependency of
# `make validate-fixtures`) for robust YAML parsing — schema descriptions use
# folded block scalars that plain grep/awk cannot parse reliably.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
source scripts/lib-provenance.sh

SCHEMAS_DIR=".strategist/schemas"
OUT="docs/generated/schema-index.md"

python3 -c "import yaml" 2>/dev/null || python3 -m pip install --user pyyaml >/dev/null

{
  provenance_header "${SCHEMAS_DIR}/*.yaml (description field)" "scripts/generate-schema-index.sh"
  echo
  echo "# Schema Index"
  echo
  echo "| Schema | Description |"
  echo "|---|---|"
  PYTHONIOENCODING=utf-8 python3 - "$SCHEMAS_DIR" <<'PY'
import sys, os, glob, yaml

schemas_dir = sys.argv[1]
for path in sorted(glob.glob(os.path.join(schemas_dir, "*.yaml"))):
    name = os.path.basename(path)
    try:
        with open(path, "r", encoding="utf-8") as f:
            doc = yaml.safe_load(f) or {}
        desc = str(doc.get("description", "")).strip()
        desc = " ".join(desc.split())  # collapse folded newlines/whitespace
    except Exception as exc:  # noqa: BLE001 - best-effort row, never abort the table
        desc = f"(unparseable: {exc})"
    # Escape pipes so a stray "|" in a description cannot break the table.
    desc = desc.replace("|", "\\|")
    print(f"| `{name}` | {desc} |")
PY
} >"$OUT"

echo "generate-schema-index: wrote $OUT"
