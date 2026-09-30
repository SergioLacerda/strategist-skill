# ADR-0060 — Weapon Acquisition Flow and Registry-Derived Roster

**Status:** Accepted
**Date:** 2026-09-30
**Mission:** `20260930-review-roster-weapons`
**Related:** [ADR-0035](0035-embedded-weapon-fallback-policy.md), [ADR-0055](0055-delegated-weapon-channel-is-the-host-skill-loader.md), [ADR-0056](0056-jewelcrafter-public-role-and-bau-tesouro-binding.md), [ADR-0059](0059-treasure-chest-core-removal-and-external-cutover.md)

## Context

The proposal `refactor_rooster_weapons.txt` asked for a fixed flow: source, input,
discovery, roster, compatibility, wizard, binding, runtime. The Ranger evaluation of this
mission found the separation of Roster, Compatibility, Binding and Runtime implemented in
`CompiledRegistry` (`internal/domain/compiled_registry.go`) and
`ResolveRoleWeaponBinding` (`internal/domain/role_weapon_binding.go`), with three residuals:

- the `check` command still hardcodes a two-pair roster
  (`internal/check/check_weapon_roster.go`, from ADR-0035 Decision 1), so the roster knows Roles;
- the registry rejects two Weapons with the same ID
  (`internal/domain/compiled_registry_validation.go`), so `id@version` identity is not possible;
- the raw input, compile and register stages exist (`external-skills-source/` and
  `strategist plugins prepare-embedded`) but are not recorded as one deliberate flow.

## Decision

1. **Acquisition is three stages, with no new stage.**
   (1) Raw input: packages are placed in `external-skills-source/<id>/`, manually or by an
   external repository. (2) Compile and register: `strategist plugins prepare-embedded`
   (`make generate-embedded`) is the only path that turns raw input into catalog entries,
   mirrors and lock. (3) Build and resolve: the `CompiledRegistry` derives compatibility and
   Ranked bindings; runtime only resolves persisted bindings.
2. **The raw input directory keeps its name, `external-skills-source/`.** It already supports
   outsider skills. `input/weapons/` is not introduced and nothing is renamed.
3. **Tags are declared, compatibility is computed.** A package declares `roles`,
   `supported_slots` and `version` in its sidecar; the digest is computed at stage 2; stage 3
   crosses Weapon tags with Role contracts. The roster lists Weapons without Roles.
4. **The sidecar `strategist.yaml` stays mandatory, and its creation becomes a formalized,
   automatic process** instead of hand authoring. The process must be deterministic and
   reviewable. Its exact contract (inputs, which fields it may derive, and how a human confirms
   the declared `roles` and `supported_slots`) is defined in the implementation handoff; this ADR
   does not authorize the generator to guess Role tags that the package does not declare.
5. **The `check` roster is derived from the compiled registry.** `embeddedWeaponRoster` is
   removed by an implementation handoff. The permanent pairings of ADR-0035 Decision 1 become
   registry data, and the execution pairing deferred by that ADR is resolved by the same handoff.
6. **Skill-for-Hire is a supplier of stage 1 only.** The contract is "materialize packages into
   `external-skills-source/`". No Strategist code path names it; remote registry, SemVer
   resolution, download, signing and cache stay in Skill-for-Hire. Ranked stays fail-closed: no
   host loader and no catalog-order fallback.
7. **Weapon identity is `id@version` plus digest.** The owner chose this policy on 2026-09-30,
   so `brainstorming@1.4.0` and `brainstorming@2.0.0` may coexist in the registry. The compiled
   registry, compatibility entries, Ranked bindings, the resolved binding and the lock are keyed
   by `id@version`, and the digest remains part of the identity. A binding that omits the version
   is rejected, never defaulted. Ranked stays fail-closed: no catalog-order fallback and no
   "latest" resolution. This replaces the current one-version-per-ID rule enforced by
   `validateCompiledWeapons`; whether that rule was deliberate is no longer relevant. The
   Wizard lists compatible versions. Custom Weapons keep their existing path; whether they need
   version keying is left to the implementation handoff.

## Consequences

### Positive

- The three stages map to existing commands, so the flow needs no new runtime surface.
- The roster stops depending on Role knowledge, and `check` cannot drift from the registry.
- A future Skill-for-Hire integration needs only stage 1 delivery.

### Negative / Costs

- Decision 4 needs a designed generator and a confirmation step before it can ship.
- The `id@version` policy changes Ranked lock and binding digest inputs, so existing locks need a migration with rollback
  (runbook `embedded-weapon-ingestion-migration-rollback`).

## Rejected Alternatives

- **Rename `external-skills-source/` to `input/weapons/`:** touches the Makefile,
  `prepare-embedded`, the lock, the drift check and docs, with no capability gain.
- **Keep one version per Weapon ID:** simpler, but blocks concurrent versions that the proposal requires.
- **Make the sidecar optional:** ingestion would have to guess Role tags.

## Validation Requirements

- `strategist check --json` derives its roster from the compiled registry and reports every
  registered Weapon/Role pairing.
- `make embed-skills-check` stays clean after any stage 1 or stage 2 change.
- A package without a sidecar is rejected or scaffolded through the formalized process, never
  ingested silently.

## Scope Boundary

This ADR records decisions only. It does not authorize source, test, configuration or generated
artifact changes; those are implementation handoffs in
`.analysis/refined/20260930-review-roster-weapons/tasks.md` (items 2.1-2.4) and need a separate approval.
