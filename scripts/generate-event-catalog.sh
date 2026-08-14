#!/usr/bin/env bash
# Generates docs/generated/event-catalog.md from the Attr* constants in
# internal/telemetry/schema.go. Deliberately does not duplicate the
# EventSink/event-envelope design from docs/adr/0024-pluggable-governance-and-telemetry.md
# — this is a catalog of today's actual OTel/log attribute keys, not a spec
# for the not-yet-built envelope.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
source scripts/lib-provenance.sh

SRC="internal/telemetry/schema.go"
OUT="docs/generated/event-catalog.md"

{
  provenance_header "${SRC} (Attr* constants)" "scripts/generate-event-catalog.sh"
  echo
  echo "# Event / Attribute Catalog"
  echo
  echo "OTel span and \`slog\` attribute keys currently emitted by Strategist,"
  echo "extracted from \`${SRC}\`. See"
  echo "\`docs/adr/0024-pluggable-governance-and-telemetry.md\` for the proposed"
  echo "(not yet built) \`EventSink\`/event-envelope design this catalog will"
  echo "feed once that lands."
  echo
  echo "| Go Constant | Attribute Key |"
  echo "|---|---|"
  grep -oE '^\s*Attr[A-Za-z0-9]+\s*=\s*"[^"]*"' "$SRC" | \
    sed -E 's/^\s*(Attr[A-Za-z0-9]+)\s*=\s*"([^"]*)"/| `\1` | `\2` |/'
} >"$OUT"

echo "generate-event-catalog: wrote $OUT"
