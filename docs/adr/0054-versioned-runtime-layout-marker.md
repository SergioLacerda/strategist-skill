# ADR-0054 — Versioned Runtime-Layout Marker for Retiring the Generated Compat View

**Status:** Accepted
**Date:** 2026-09-25
**Mission:** `20260925-compat-view-removal-refinement`
**Refines:** ADR-0030 (Migration steps 3, 5 and 8)
**Related:** ADR-0029, ADR-0035, ADR-0039, ADR-0042, ADR-0045, ADR-0046, ADR-0053

## Context

ADR-0030 migrated plugin state in steps. The steps relevant here are: generate
`.strategist/skills/<id>/skill.yaml` as a compatibility view (step 3), dual-read with legacy
state authoritative at first (step 5), and "stop writing legacy registries only after a
documented deprecation period" (step 8). That period never received a trigger.

Before this work the view was read and written by:

- slot resolution (`check_slots.go`) and role validation, which treated an absent view as a
  native binding;
- readiness probing (`check_readiness.go`), which derived the Descriptor and Entrypoint from
  the view (`legacy_descriptor_valid`, `entrypoint_manifest_verified`);
- `check_weapon_bindings` and the embedded roster, which enumerated `skills/*/skill.yaml`;
- the grant readiness, which read `requested_permissions` from the view;
- the wizard, five embedded mirrors, the drift check, installable detection (keyed on the
  catalog's `legacy_manifest_path`), the dojo and 19 test files.

The generator output (`generateLegacyProviderManifest`) is an input to the normalized package
digest and the catalog node digest. Changing its bytes would re-pin every embedded Weapon.

The only readers are Strategist's own code. Third-party skills and the Treasure Chest do not
curate the view, so a calendar deprecation window says nothing about when removal is safe. What
makes removal safe is migrating the readers and making sure every running binary can tell which
layout it is looking at.

Three facts limit the design:

1. The only version comparator (`internal/plugins/resolver_versions.go`) treats `dev` and
   `-dirty` builds as zeros, so comparing binary semver would silently mislead.
2. `prepare-embedded` re-marshals `plugins/catalog.yaml` and drops unknown header keys.
3. `upgrade` reports an orphaned file only once, and `install` never reports it.

## Decision

All decisions below were approved at the gate on 2026-09-25.

1. **The marker is a version, not a date.** A versioned marker gates the view's retirement. It
   is internalized in Strategist and embedded in the binary. It is not a documentation,
   Treasure Chest or third-party concern. The marker never appears in any digest input.
   ADR-0030's "documented deprecation period" (step 8) is replaced by this marker.
2. **Fact authority.** `risk_score` and `scratch_root` are declared in the catalog entry for
   cataloged Weapons and in `adapter.yaml` for `provider add` packages. The view is never an
   authority. `domain.ResolveWeaponFacts` resolves in the order catalog, then `adapter.yaml` of
   a `plugins.lock` custom binding, then the view as a transitional fallback.
3. **Marker home.** The authority is the constant `domain.RuntimeLayoutGeneration`
   (`internal/domain/runtime_layout.go`), outside every digest input. Install and upgrade copy
   it into `.install-manifest.json` as `runtime_layout_generation,omitempty`, next to
   `leveling_policy_version`. A reader treats an absent field as 0, never as an error.
4. **Scheme.** The generation is a monotonic integer: 0 means legacy or absent, 1 means
   generation N-1 (layout-aware, view still shipped), 2 means generation N (view retired). The
   commit that changes the layout also bumps the constant, and a unit test pins its value. The
   human-facing "removed in vX.Y.Z" is whichever tag first ships N.
5. **Skew.** When the runtime generation is newer than the binary constant, trust the binary.
   `strategist check` emits the non-blocking advisory `runtime_layout_newer_than_binary`,
   registered under `fingerprint_integrity`. It is distinct from the existing hash-based
   `runtime_newer_than_binary`, which stays blocking and keeps its `--allow-downgrade`
   semantics.
6. **Reporting.** `strategist upgrade` classifies view files as `orphaned`, as it does today. At
   generation N or later, `strategist check` also emits the non-blocking `compat_view_residual`
   when a view exists for a catalog-listed Weapon, saying that edits to the view no longer take
   effect. Views are never deleted automatically.
7. **Two-step publication.** Generation N-1 (constant, manifest field and skew advisory) ships
   at least one release before generation N, which stops writing and shipping the view.
8. **Roster.** The roster is the set of catalog entries with `compatibility_source: embedded`
   and a `canonical_role`. A separate check asserts that `skills/<id>/SKILL.md` is present on
   disk.
9. **Slot resolution.** One resolver serves slot resolution and native detection, in this order:
   1. the catalog entry (`native_role` leads to the native branch, `embedded` to the Weapon
      branch);
   2. a `plugins.lock` custom binding, meaning a package under `providers/<id>/`;
   3. the compat view, only during the transition and only before generation N;
   4. `roles/<id>.yaml`.

   "No file" no longer means "native". Step 2 is implemented in the fact resolver but not
   reachable from `strategist check` (see Open items).
10. **Readiness.** For a cataloged Weapon, the Descriptor and Source come from the catalog
    entry. The Entrypoint is chosen by `runtime.kind`: for `embedded` or `host`, `SKILL.md`
    must be present and non-empty; for `openspec_root`, the runtime root must be present. A
    `provider add` package uses its `adapter.yaml` entrypoints. The reason codes
    `catalog_entry_valid`, `catalog_entry_present`, `entrypoint_payload_*` and `runtime_root_*`
    replace `legacy_descriptor_valid` and `entrypoint_manifest_verified` for cataloged Weapons.
11. **Normative prose.** P1 (`templates/agent-protocol.md`) and P2
    (`contracts/machine/preflight.yaml` and its test contract) are edited in wave 4, together
    with the retirement.
12. **Uncataloged hand-placed views.** At N-1, resolving through such a view emits the advisory
    `compat_view_uncataloged`. At N, the fallback is removed. The user approved this decision
    although the Archivist rated it only 45% confident, because the repository shows no
    evidence of such views.
13. **Generator.** The generator stays as the normalized-digest input. It is split into
    `normalizedDigestManifest` (digest input) and a thin view writer. Its output bytes do not
    change.
14. **Digest freeze.** No wave edits a file that is a digest input. That includes
    `roles/{ranger,archivist,sniper}.yaml`, `internal_skills/<role>/SKILL.md`, the role
    conformance tests, `external-skills-source/**`, and the connector and policy sources. At the
    end of the work, `external-skills-source.lock.yaml`, the catalog stamps, the
    `ranked-runtimes.yaml` `contract_digest` and the `plugins.lock` node digests are
    byte-identical.

### Rejected alternatives

- **Calendar deprecation window.** Time does not measure whether the readers have migrated.
- **Marker curated by documentation, the Treasure Chest or a third-party skill.** Only
  Strategist reads the view.
- **Marker inside the generated `skill.yaml`.** It would enter a digest input.
- **Catalog header as marker home.** `prepare-embedded` drops unknown keys, the catalog has four
  parsers, and custom providers are absent from it.
- **Dedicated embedded `runtime-layout.yaml`.** It duplicates the constant and adds a managed
  path.
- **Implicit marker only (absence from the embedded tree).** It gives no skew diagnostic and no
  persistent signal.
- **Release semver.** `dev` and `-dirty` builds compare as 0.
- **Trusting the runtime marker when it is newer.** Released binaries cannot change.
- **Blocking an older binary.** The view is not normative.
- **Reporting through `upgrade` only.** The signal is lost after the first apply.
- **Shipping N-1 and N in one release.** Older binaries would fail with the existing "provider
  not installed" message and never see the advisory.
- **`plugins.lock` inventory as roster.** Only 2 of the 5 embedded Weapons appear there, so it
  would report false "missing" results.
- **Keeping enumeration by existence.** That is the dependency being removed.
- **Catalog then role file, with no custom-binding step.** `provider add` slots would keep
  resolving only by accident.
- **Keeping the old reason codes with new meanings.** JSON consumers would read changed
  semantics without any signal.
- **Keeping the uncataloged fallback indefinitely, or removing it without an advisory.**
- **Deleting the generator and re-pinning every Weapon.**
- **Editing digest inputs and accepting a ranked re-pin.**

## Consequences

### Waves and status

Waves 0 to 3 are additive. Wave 4 is reverted by restoring the mirrors, and older runtimes
self-heal on `upgrade`.

| Wave | Generation | Content | Status (2026-09-25) |
|---|---|---|---|
| 0 | — | Investigate U-01 and SQ-006. Correct the ADR-0030 amendment. | Done |
| 1 | publishes 1 | Constant, manifest field, `runtime_layout_newer_than_binary`. | Done |
| 2 | 1 | Migrate the readers: one resolver, roster and enumeration, readiness, grant readiness, `ResolveWeaponFacts`. `check` and role validation pass with no view present. | Done |
| 3 | 1 | Migrate the dojo, add a shared catalog fixture helper, migrate test files, re-key installable detection on `installable`. Split the generator. Add `compat_view_uncataloged`. | Done, except fixtures that deliberately test the transitional branch and the installer tests |
| 4 | publishes 2 | Stop the wizard write, the mirrors and the drift check. Bump the constant, drop the view branch, add `compat_view_residual` and the orphan test, edit P1/P2/P3. `legacy_manifest_path` may then be removed. | Blocked until generation 1 has shipped one release |
| 5 | 2 | Assert the byte-identical locks and digests. A grep proves that no digest input mentions the marker. | Blocked on wave 4 |

The lock files were byte-identical to the pre-work snapshot after waves 0 to 3, and a clean
install with every `skills/*/skill.yaml` removed passes `strategist check`.

### Breaking changes

- The readiness reason codes change, which breaks preflight JSON consumers and pinned tests.
  Consumers and tests were updated in the same change.
- The error text that names a missing `skills/<id>/skill.yaml` changes.
- At N, a hand-placed view for an uncataloged Weapon stops resolving.
- Editing P1 and P2 triggers one `runtime_stale_auto_repairable` repair.

### Open items

- **U-01 (answered: no).** `strategist check` never resolves a package added with
  `strategist provider add`; the workspace ends in a `blocked` preflight. Resolving it
  end to end (`active.yaml` update, instance-id versus provider-id spelling, readiness from
  `providers/<id>/`, and the `mode: ranked` value inherited by a new binding) is a separate
  mission. Decision 9 step 2 stays documented until then.
- **SQ-006 (settled).** The grant readiness now reads `requested_permissions` from the resolved
  facts. Embedded Weapons declare none, so the result stays `no_permissions_requested` and hides
  no gap. Sourcing permissions from `adapter.yaml` waits for the `provider add` mission.
- **SQ-003 (optional).** `install` could also report orphans.
- **Exception to ADR-0030 Decision 5.** For cataloged Weapons, slot resolution reads the catalog
  entry before the `plugins.lock` binding. This is recorded in the ADR-0030 amendment.

### Risks and mitigations

- **RK-1 (high): semver compares zeros on dev builds.** Mitigation: the integer generation.
- **RK-2 (medium): an old binary re-saving the manifest drops the field.** Mitigation: an absent
  field means 0, and writers always stamp the constant.
- **RK-3 (medium): edits to a customized view silently stop taking effect.** Mitigation: the
  `compat_view_residual` text says so, and nothing is deleted automatically.
- **RK-4 (medium): an uncataloged view loses resolution.** Mitigation: the N-1 advisory.
  `provider add` packages are unaffected.
- **RK-5 (high): dropping the mirrors before the readers migrate breaks `check` on fresh
  installs.** Mitigation: wave 2 comes strictly before wave 4, and every wave is tested with the
  view absent.
- **RK-6 (high): editing role files, role SKILLs or conformance tests re-pins ranked
  certification.** Mitigation: those files are forbidden in every wave (Decision 14).
- **RK-7 (low): editing normative prose triggers a `runtime_stale` repair.** Mitigation: the
  edits are batched in wave 4.
- **RK-8 (medium): removing `legacy_manifest_path` first breaks installable detection.**
  Mitigation: detection is already re-keyed on `installable`, before the field is removed.

### Relationship to other ADRs

- This ADR is independent of ADR-0053. Neither depends on the other.
- It corrects nothing in ADR-0030 §1–§9. It replaces the trigger in Migration step 8, sets the
  end of the dual-read in step 5, and is summarized in the ADR-0030 amendment.
