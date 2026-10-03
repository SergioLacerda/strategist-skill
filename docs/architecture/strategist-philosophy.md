# Strategist Philosophy and Engineering Principles

**Status:** Accepted
**Last Updated:** 2026-10-02

This document states the reasons behind the Strategist contracts. It and
[ADR-0064](../adr/0064-canonical-seven-family-taxonomy.md) are authoritative for
the canonical conceptual vocabulary. Numbered runtime contracts and the Go
implementation remain authoritative for current behavior; when they still use
older terminology, documentation must label that as implementation state rather
than silently redefining the canonical model.

## Seven families, one responsibility test

Strategist uses seven canonical families:

| Family | Responsibility |
| --- | --- |
| **Role** | Owns judgment, responsibility, and handoffs. |
| **Weapon** | Supplies an equipable external skill package to a Role. |
| **Feat** | Expresses contextual, judgment-based behavior. |
| **Tool** | Performs an operation over inputs and produces a result. |
| **Mechanism** | Enforces a deterministic invariant, policy, or validation rule. |
| **Stage** | Defines a governed operational flow that may contain several phases. |
| **Artifact** | Materializes state, evidence, decisions, or knowledge. |

The family follows responsibility, not file type or implementation shape. A
serialized contract is an Artifact; the rules it declares are Mechanisms. A
Role may invoke a Feat, use a Tool, and wield a Weapon inside one Stage without
merging those responsibilities.

## Fixed pipeline, replaceable weapons

Strategist owns one auditable mission pipeline:

```text
Scout → Ranger → Archivist → Approval Gate → Sniper → Learning
```

The roles and checkpoints are fixed. A configured skill package is a replaceable
weapon used inside a role boundary; it cannot redefine the pipeline, introduce a
second state protocol, or silently replace another role/provider. Discovery
subtypes are routed by Scout and normalized by Ranger. Static provider readiness
is never presented as proof that a provider was invoked.

## Evidence before authority

Different records answer different questions and must not be collapsed:

- `.strategist/plugins.lock` is the durable authority for Custom bindings.
- Certified embedded catalog data is the authority for Ranked bindings.
- Package provenance, digests, and compatibility records are evidence, not
  authorization.
- A successful runtime probe is required for live readiness.

Missing, stale, unsupported, failed, or contradictory evidence fails closed.
The system does not repair a binding, choose a native fallback, or infer live
behavior from a catalog entry.

Architecture statements use five independent evidence states: **conceptual**,
**accepted**, **implemented**, **verified**, and **invocable**. Acceptance does
not prove implementation; implementation does not prove verification; and
verification does not prove a live, authorized invocation path.

## One mission transition facade

`MissionEngine` is the mission-level transition facade. Callers submit the
mission event vocabulary through it; validation, replay, restore, gate ordering,
and retry boundaries remain consistent across live and restored missions. The
lower-level transition table is an implementation detail, not a second public
authority.

The operational flows are Stages: **FULL** for discovery through governed
delivery, **SHORT** for bounded materialization, and **ROSTER** for Weapon
discovery, compatibility, selection, and binding. Discovery, refinement,
approval, and delivery are phases within a Stage. Route identifiers are
compatibility-level selection outcomes, not another taxonomy family.

## Approval before materialization

Analysis and refinement are separate from execution. The Approval Gate is a
human decision point, and local execution policy does not replace it. Sniper's
current contract materializes approved documentation and handoff artifacts; it
does not edit source code, tests, hooks, locks, generated defaults, or release
state. Code implementation requires an execution path whose contract explicitly
permits that mutation.

## Bounded runtime and hermetic validation

Provider-owned runtimes use explicit roots, minimal environments, and scoped
writable directories. Tests isolate HOME, XDG state, provider scratch, and Go
caches. A fixture can prove containment and contract behavior, but it cannot
certify an external live provider. Release and CI claims must name the evidence
layer they actually verify.

## Change discipline

Before changing a contract, locate its source writer, generated/runtime mirror,
tests, and downstream consumers. Prefer a narrow behavior-preserving change,
add a regression test at the violated boundary, regenerate derived artifacts,
and run the smallest relevant hermetic gate before broader validation. Do not
close an analysis package or call a release complete from checkboxes alone.

Documentation may lead an implementation migration, but it must say so. A
taxonomy decision creates a target model; source/runtime vocabulary changes
remain separate work until implemented and verified with evidence.
