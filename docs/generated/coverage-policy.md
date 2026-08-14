<!--
generated: true
source: scripts/coverage-packages.tsv (source of truth — unchanged by this generator)
generator: scripts/generate-coverage-policy.sh
generator_version: 1
do not edit manually — regenerate with: make docs-generate
-->

# Coverage Policy

Per-package minimum coverage thresholds enforced by `make cover-gate`
(`scripts/check-coverage-gate.sh`). This file is a generated view of
`scripts/coverage-packages.tsv` — edit the TSV, not this file.

| Package | Minimum Coverage | Reason |
|---|---:|---|
| `internal/stale` | 90% | staleness contract gate |
| `internal/compile` | 90% | compiled artifact contract gate |
| `internal/install` | 90% | installer and runtime materialization gate |
| `internal/embed` | 90% | embedded defaults availability gate |
| `internal/telemetry` | 90% | governance telemetry gate |
| `cmd/strategist` | 90% | CLI contract surface gate - remeasured 94.5% after treasure-chest cluster moved to internal/treasurecli (20260806-treasure-chest-cmd-consolidation) |
| `internal/treasurecli` | 90% | treasure-chest/runbook CLI command cluster, extracted from cmd/strategist (20260806-treasure-chest-cmd-consolidation) - measured 95.5% |
| `internal/cliutil` | 90% | shared CLI helpers extracted from cmd/strategist during the same move - measured 100% |
| `internal/dojo` | 90% | widened cover-gate scope (T3) - measured 90.6% |
| `internal/domain` | 90% | widened cover-gate scope (T3) - measured 95.1% |
| `internal/governance` | 90% | widened cover-gate scope (T3) - measured 98.9% |
| `internal/handoff` | 90% | widened cover-gate scope (T3) - measured 91.6% |
| `internal/i18n` | 90% | widened cover-gate scope (T3) - measured 94.4% |
| `internal/runbook` | 90% | widened cover-gate scope (T3) - measured 95.4% |
| `internal/validate` | 90% | widened cover-gate scope (T3) - measured 90.4% |
| `internal/eval` | 90% | raised from 56.9% via critical_hit_trigger Target + tests (20260806-critical-hit-pure-function-extraction) - measured 95.5% |
| `internal/integrity` | 90% | already above baseline via jewel loader/atomic lock test additions (569a1ae) - measured 98.2% |
| `internal/runtimefs` | 90% | already above baseline via jewel loader/atomic lock test additions (569a1ae) - measured 100.0% |
| `internal/treasure` | 95% | raised from 74.5% (SQ-005) to 88.2% (20260805-treasure-coverage-phase2) to 95.5% (20260806-treasure-coverage-95-plan) - measured 95.5% |
