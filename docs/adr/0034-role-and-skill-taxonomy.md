# ADR-0034 — Role and Skill Taxonomy (Papel / Skill / Arma)

**Status:** Accepted
**Date:** 2026-09-13
**Mission:** `20260913-role-skill-weapon-taxonomy`

## Context

Strategist has real, structural distinctions between its Roles and Skills that were
never named as one coherent vocabulary: some Roles have no configurable slot at all
(Scout and Strategist's other internal pipeline helpers), while others are pluggable
by design (Ranger, Archivist, Sniper). Skills, in turn, split into three provenance
tiers already visible in `plugins/catalog.yaml`, but without a name a user or
contributor could reach for. Two decorative fields, `provider_class` and
`specialization_taxonomy`, added complexity without being read by any selection or
compatibility logic. This ADR names both taxonomies against the structural facts
that already make them true, and removes the decorative fields, so future contracts,
documentation, and code comments have one vocabulary to cite instead of re-deriving
the distinction from first principles each time.

## Decision

### 1. Papéis (Roles) — all 9 existing today

- **1.1 Papel Interno (6)** — a Role with `internal_skills/<name>/` but no
  `roles/<name>.yaml` and no `active.yaml#slots` entry. Never customizable.
  `scout`, `context-enrichment`, `dossier-builder`, `learning-curator`,
  `prompt-intake`, `response-critic`.
- **1.2 Papel Externo (3)** — a Role with both `internal_skills/<name>/` (native
  fallback) and `roles/<name>.yaml` (role contract) and an `active.yaml#slots`
  entry. Pluggable by design; `strategist check`'s SLOTS/READINESS output validates
  invocability for exactly these before a mission starts. `ranger`, `archivist`,
  `sniper`.
  - Documented exception: Ranger is structurally a Papel Externo but always resolves
    natively today, regardless of `active.slots.discovery`'s configured value (see
    `contracts/narrative/00-routing.md` § Discovery Weapon Resolution by Subtype —
    a deliberate guardrail, not an oversight). "Papel Externo" names structural
    pluggability, not a guarantee of current external invocation.

### 2. Skills ("Armas")

- **2.1 Interna** — an `internal_skills/<name>/`'s own content: the SKILL.md/skill.yaml
  Strategist ships and invokes for its own Papéis (Interno or Externo alike). Never
  altered by a user.
- **2.2 Embarcada** — a Skill imported from the standard `external-skills-source/`
  path and materialized into the internal catalog (`compatibility_source: embedded`).
  Selectable via the Wizard. **A Papel Externo with an active Embarcada Skill bound
  to it is called an "Arma."**
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

### 4. Simplified Arma model — one primary weapon per Papel, plus its dependencies

A Papel's candidate Armas are every catalog entry whose `canonical_role` matches it.
Exactly one is primary, via a `default: true` catalog field wired through to
`domain.ProviderContract.Default` (the domain field already existed from the prior
Role/Provider convergence mission but was never populated from the catalog).
`provider_class`, `known_provider_class`, and `specialization_taxonomy` are removed:
no code branched on their values for any selection or compatibility decision, and the
`dojo` manifest checker treats field names as generic, scenario-declared assertions,
not semantically meaningful checks. This is a backend/data-model simplification only
— no Wizard display change is included.

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
- The Ranger native-invocation exception is now explicitly documented as
  intentional, reducing the chance a future contributor "fixes" it as a bug.
- A single `compatibility_source: external` value removes a real source of naming
  ambiguity that no code ever actually distinguished between.
- The Skill dependency rule has one enforcement point (`ValidateCatalogDependencies`)
  before the embedded-skill ingestion pipeline is built against it, rather than
  after — and it already surfaces one real, pre-existing gap (`brainstorming` →
  `writing-plans`) instead of hiding it.
- `domain.ProviderContract.Default` is no longer a dormant field.

### Negative

- Adds one more named concept ("Arma") atop an already dense vocabulary
  (Role/Provider/Binding/Source from `strategist-papeis-personagens-skills-nativas`);
  future documentation must keep the two vocabularies from drifting out of sync.
- `brainstorming`'s own declared dependency is now visibly unresolved (via the new
  validator) rather than silently absent — intentional, but a future ingestion
  mission must actually catalogue `writing-plans` to close it, not just document it.

## Implementation boundary

This ADR records the taxonomy and reflects work already landed in this mission:
the `compatibility_source` rename (`internal/embed/defaults/plugins/catalog.yaml`),
removal of `provider_class`/`known_provider_class`/`specialization_taxonomy`
(`internal/domain/install_types.go`, `internal/install/plugin_catalog.go`,
`internal/install/plugin_catalog_legacy.go`, the two embedded `skill.yaml`
manifests), wiring of `default`/`ProviderContract.Default`
(`internal/install/role_provider_catalog_mapping.go`), and the dependency
validator (`internal/install/plugin_catalog_dependencies.go`). It does not
implement the `local_path` connector or the directory-to-catalog generator —
those remain `20260913-embedded-skill-directory-catalog`'s scope, now declared to
depend on this mission's dependency-validator landing first (see that mission's
`tasks.md` Task 2, Revision 4). It does not change any Wizard display/UX code.
