#!/usr/bin/env bash
set -euo pipefail

echo "Checking governance redirectors..."

for f in CLAUDE.md AGENTS.md GEMINI.md; do
  grep -q "Governance fingerprint:" "$f" || { echo "DRIFT: $f missing governance fingerprint header"; exit 1; }
  grep -q "agent-instructions.md" "$f" || { echo "DRIFT: $f missing .sdd/agent-instructions.md reference"; exit 1; }
done

echo "Governance redirectors: OK"
