# Governance Hierarchy — Where Each Layer Actually Lives

**Status:** Accepted
**Date:** 2026-10-02
**Mission:** `20260915-adr-hierarchy-docs-reorg`

This page is a locator, not a rewrite. It maps the conceptual
philosophy → principles → mandates/policies → ADR → implementation ladder onto the
concrete places each layer actually exists in this repository today. No content is
duplicated out of `.strategist/contracts/` or any ADR — every row below points at the
canonical source instead of copying it.

| Layer | Purpose | Where it lives today |
| --- | --- | --- |
| **Philosophy** | Identity, values, why the project exists | [`strategist-philosophy.md`](strategist-philosophy.md) is the canonical authored philosophy surface. |
| **Principles** | Engineering criteria derived from the philosophy | [`strategist-philosophy.md`](strategist-philosophy.md) defines the durable principles; [`strategist-concepts.md`](strategist-concepts.md) applies them to the maintained conceptual model. |
| **Mandates / Policies** | Normative, verifiable rules that must survive implementation changes | `.strategist/contracts/` — machine contracts (`contracts/machine/*.yaml`, e.g. `errors.yaml`, `approval-gate.yaml`) and narrative contracts (`contracts/narrative/*.md`, e.g. `00-routing.md`), generated from this project's own `internal/embed/defaults/contracts/`. This is the project's normative-and-durable rule surface — nothing about `.strategist/` is renamed or restructured by this page; it only points at what already exists there. |
| **ADR** | Contextual, historical decisions: what problem, what alternatives, what was chosen | [`docs/adr/`](../adr/) — flat by [ADR-0015](../adr/0015-adr-index-by-theme-not-subfolders.md), indexed by theme in [`docs/adr/README.md`](../adr/README.md). |
| **Implementation** | Materialization of the decision | The codebase itself (`internal/`, `cmd/`). |

## Why mandates aren't duplicated as `docs/` prose

An earlier draft of this page considered a new `docs/governance/mandates/` directory,
distilling rules already embedded in individual ADRs (e.g.
[ADR-0003](../adr/0003-approval-gate-obrigatorio.md),
[ADR-0014](../adr/0014-monorepo-and-toolchain-policy.md)) into standalone mandate files.
That was rejected: it would duplicate content already enforced in
`.strategist/contracts/`, creating two rule surfaces that can silently drift apart, with
no mechanism to keep hand-written prose in sync with the actual enforced contracts. See
[ADR-0038](../adr/0038-docs-information-architecture-and-mandate-layer.md) for the full
decision record.

## Relationship to the canonical seven-family taxonomy

[ADR-0064](../adr/0064-canonical-seven-family-taxonomy.md) defines the current
doc-facing families: Role, Weapon, Feat, Tool, Mechanism, Stage, and Artifact. This
taxonomy is orthogonal to the governance ladder above: the taxonomy identifies what
kind of system entity is being discussed, while the ladder identifies the authority
of the statement about it.

[ADR-0034](../adr/0034-role-and-skill-taxonomy.md) remains historical decision evidence
for the Role/Weapon distinction, and [ADR-0053](../adr/0053-taxonomy-classification-criterion-and-sniper-extensibility.md)
records the superseded six-family model. Current Go types, schemas, and runtime registry
filenames may retain earlier vocabulary as implementation evidence. They do not
supersede ADR-0064, and their migration requires a separately approved implementation
wave. A conceptual or accepted entity must not be described as implemented, verified,
or invocable without evidence for that stronger state.
