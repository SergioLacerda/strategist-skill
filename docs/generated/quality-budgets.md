<!--
generated: true
source: scripts/quality-budgets.tsv (source of truth — unchanged by this generator)
generator: scripts/generate-quality-budgets.sh
generator_version: 1
do not edit manually — regenerate with: make docs-generate
-->

# Quality Budgets

Per-file line-count exceptions enforced by `make quality-budget-gate`
(`scripts/check-quality-budgets.sh`). This file is a generated view of
`scripts/quality-budgets.tsv` — edit the TSV, not this file.

| Path | Max Lines | Reason |
|---|---:|---|
| `treasure-chest/cli/treasure_chest_items.go` | 250 | treasure chest item command flow is pending a separate refactor (moved to internal/treasurecli by 20260806-treasure-chest-cmd-consolidation; relocated to treasure-chest/cli by 20260915-treasure-chest-relocation-verification) |
| `internal/compile/agent_awareness.go` | 280 | agent-awareness writer is intentionally centralized |
| `internal/integrity/warning.go` | 290 | config integrity warning formatter is cohesive |
| `treasure-chest/index.go` | 230 | treasure candidate indexing remains cohesive |
| `treasure-chest/scan.go` | 240 | treasure scanner orchestration remains cohesive |
| `treasure-chest/status.go` | 230 | treasure status transitions remain cohesive |
