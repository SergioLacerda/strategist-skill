# ADR-0034 — Role and Skill Taxonomy (Papel / Skill / Arma)

**Status:** Accepted
**Date:** 2026-09-13
**Mission:** `20260913-role-skill-weapon-taxonomy`

## Context

Strategist has real distinctions between identity-bearing Roles, provider Skills,
and internal pipeline services. Earlier documentation collapsed ownership and
pluggability into "internal" versus "external", which made the discovery boundary
ambiguous. This ADR names the independent dimensions and the route artifacts that
connect them, so contracts, documentation, and code do not infer policy from
directory names.

## Decision

### 1. Papéis (Roles)

Every role declares two independent contract fields:

- `origin: native | external` identifies who owns the role contract.
- `extensibility: fixed | pluggable` identifies whether a provider may fill the
  role. The legacy `pluggable` field remains a derived compatibility input while
  workspaces migrate.

The current registry is deliberately small: Scout is `native/fixed`, Ranger and
Archivist are `native/pluggable`, and Sniper is `native/fixed` for this change.
`context-enrichment`, `dossier-builder`, `learning-curator`, `prompt-intake`, and
`response-critic` are pipeline services, not additional identity-bearing roles.
Unverified names such as Pathfinder, Cartographer, Jeweler, and Jewelcrafter are
not activated by this ADR.

### 2. Skills ("Armas")

- **2.1 Interna** — an `internal_skills/<name>/`'s own content: the SKILL.md/skill.yaml
  Strategist ships and invokes for its own Papéis (Interno or Externo alike). Never
  altered by a user.
- **2.2 Embarcada** — a Skill imported from the standard `external-skills-source/`
  path and materialized into the internal catalog (`compatibility_source: embedded`).
  Selectable via the Wizard. A selected Skill is a Weapon only when the owning role
  invokes it through its declared contract; catalog metadata alone is not invocation
  evidence.
- **2.3 Externa** — a Skill outside the offered catalog, selected by the client from
  their own standalone workspace at Wizard time, known to Strategist (if at all) only
  by risk/compatibility metadata for validation, carrying a single
  `compatibility_source: external` value (unified; previously split across the
  ambiguous `legacy_registry` and `sdd_skill` spellings — no code branched on either
  specific string, so the rename changes no runtime behavior).

### 3. Skill dependency rule

A Skill (Arma) may depend on other Skills — e.g. `brainstorming` invokes
`writing-plans` to produce its artifact, declared via `auxiliary_tools_allowed`.
Rule: a master Skill is never catalogued as Embarcada while a declared dependency
(`auxiliary_tools_allowed` or the structured `dependencies` field) is absent —
importing the master requires resolving and importing its dependencies too. A
reusable validator, `install.ValidateCatalogDependencies`, checks both mechanisms
against the rest of the catalog and reports every unresolved dependency; it does not
mutate the catalog or silently drop an offending entry — the caller (a future
ingestion pipeline) decides whether a violation blocks cataloguing. Running it
against today's catalog documents one known, pre-existing gap: `brainstorming`'s own
`writing-plans` dependency is not yet its own catalog entry — this ADR does not
retroactively fabricate one; a future ingestion mission catalogues it for real.

### 4. Role-owned Weapon and Ability model

Ranger owns the discovery route and must invoke the selected `brainstorming` Weapon.
The Weapon result is untrusted input: Ranger normalizes and validates it before the
handoff. Missing, incompatible, or failed invocation emits the stable
`role_invocation_failed` outcome and fails closed; there is no silent native fallback.

INITIATIVE is a consultative Ability. It consumes an immutable LEVELING resolution
and may return advice, but cannot mutate the model, provider, capability, effort,
gate, or implementation authorization. Prompt Intake, Context Enrichment, Dossier
Builder, Response Critic, and Learning Curator remain pipeline services rather than
Roles or Weapons.

One primary Weapon per Role remains supported, via `default: true`, together with
its dependencies:

A Papel's candidate Armas are every catalog entry whose explicit `roles` affinity
contains it. `canonical_role` remains a backwards-compatible single-role alias.
Exactly one is primary, via a `default: true` catalog field wired through to
`domain.ProviderContract.Default` (the domain field already existed from the prior
Role/Provider convergence mission but was never populated from the catalog).
`provider_class`, `known_provider_class`, and `specialization_taxonomy` are removed:
no code branched on their values for any selection or compatibility decision, and the
`dojo` manifest checker treats field names as generic, scenario-declared assertions,
not semantically meaningful checks. This is a backend/data-model simplification only
— no Wizard display change is included.

### 5. Final seven-family nomenclature (2026-09-24)

The public taxonomy is now canonical across architecture and contracts:

The seven canonical families are Roles, Weapons, Abilities, Pipeline Services,
Mechanisms, Routes, and Artifacts.

