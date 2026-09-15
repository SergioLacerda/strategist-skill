# Governance Hierarchy — Where Each Layer Actually Lives

**Status:** Accepted
**Date:** 2026-09-15
**Mission:** `20260915-adr-hierarchy-docs-reorg`

This page is a locator, not a rewrite. It maps the conceptual
philosophy → principles → mandates/policies → ADR → implementation ladder onto the
concrete places each layer actually exists in this repository today. No content is
duplicated out of `.strategist/contracts/` or any ADR — every row below points at the
canonical source instead of copying it.

| Layer | Purpose | Where it lives today |
| --- | --- | --- |
| **Philosophy** | Identity, values, why the project exists | Not currently formalized as a standalone document. Informally expressed in [`docs/README.md`](../README.md)'s framing and `.strategist/SKILL.md`'s "What Strategist Does" section. This is a real gap, named here rather than silently filled — closing it is a separate, future decision, not assumed by this page. |
| **Principles** | Engineering criteria derived from the philosophy | Same status as Philosophy: not yet a standalone document. Partially visible in [`strategist-concepts.md`](strategist-concepts.md) and scattered ADR "Context" sections. |
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

## Relationship to ADR-0034's Role/Skill/Weapon taxonomy

The Papel (Role) / Skill / Arma (Weapon) vocabulary — a separate, already-accepted
taxonomy for *who executes* a mission phase and *what capability* they use — is defined
in [ADR-0034](../adr/0034-role-and-skill-taxonomy.md) and is orthogonal to the governance
ladder above: it describes execution structure, not normative-rule layering.
