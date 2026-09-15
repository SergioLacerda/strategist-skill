# Documentation — Strategist Skill

**Status:** Accepted
**Last Updated:** 2026-09-15

This is the entry point for all skill documentation. Use the table below to navigate by intent.

---

## Navigate by Intent

| I want to... | Start with |
|--------------|-----------|
| Learn the roles, phase questions, artifacts, and first-mission flow | [`docs/onboarding/quickstart-concepts.md`](onboarding/quickstart-concepts.md) |
| Understand the overall architecture | [`docs/architecture/overview.md`](architecture/overview.md) |
| See where philosophy, principles, mandates, ADRs, and implementation each actually live | [`docs/architecture/governance-hierarchy.md`](architecture/governance-hierarchy.md) |
| Review design decisions | [`docs/adr/`](adr/) |
| Configure the skill | [`docs/configuration.md`](configuration.md) |
| Understand the internals | [`docs/architecture/skill-internals.md`](architecture/skill-internals.md) |
| Use the CLI | [`docs/cli-reference.md`](cli-reference.md) |
| See C4 diagrams | [`docs/architecture/c4-diagrams.md`](architecture/c4-diagrams.md) |
| See performance baseline | [`docs/performance-baseline.md`](performance-baseline.md) |
| Understand the learning pipeline | [`docs/learning-pipeline.md`](learning-pipeline.md) |
| Understand the skill mental model | [`docs/architecture/mental-model.md`](architecture/mental-model.md) |
| Consume telemetry and observability | [`docs/observability-contract.md`](observability-contract.md) |
| Reference fundamental concepts | [`docs/architecture/strategist-concepts.md`](architecture/strategist-concepts.md) |
| Look up which script backs which `make` target | [`docs/makefile-scripts.md`](makefile-scripts.md) |
| Understand the test style taxonomy and coverage gates | [`docs/test-styles.md`](test-styles.md) |
| Check unit test coverage gaps and their implementation status | [`docs/test-coverage-gaps.md`](test-coverage-gaps.md) |
| Check integration-style test coverage gaps and their implementation status | [`docs/integration-coverage-gaps.md`](integration-coverage-gaps.md) |
| See which drift class each detector actually checks | [`docs/drift-detection-matrix.md`](drift-detection-matrix.md) |
| Browse a generated, always-current inventory of contracts, schemas, CLI commands, telemetry events, and quality gates (see [ADR-0025](adr/0025-generated-documentation-anti-drift.md)) | [`docs/generated/`](generated/) |
| Understand the Role/Skill/Weapon ("Papel"/"Arma") taxonomy | [ADR-0034](adr/0034-role-and-skill-taxonomy.md) |

---

## What the runtime actually reads vs. what is directive-only

Not all of `docs/` is consumed the same way. Only `docs/runbooks/*.runbook.yaml`
sidecars are read by the runtime at mission time — scored against mission signals by
`internal/runbook.Select()` (see `internal/treasurecli/runbook_select.go`) and
registered as a knowledge source in `.strategist/treasure-chests.yaml` and
`.strategist/knowledge.index.yaml`. Every other part of `docs/` — `adr/`, `architecture/`,
`onboarding/`, `design/`, `plans/`, `generated/`, and the top-level files in this
directory — is directive-only: written for humans and for agents to read during a
mission, but never loaded back by the pipeline itself.

---

## Language Policy

Documentation follows the skill's language configuration (`language.docs`). All files under `docs/` are written in that language. Interaction artifacts follow `language.chat`. Code internals follow `language.code`.

---

## Maintenance Rule

Every pull request that adds a new file under `docs/` must update the navigation table above.
