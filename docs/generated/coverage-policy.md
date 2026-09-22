<!--
generated: true
source: scripts/coverage-packages.tsv and scripts/coverage-exemptions.tsv (policy sources — unchanged by this generator)
generator: scripts/generate-coverage-policy.sh
generator_version: 1
do not edit manually — regenerate with: make docs-generate
-->

# Coverage Policy

Per-package minimum coverage thresholds enforced by `make cover-gate`
(`scripts/check-coverage-gate.sh`). Production inventory is discovered
from `cmd/...`, `internal/...`, and `treasure-chest/...`; test packages
are outside this gate. Thresholds remain in `scripts/coverage-packages.tsv`; reviewed exceptions
are recorded in `scripts/coverage-exemptions.tsv`. Edit those sources, not this file.

| Package | Minimum Coverage | Reason |
|---|---:|---|
| `internal/stale` | 90% | staleness contract gate |
| `internal/compile` | 95% | compiled artifact contract gate - raised to 95% (20260901-coverage-standard-95) - measured 96.1% |
| `internal/install` | 90% | installer and runtime materialization gate |
| `internal/embed` | 95% | embedded defaults availability gate - raised to 95% (20260901-coverage-standard-95) - measured 96.5% |
| `internal/telemetry` | 90% | governance telemetry gate |
| `cmd/strategist` | 90% | CLI contract surface gate - remeasured 94.1% after check* cluster moved to internal/check (20260816-cmd-strategist-cli-reorg) |
| `treasure-chest/cli` | 90% | treasure-chest/runbook CLI command cluster, extracted from cmd/strategist (20260806-treasure-chest-cmd-consolidation) - measured 95.5%; path updated from internal/treasurecli after in-repo isolation move (ADR 0040) |
| `internal/cliutil` | 90% | shared CLI helpers extracted from cmd/strategist during the same move - measured 100% |
| `internal/check` | 90% | check/check-stale CLI command cluster, extracted from cmd/strategist (20260816-cmd-strategist-cli-reorg) - measured 95.6% |
| `internal/dojo` | 90% | widened cover-gate scope (T3) - measured 90.6% |
| `internal/domain` | 90% | widened cover-gate scope (T3) - measured 95.1% |
| `internal/governance` | 95% | widened cover-gate scope (T3) - raised to 95% (20260901-coverage-standard-95) - measured 96.0% |
| `internal/handoff` | 95% | widened cover-gate scope (T3) - raised to 95% (20260901-coverage-standard-95) - measured 96.2% |
| `internal/i18n` | 95% | widened cover-gate scope (T3) - raised to 95% (20260901-coverage-standard-95) - measured 100.0% |
| `internal/runbook` | 90% | widened cover-gate scope (T3) - measured 95.4% |
| `internal/validate` | 95% | widened cover-gate scope (T3) - raised to 95% (20260901-coverage-standard-95) - measured 100.0% |
| `internal/eval` | 90% | raised from 56.9% via critical_hit_trigger Target + tests (20260806-critical-hit-pure-function-extraction) - remeasured 94.6% after reporter.go/harness.go file-size split |
| `internal/integrity` | 95% | already above baseline via jewel loader/atomic lock test additions (569a1ae) - raised to 95% (20260901-coverage-standard-95) - measured 96.4% |
| `internal/runtimefs` | 90% | already above baseline via jewel loader/atomic lock test additions (569a1ae) - measured 100.0% |
| `treasure-chest` | 95% | raised from 74.5% (SQ-005) to 88.2% (20260805-treasure-coverage-phase2) to 95.5% (20260806-treasure-coverage-95-plan), regressed to 94.1% by later feature work, closed again (20260901-coverage-standard-95) - measured 95.2%; path updated from internal/treasure after in-repo isolation move (ADR 0040) |
| `internal/governancebridge` | 0% | pure interface/type declarations, no executable statements ("[no statements]") - see 20260830-pending-v3-disposition E4 |
| `internal/plugins` | 95% | raised from 85.8% to 98.6% (2026-08-30, resolver edge-case tests) - measured 98.6% |
| `internal/plugins/conformance` | 95% | raised from 74.3% to 97.1% (2026-08-30, Validate/Stale/levelRank edge cases) - measured 97.1% |
| `internal/plugins/connectors` | 95% | raised from 78.9% to 100.0% (2026-08-30, NativeRuntimeConnector branch coverage) - remeasured 97.3% after local_path_connector.go file-size split (digestPackageDirectory relErr branch still unreachable) |
| `internal/plugins/lifecycle` | 95% | raised from 79.3% to 97.3% (2026-08-30, idempotency/error-path tests) - measured 97.3% |
| `internal/plugins/policy` | 95% | raised from 94.1% to 100.0% (2026-08-30, adapter-digest branches) - measured 100.0% |
| `internal/plugins/trust` | 95% | raised from 88.5% to 100.0% (2026-08-30, publisher/source/freshness/deprecation edge cases) - measured 100.0% |
| `internal/telemetry/sink` | 93% | measured 93.3% (self-coverage, non-recursive) - see 20260830-pending-v3-disposition E4 |
| `internal/telemetry/sink/external` | 100% | measured 100.0% - see 20260830-pending-v3-disposition E4 |
| `internal/telemetry/sink/jsonl` | 100% | measured 100.0% - see 20260830-pending-v3-disposition E4 |
| `internal/telemetry/sink/noop` | 100% | measured 100.0% - see 20260830-pending-v3-disposition E4 |
| `internal/telemetry/sink/otel` | 95% | raised from 68.8% to 100.0% (2026-08-30, full severity-mapping table) - measured 100.0% |
| `internal/telemetry/sink/slog` | 95% | raised from 68.8% to 100.0% (2026-08-30, full severity-mapping table) - measured 100.0% |
| `internal/testutil` | 95% | raised from 0.0% to 100.0% (2026-08-30, direct helper tests added; no longer excluded from `make test`) - measured 100.0% |
| `internal/runtimepayload` | 75% | embedded OpenSpec bundle verification/materialization gate - measured 79.6% (20260920-drift-a-windows-standalone-install) |
| `internal/leveling` | 90% | provider-neutral LEVELING policy, fallback, and digest contract |
| `internal/missionview` | 90% | read-only mission projection and deterministic renderer contract |

