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
| [0031](0031-release-consumption-for-downstream-repos.md) | Release consumption for downstream repos |

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
| [0029](0029-external-skill-provider-lifecycle.md) | External skill provider lifecycle |
| [0030](0030-world-class-strategist-plugin-architecture.md) | World-class Strategist plugin architecture |
| [0032](0032-external-skill-cli-embedding-and-treasure-chest-ownership.md) | External skill CLI embedding and Treasure Chest ownership |
| [0033](0033-orka-skill-package-and-project-adapter-boundary.md) | Orka skill package and project adapter boundary |
| [0034](0034-role-and-skill-taxonomy.md) | Role and Skill Taxonomy |
| [0035](0035-embedded-weapon-fallback-policy.md) | Embedded weapon roster and fallback notification policy |
| [0036](0036-openspec-explore-canonical-role-correction.md) | openspec-explore's canonical role: ranger, not archivist |
| [0037](0037-wizard-role-binding-persistence.md) | Wizard role binding persistence (discovery + refinement) |
| [0038](0038-docs-information-architecture-and-mandate-layer.md) | Docs information architecture and mandate layer |
| [0039](0039-weapon-scratch-root-declaration.md) | Weapon scratch-root declaration |
| [0041](0041-cli-enforcement-sequencing-and-role-invocation-plan-naming.md) | CLI enforcement sequencing and role invocation plan naming |
| [0042](0042-ranked-custom-binding-persistence.md) | Ranked/custom binding persistence |
| [0043](0043-ranked-pipeline-pilot-implementation-decisions.md) | Ranked pipeline pilot implementation decisions |
| [0044](0044-preflight-startup-consumes-preflightresult.md) | Preflight startup consumes PreflightResult |
| [0047](0047-hardening-authorities-and-outcomes.md) | Hardening authorities and outcomes |
| [0048](0048-skill-package-evidence-authorities.md) | Skill package evidence authorities |
| [0051](0051-remote-provider-acquisition-and-trust-policy.md) | Remote provider acquisition and trust policy (proposed) |
| [0053](0053-taxonomy-classification-criterion-and-sniper-extensibility.md) | Taxonomy classification criterion and Sniper extensibility (amends 0034) |
| [0054](0054-versioned-runtime-layout-marker.md) | Versioned runtime-layout marker for retiring the generated compat view (refines 0030) |
| [0055](0055-delegated-weapon-channel-is-the-host-skill-loader.md) | A delegated Role reaches an embedded Weapon through the host skill loader (extends 0029; proposed) |
| [0057](0057-mission-state-concurrency-and-atomicity.md) | Mission state concurrency and atomicity: flock plus atomic rename (revisits 0008; proposed) |
| [0060](0060-weapon-acquisition-flow-and-registry-derived-roster.md) | Weapon acquisition flow and registry-derived roster (refines 0035; Weapon identity id@version) |
| [0061](0061-weapon-sidecar-generation-and-versioned-catalog.md) | Weapon sidecar generation contract and versioned catalog layout (extends 0060) |
| [0062](0062-governance-compatibility-implementation.md) | Provider-agnostic governance compatibility implementation |
| [0063](0063-governance-compatibility.md) | Provider-agnostic governance compatibility boundary |
| [0064](0064-canonical-seven-family-taxonomy.md) | Canonical seven-family taxonomy and evidence-state model |
| [0065](0065-provider-integration-mechanism.md) | Provider Integration Mechanism (JEV first, `main` fallback) |

### Knowledge & Jewels

| ADR | Title |
|---|---|
| [0011](0011-jewel-promotion-trust-ceiling-exception.md) | Jewel promotion: trust-tier ceiling replaces human pre-approval |
| [0012](0012-jewel-lifecycle-statuses.md) | Jewel lifecycle statuses supersede the active/deprecated model |
| [0040](0040-treasure-chest-in-repo-isolation-staging.md) | Treasure Chest in-repo isolation staging (Jewelcrafter role) |
| [0049](0049-atlas-treasure-chest-boundary.md) | Atlas/Treasure Chest staged capability boundary |
| [0050](0050-confidence-governance-contract.md) | Versioned confidence governance contract |
| [0056](0056-jewelcrafter-public-role-and-bau-tesouro-binding.md) | Jewelcrafter public role and BAU/Tesouro binding |
| [0058](0058-cartographer-atlas-structure-boundary.md) | Cartographer/Atlas structure boundary |
| [0059](0059-treasure-chest-core-removal-and-external-cutover.md) | Treasure Chest core removal and external cutover |

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
| [0045](0045-ranked-testsuitedigest-shared-pin.md) | Ranked TestSuiteDigest shared pin |
| [0046](0046-per-role-testsuitedigest-fulfills-dec003.md) | Per-role TestSuiteDigest fulfills DEC-003 |

### Project & Tooling

| ADR | Title |
|---|---|
| [0014](0014-monorepo-and-toolchain-policy.md) | Monorepo and toolchain policy |
| [0023](0023-codeql-js-astro-coverage.md) | CodeQL Coverage: `javascript-typescript` Matrix Leg for `web/landing/` |
| [0052](0052-cicd-enforcement-policy.md) | CI/CD enforcement policy (proposed) |

## Maintaining this index

When a new ADR is added (via the Strategist Opportunity Attack → ADR side quest, or
manually), add one row to the matching theme table above, creating a new theme table if
none fits. Do not create a new theme for a single one-off ADR unless a second one is
likely soon — prefer folding it into the closest existing theme.
