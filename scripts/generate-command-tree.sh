#!/usr/bin/env bash
# Generates docs/generated/command-tree.md from the built strategist binary's
# own --help output, walked recursively. This is the actual user-facing
# command surface, not a static parse of cobra.Command{} literals scattered
# across cmd/strategist/ and internal/treasurecli/ (registration is spread
# over ~15 files — walking the binary is the only reliably deterministic
# source; see docs/adr/0025-generated-documentation-anti-drift.md).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
source scripts/lib-provenance.sh

BIN="${1:-bin/strategist}"
OUT="docs/generated/command-tree.md"

if [[ ! -x "$BIN" ]]; then
  echo "generate-command-tree: $BIN not found or not executable — run 'make build' first" >&2
  exit 1
fi

walk() {
  local cmd_path="$1"
  local depth="$2"
  local help in_section=0 indent
  help="$("$BIN" $cmd_path --help 2>&1)" || true
  indent=$(printf '%*s' "$((depth * 2))" '')
  while IFS= read -r line; do
    if [[ "$line" == "Available Commands:"* ]]; then
      in_section=1
      continue
    fi
    if [[ $in_section -eq 1 ]]; then
      [[ -z "$line" ]] && break
      [[ "$line" == "  "* ]] || break
      local sub desc
      sub=$(awk '{print $1}' <<<"$line")
      desc=$(sed -E 's/^  [^[:space:]]+[[:space:]]+//' <<<"$line")
      printf '%s- `%s` — %s\n' "$indent" "$sub" "$desc"
      walk "$(printf '%s %s' "$cmd_path" "$sub" | sed -E 's/^ //')" "$((depth + 1))"
    fi
  done <<<"$help"
}

{
  provenance_header "strategist --help (recursive walk of the built binary)" "scripts/generate-command-tree.sh"
  echo
  echo "# Command Tree"
  echo
  echo "Full command surface of the \`strategist\` CLI, walked from \`$BIN --help\`."
  echo
  root_short=$("$BIN" --help 2>&1 | sed -n '1p')
  echo "\`strategist\` — ${root_short}"
  echo
  walk "" 0
} >"$OUT"

echo "generate-command-tree: wrote $OUT"