## Reviewed Exemptions

These production packages are present in the inventory but intentionally have no threshold row.

| Package | Owner | Reason |
|---|---|---|
| `internal/authorization` | quality-maintainers | baseline coverage policy is pending a dedicated authorization test budget |
| `internal/conformance` | quality-maintainers | baseline coverage policy is pending a dedicated conformance test budget |
| `internal/hardening` | quality-maintainers | baseline coverage policy is pending a dedicated hardening test budget |
| `internal/mission` | quality-maintainers | baseline coverage policy is pending a dedicated mission test budget |
| `internal/plugins/governance` | quality-maintainers | baseline coverage policy is pending a dedicated governance plugin test budget |
| `internal/provider` | quality-maintainers | local provider onboarding slice has focused contract/transaction tests (84.7%); dedicated error-matrix coverage budget remains outside this implementation slice |
| `internal/refinement` | quality-maintainers | baseline coverage policy is pending a dedicated refinement test budget |
| `internal/rolevalidation` | quality-maintainers | baseline coverage policy is pending a dedicated role validation test budget |
| `internal/runtimeenv` | quality-maintainers | baseline coverage policy is pending a dedicated runtime environment test budget |
| `treasure-chest/domain` | quality-maintainers | baseline coverage policy is pending a dedicated Treasure Chest domain test budget |
| `cmd/strategist/eval` | quality-maintainers | adapter extraction pending dedicated package-local coverage budget; existing CLI contract tests remain authoritative |
| `cmd/strategist/install` | quality-maintainers | adapter extraction pending migration of hermetic CLI install fixtures to package-local tests |
| `cmd/strategist/leveling` | quality-maintainers | adapter extraction pending dedicated package-local coverage budget; policy authority remains internal/leveling |
| `cmd/strategist/metrics` | quality-maintainers | adapter-local metrics suite is established; dedicated error-matrix coverage budget remains pending |
| `cmd/strategist/mission` | quality-maintainers | adapter extraction pending migration of mission lifecycle fixtures to package-local tests |
| `cmd/strategist/plugins` | quality-maintainers | adapter extraction (cmd-plugins-extraction) pending a dedicated package-local coverage budget; measured 84.9% on 2026-09-22 |
