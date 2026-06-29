# Documentation — Strategist Skill

**Status:** Accepted
**Last Updated:** 2026-06-06

This is the entry point for all skill documentation. Use the table below to navigate by intent.

---

## Navigate by Intent

| I want to... | Start with |
|--------------|-----------|
| Understand the overall architecture | [`docs/architecture.md`](architecture.md) |
| Review design decisions | [`docs/adr/`](adr/) |
| Configure the skill | [`docs/configuration.md`](configuration.md) |
| Understand the internals | [`docs/skill-internals.md`](skill-internals.md) |
| Use the CLI | [`docs/cli-reference.md`](cli-reference.md) |
| See C4 diagrams | [`docs/c4-diagrams.md`](c4-diagrams.md) |
| See performance baseline | [`docs/performance-baseline.md`](performance-baseline.md) |
| Understand the learning pipeline | [`docs/learning-pipeline.md`](learning-pipeline.md) |
| Understand the skill mental model | [`docs/mental-model.md`](mental-model.md) |
| Consume telemetry and observability | [`docs/observability-contract.md`](observability-contract.md) |
| Reference fundamental concepts | [`docs/strategist-concepts.md`](strategist-concepts.md) |

---

## Language Policy

Documentation follows the skill's language configuration (`language.docs`). All files under `docs/` are written in that language. Interaction artifacts follow `language.chat`. Code internals follow `language.code`.

---

## Maintenance Rule

Every pull request that adds a new file under `docs/` must update the navigation table above.
