#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

inventory="docs/generated/taxonomy-entity-inventory.yaml"
[[ -f "$inventory" ]] || { echo "FAIL: $inventory is missing — run make docs-generate" >&2; exit 1; }

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
TAXONOMY_INVENTORY_OUT="$tmp" bash scripts/generate-taxonomy-inventory.sh >/dev/null
cmp -s "$tmp" "$inventory" || {
  echo "FAIL: $inventory is out of date — run make docs-generate" >&2
  exit 1
}

python3 - "$inventory" <<'PY'
from __future__ import annotations

import re
import sys
from pathlib import Path

import yaml

path = Path(sys.argv[1])
doc = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
expected = {"role", "weapon", "feat", "tool", "mechanism", "stage", "artifact"}
if doc.get("schema_version") != "strategist-taxonomy-inventory/v1":
    raise SystemExit("FAIL: unsupported taxonomy inventory schema")
if set(doc.get("families", [])) != expected:
    raise SystemExit("FAIL: taxonomy inventory does not declare exactly seven families")

seen = set()
for item in doc.get("entities", []):
    key = (item.get("family"), item.get("id"), item.get("version", ""))
    if key in seen:
        raise SystemExit(f"FAIL: duplicate taxonomy entity {key}")
    seen.add(key)
    for source in item.get("source_paths", []):
        if not Path(source).exists():
            raise SystemExit(f"FAIL: inventory source path is missing: {source}")

mechanisms = yaml.safe_load(Path("internal/embed/defaults/contracts/machine/mechanisms.yaml").read_text(encoding="utf-8"))
indexed = {(item.get("family"), item.get("id")) for item in doc.get("entities", [])}
for row in mechanisms.get("mechanisms", []):
    key = (row.get("family"), row.get("id"))
    if key not in indexed:
        raise SystemExit(f"FAIL: registry row is absent from taxonomy inventory: {key}")

patterns = [
    re.compile(r"family:\s*ability"),
    re.compile(r"strategist\.ability"),
    re.compile(r"\bAbilityName\b"),
    re.compile(r"\bFamilyAbility\b"),
    re.compile(r"ability:\s*initiative"),
]
scan_roots = [
    Path("internal/embed/defaults"), Path("internal/initiative"),
    Path("internal/mechanisms"), Path("internal/mission"), Path("internal/telemetry"),
]
for root in scan_roots:
    for source in root.rglob("*"):
        if not source.is_file() or source.name.endswith("_test.go"):
            continue
        text = source.read_text(encoding="utf-8", errors="replace")
        for number, line in enumerate(text.splitlines(), 1):
            if any(pattern.search(line) for pattern in patterns):
                raise SystemExit(f"FAIL: active taxonomy vocabulary in {source}:{number}: {line.strip()}")
print(f"OK: taxonomy inventory valid ({len(doc.get('entities', []))} entities)")
PY
