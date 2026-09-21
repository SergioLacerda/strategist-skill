# Documentation — Strategist Skill

**Status:** Accepted
**Last Updated:** 2026-09-15

This is the entry point for all skill documentation. Start with the repository
`QUICKSTART.md` for a first mission, then use the intent index below to choose
one maintained reference.

---

## Navigate by Intent

| I want to... | Start with |
|--------------|-----------|
| Start my first mission | `QUICKSTART.md` at the repository root |
| Learn the roles, phase questions, artifacts, and first-mission flow | [`onboarding/quickstart-concepts.md`](onboarding/quickstart-concepts.md) |
| Read the maintained onboarding overview | [`onboarding/readme-en.md`](onboarding/readme-en.md) |
| Read the detailed technical onboarding | [`onboarding/readme-detailed-en.md`](onboarding/readme-detailed-en.md) |
| Understand the CLI-free mission path | [`onboarding/light-client-quickstart.md`](onboarding/light-client-quickstart.md) |
| Understand the mental model and fundamental concepts | [`architecture/mental-model.md`](architecture/mental-model.md) and [`architecture/strategist-concepts.md`](architecture/strategist-concepts.md) |
| Understand the architecture and internals | [`architecture/overview.md`](architecture/overview.md) and [`architecture/skill-internals.md`](architecture/skill-internals.md) |
| See where governance sources live | [`architecture/governance-hierarchy.md`](architecture/governance-hierarchy.md) |
| Configure or operate the skill | [`configuration.md`](configuration.md) and [`cli-reference.md`](cli-reference.md) |
| Extend the provider lifecycle | [`provider-extension.md`](provider-extension.md) |
| Review design decisions and taxonomy | [`adr/`](adr/) and [ADR-0034](adr/0034-role-and-skill-taxonomy.md) |
| Understand runbooks, scripts, and `make` targets | [`makefile-scripts.md`](makefile-scripts.md) and [`runbooks/`](runbooks/) |
| Understand testing, coverage, and performance | [`test-styles.md`](test-styles.md), [`test-coverage-gaps.md`](test-coverage-gaps.md), [`integration-coverage-gaps.md`](integration-coverage-gaps.md), and [`performance-baseline.md`](performance-baseline.md) |
| Understand drift detection and learning | [`drift-detection-matrix.md`](drift-detection-matrix.md) and [`learning-pipeline.md`](learning-pipeline.md) |
| Consume telemetry and observability | [`observability-contract.md`](observability-contract.md) |
| Browse the generated Contract Inventory | [`generated/contract-index.md`](generated/contract-index.md) |
| Browse the generated CLI Inventory | [`generated/command-tree.md`](generated/command-tree.md) |
| Browse generated references | [`generated/README.md`](generated/README.md) (see [ADR-0025](adr/0025-generated-documentation-anti-drift.md)) |

Generated references are read-only materializations with provenance headers;
`make docs-generate` is their sole generator entrypoint. Do not edit files under
`docs/generated/` by hand.

The canonical authored onboarding surfaces are listed in
[`index-ownership.tsv`](index-ownership.tsv). The deterministic ownership gate
checks only that allowlist. Generated references, ADRs, and runbooks are
explicitly excluded because they retain their own indexes.

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

Every pull request that adds a new authored file under `docs/` must assign it one
owning intent row above. Generated references, ADRs, and runbooks keep their
own indexes and are linked here as bounded collections rather than duplicated
in this table.