| Family | Responsibility question | Examples |
| --- | --- | --- |
| Role | Who owns and performs a responsibility? | Scout, Ranger, Archivist, Sniper |
| Weapon | Which bounded skill package does a pluggable Role employ? | `brainstorming`, `openspec-propose` |
| Ability | Which reusable mission behavior is performed? | INITIATIVE, Search, Opportunity Attack, Side Quest |
| Pipeline Service | Which fixed or conditional service supports the pipeline? | Prompt Intake, Context Enrichment, Dossier Builder, Response Critic, Learning Curator |
| Mechanism | Which rule governs identity, transfer, authorization, or integrity? | Role Contract, Weapon Binding, Handoff, Approval Gate, compatibility, fingerprint |
| Route | Which pipeline shape did Scout select? | `full_pipeline`, `implementation_short_route`, `critical_hit` |
| Artifact | Which materialized result is transported or persisted? | analysis, dossier, evidence pack, refined package, ADR |

`LEVELING` is an immutable operational resolver consumed by the INITIATIVE
Ability. It is not a Role, Weapon, provider, Route, or execution authority.
The role axes remain independent: `origin` identifies contract ownership and
`extensibility` identifies whether a compatible Weapon/provider may fill the
role. “Internal role” and “external role” are historical compatibility wording,
not active taxonomy aliases. Pathfinder, Cartographer, Jeweler, and
Jewelcrafter remain inactive proposals.

## Alternatives considered

- **Leave the distinction implicit, document by example only.** Rejected — every
  related mission this session already had to re-explain the same four tiers from
  scratch; one named vocabulary removes that repeated cost.
- **Keep `legacy_registry`/`sdd_skill` as two separate, unrenamed values and only
  document them as one tier in prose.** Rejected — the requester wanted no
  legacy/ambiguous naming retained, to avoid future confusion about whether the two
  spellings meant different things.
- **Defer dependency-rule enforcement entirely to the embedded-skill ingestion
  mission.** Rejected — the requester wanted a single, unified enforcement point
  built first, so that mission's ingestion pipeline is built against a settled
  dependency contract instead of retrofitting one later.
- **Migrate `auxiliary_tools_allowed` fully into the structured `dependencies`
  field before building enforcement.** Rejected for this mission — `dependencies`
  is already wired into the resolver's dependency graph while
  `auxiliary_tools_allowed` is not, but forcing a migration now, with zero real
  `dependencies` data in the catalog today, would add risk without proportional
  value; the validator checks both, and a future mission may still consolidate.

## Consequences

### Positive

- One vocabulary future contracts, ADRs, and code comments can cite by name instead
  of re-deriving the Papel/Skill distinction each time.
- The fixed-role normalization boundary is explicit, reducing the chance an
  external weapon bypasses Ranger's checkpoints.
- A single `compatibility_source: external` value removes a real source of naming
  ambiguity that no code ever actually distinguished between.
- The Skill dependency rule has one enforcement point (`ValidateCatalogDependencies`)
  before the embedded-skill ingestion pipeline is built against it, rather than
  after — and it already surfaces one real, pre-existing gap (`brainstorming` →
  `writing-plans`) instead of hiding it.
- `domain.ProviderContract.Default` is no longer a dormant field.

### Negative

- Adds named concepts ("Arma", Ability, service, route, and artifact) atop an already dense vocabulary
  (Role/Provider/Binding/Source from `strategist-papeis-personagens-skills-nativas`);
  future documentation must keep the two vocabularies from drifting out of sync.
- `brainstorming`'s own declared dependency is now visibly unresolved (via the new
  validator) rather than silently absent — intentional, but a future ingestion
  mission must actually catalogue `writing-plans` to close it, not just document it.

## Implementation boundary

This ADR records the taxonomy and reflects work landed in this mission:
the `compatibility_source` rename (`internal/embed/defaults/plugins/catalog.yaml`),
removal of `provider_class`/`known_provider_class`/`specialization_taxonomy`
(`internal/domain/install_types.go`, `internal/install/plugin_catalog.go`,
`internal/install/plugin_catalog_legacy.go`, the two embedded `skill.yaml`
manifests), wiring of `default`/`ProviderContract.Default`
(`internal/install/role_provider_catalog_mapping.go`), the role origin and
extensibility contract (`internal/domain/role_taxonomy.go`, role registry, and
role manifests), Ranger Weapon fail-closed contract, and the dependency validator
(`internal/install/plugin_catalog_dependencies.go`). It does not
implement the `local_path` connector or the directory-to-catalog generator —
those remain `20260913-embedded-skill-directory-catalog`'s scope, now declared to
depend on this mission's dependency-validator landing first (see that mission's
`tasks.md` Task 2, Revision 4). It does not change any Wizard display/UX code.
