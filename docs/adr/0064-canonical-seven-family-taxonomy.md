# ADR-0064 — Canonical Seven-Family Taxonomy

**Status:** Accepted
**Date:** 2026-10-02
**Mission:** `20261002-documentation-taxonomy-canonical-review-r2`
**Supersedes for canonical documentation:** the six-family classification in [ADR-0053](0053-taxonomy-classification-criterion-and-sniper-extensibility.md)
**Related:** [ADR-0034](0034-role-and-skill-taxonomy.md), [ADR-0049](0049-atlas-treasure-chest-boundary.md), [ADR-0056](0056-jewelcrafter-public-role-and-bau-tesouro-binding.md), [ADR-0058](0058-cartographer-atlas-structure-boundary.md), [ADR-0060](0060-weapon-acquisition-flow-and-registry-derived-roster.md), and [ADR-0061](0061-weapon-sidecar-generation-and-versioned-catalog.md)

## Context

Strategist documentation accumulated overlapping classification models. ADR-0053 established six public families—Roles, Weapons, Abilities, Mechanisms, Pipeline, and Artifacts—and classified both INITIATIVE and LEVELING as Mechanisms. Later architecture work made several distinctions important enough to require separate canonical families:

- judgment-based behavior differs from deterministic enforcement;
- an operation that evaluates state and produces a result differs from the rule that constrains it;
- an operational flow differs from an individual pipeline phase;
- an accepted concept differs from an implemented or invocable capability.

Without these distinctions, documentation tends to collapse agents, external packages, judgment, application services, invariant rules, operational flows, and materialized records into the same vocabulary.

## Decision

Strategist adopts seven canonical documentation families:

| Family | Canonical meaning | Representative examples |
| --- | --- | --- |
| **Role** | Agent persona that owns judgment, responsibility, and handoffs | Scout, Ranger, Archivist, Sniper, Cartographer, Jewelcrafter, Sharpshooter |
| **Weapon** | Equipable external skill package used by a Role | Brainstorming, OpenSpec Propose, Precise Shot |
| **Feat** | Contextual, judgment-based behavior invoked by a Role | INITIATIVE, Critical Hit, Riposte, Opportunity Attack |
| **Tool** | Operation or application service that evaluates inputs and produces a result | LEVELING, Weapon Discovery, Compatibility Resolver, Binding Resolver |
| **Mechanism** | Deterministic invariant, validation, policy, or enforcement rule | Approval rules, handoff validation, binding rules, integrity checks, budget enforcement |
| **Stage** | Governed operational flow or context that may contain several phases | FULL, SHORT, ROSTER |
| **Artifact** | Materialized state, evidence, decision, or knowledge with identity and lifecycle | Analysis, Refined Package, Handoff, ADR, Weapon Roster, Weapon Binding, Confidence Report |

Classification follows responsibility rather than implementation form:

1. Who owns the judgment? Classify it as a Role.
2. What package is equipped by that Role? Classify it as a Weapon.
3. Is the behavior contextual and judgment-based? Classify it as a Feat.
4. Does it perform an operation and return a result? Classify it as a Tool.
5. Is its outcome fixed by explicit inputs and rules? Classify it as a Mechanism.
6. Does it describe a governed operational flow containing phases? Classify it as a Stage.
7. Is it materialized state or knowledge? Classify it as an Artifact.

An item may have multiple facets, but each facet must be named explicitly. A serialized contract is an Artifact; the deterministic rules it declares are Mechanisms.

### Canonical corrections

- **INITIATIVE is a Feat.** It is contextual judgment that selects or requests proportional handling; deterministic eligibility and budget rules remain Mechanisms.
- **LEVELING is a Tool.** It consumes structured mission and confidence inputs and produces an effort resolution. The policies and enforcement that constrain it are Mechanisms.
- **FULL, SHORT, and ROSTER are Stages.** Discovery, refinement, approval, and delivery are internal phases within those operational flows.
- **ROSTER and Weapon Roster are distinct.** ROSTER is the Stage that prepares selection and binding; Weapon Roster is the Artifact produced by discovery.
- **Routes are selection outcomes, not a taxonomy family.** Current route identifiers may remain compatibility vocabulary until runtime migration is separately approved.

## Evidence-state model

Every architectural entity must be described independently along these states:

| State | Meaning |
| --- | --- |
| **Conceptual** | Defined as an idea or target model. |
| **Accepted** | Approved by an authoritative decision record. |
| **Implemented** | Represented by current source or runtime behavior. |
| **Verified** | Supported by current checks or evidence for the stated scope. |
| **Invocable** | Available through a working, authorized runtime path. |

These states are not a maturity ladder that may be inferred automatically. In particular, acceptance does not prove implementation, static implementation does not prove verification, and verification does not prove live invocability.

## Authority and migration boundary

This ADR is the authority for authored architecture, onboarding, navigation, and taxonomy documentation. ADR-0053 remains historical evidence of the previous six-family decision and continues to explain its other accepted boundaries unless explicitly superseded.

This decision does **not** authorize source/runtime changes. Existing contracts, generated artifacts, schemas, tests, provider metadata, and runtime vocabulary may continue to expose the earlier model until a separate implementation wave migrates them with compatibility and validation evidence.

## Consequences

- Architecture documentation gains one stable vocabulary for concepts that previously overlapped.
- Documentation must distinguish canonical direction from current runtime evidence.
- Tools and Mechanisms must not be conflated: operations produce results; rules constrain outcomes.
- Stages and their internal phases must be named separately.
- Accepted Roles and capabilities must not be described as implemented or invocable without corresponding evidence.
- Later source/runtime refactoring must reconcile existing six-family and Pipeline vocabulary deliberately rather than treating documentation alignment as implementation completion.

## Rejected alternative

Retaining the six-family model was rejected because it keeps Feats inside Abilities, Tools inside Mechanisms, and Stages inside Pipeline. Those collapses obscure responsibility, enforcement, lifecycle, and evidence boundaries that the current architecture needs to state explicitly.
