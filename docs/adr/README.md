# ADR Index

**Status:** Accepted
**Date:** 2026-08-04

Architecture Decision Records for this repository, listed in numeric order under
`docs/adr/`. Files stay flat (`NNNN-slug.md`, no subfolders) — see
[ADR-0015](0015-adr-index-by-theme-not-subfolders.md) for why. This table exists to make
lookup by topic fast without moving any file.

## By theme

### Build & Artifacts

| ADR | Title |
|---|---|
| [0001](0001-compiled-artifacts-gzip-json.md) | Compiled artifacts in gzip+JSON with fast path |
| [0002](0002-defaults-embutidos-embed-fs.md) | Defaults embedded in the binary via embed.FS |
| [0007](0007-structural-compression-agent-contract.md) | Structural Compression Pipeline — Agent Contract vs Go Runtime |
| [0025](0025-generated-documentation-anti-drift.md) | Generated Documentation and AI-First Anti-Drift |

### Pipeline & Governance Mechanics

| ADR | Title |
|---|---|
| [0003](0003-approval-gate-obrigatorio.md) | Approval gate mandatory and never bypassable |
| [0004](0004-learning-loop-nao-bloqueante.md) | Non-blocking learning loop |
| [0005](0005-slot-write-contracts.md) | Per-slot write contracts (read_only / write_analysis / controlled) |
| [0008](0008-single-session-assumption.md) | Single-Session Workspace Assumption |
| [0009](0009-learning-pipeline-semantic-retrieval-deferred.md) | Semantic Retrieval Deferred In Learning Pipeline |
| [0010](0010-ordered-contracts-and-mission-observability.md) | Ordered contracts and mission observability |
| [0015](0015-adr-index-by-theme-not-subfolders.md) | ADR index by theme instead of physical subfolders |
| [0024](0024-pluggable-governance-and-telemetry.md) | Pluggable Governance and AI-First Telemetry |
| [0027](0027-refinement-native-role-for-light-client.md) | Refinement (Archivist) as a native role — mission-scoped precedent |
| [0028](0028-native-role-resilient-baseline.md) | Native roles as the resilient baseline |

### Knowledge & Jewels

| ADR | Title |
|---|---|
| [0011](0011-jewel-promotion-trust-ceiling-exception.md) | Jewel promotion: trust-tier ceiling replaces human pre-approval |
| [0012](0012-jewel-lifecycle-statuses.md) | Jewel lifecycle statuses supersede the active/deprecated model |

### Execution (Sniper)

| ADR | Title |
|---|---|
| [0013](0013-sniper-documentation-asset-exception.md) | Narrow documentation-asset exception to Sniper's code-file prohibition |

### Testing

| ADR | Title |
|---|---|
| [0006](0006-e2e-test-entry-point-full-install-pipeline.md) | E2E Test Entry Point via Full Install Pipeline |
| [0016](0016-test-framework-v2.md) | internal/eval Phase 1 Scope: Deterministic Domain Surface, Not FakeProvider |
| [0017](0017-eval-fake-provider.md) | internal/eval Phase 2: Fixture-Based Content Assertions, Not FakeProvider |
| [0018](0018-eval-harvest.md) | `strategist eval harvest`: Reuse Existing Scan, Copy Whole Fixtures |
| [0019](0019-lm-studio-eval.md) | LM Studio Local Quality Review: Runbook, Not Code |
| [0020](0020-promptfoo-ci-adapter.md) | Promptfoo Adapter: Formalized Content, No CI Wiring |
| [0021](0021-eval-cli-subcommand.md) | `strategist eval run`: Wrap `go test`, One Flexible Subcommand |
| [0022](0022-treasure-scan-sq-block-bug.md) | `eval harvest --all`: Tolerant Scan, No Parser Change |
| [0026](0026-deterministic-golden-testing.md) | Deterministic Golden Testing for Generated Artifacts |

### Project & Tooling

| ADR | Title |
|---|---|
| [0014](0014-monorepo-and-toolchain-policy.md) | Monorepo and toolchain policy |
| [0023](0023-codeql-js-astro-coverage.md) | CodeQL Coverage: `javascript-typescript` Matrix Leg for `web/landing/` |

## Maintaining this index

When a new ADR is added (via the Strategist Opportunity Attack → ADR side quest, or
manually), add one row to the matching theme table above, creating a new theme table if
none fits. Do not create a new theme for a single one-off ADR unless a second one is
likely soon — prefer folding it into the closest existing theme.
