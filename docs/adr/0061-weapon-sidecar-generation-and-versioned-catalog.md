# ADR-0061 — Weapon Sidecar Generation Contract and Versioned Catalog Layout

**Status:** Accepted
**Date:** 2026-09-30
**Mission:** `20260930-review-roster-weapons`
**Related:** [ADR-0033](0033-orka-skill-package-and-project-adapter-boundary.md), [ADR-0060](0060-weapon-acquisition-flow-and-registry-derived-roster.md)

## Context

ADR-0060 keeps the `strategist.yaml` sidecar mandatory and asks for a formalized automatic
generation process (Decision 4), and it adopts `id@version` identity (Decision 7). The registry,
binding and lock already carry `id@version`; the catalog, payload layout and Wizard are still keyed
by provider ID. This ADR fixes the generator contract and the target layout. The working analysis
is `.analysis/pending/weapon-sidecar-contract-and-versioned-catalog-plan.md`.

## Decision

### Sidecar generation

1. **Command:** `strategist plugins scaffold-sidecar <package-dir> --role <role>... --slot <slot>
   [--runtime embedded|openspec_root] [--provenance <file>] [--default] [--force] [--check]`.
2. **Declared, never inferred:** `roles`, `supported_slots` and `runtime` come from the operator.
   `--role` and `--slot` are required and are the human confirmation. A Role that does not own a
   listed slot in the Role registry is rejected.
3. **Runtime default is `embedded`** (`execution_mode: prompt_bridge`). `openspec_root` is refused
   unless the package carries its `runtime/` bundle and `runtime.lock.yaml` and the `--provenance`
   file supplies the `runtime:` block (root, bootstrap, healthcheck, pinned versions). The generator
   never invents pinned runtime identity, and the block is validated before it is written.
4. **Derived by fixed rules:** `canonical_role` (first role), `version` (from `SKILL.md`
   `metadata.version`; when the package declares none, the explicit `--version` flag; otherwise
   `version_missing`), `risk_score` (discovery and refinement `write_analysis`,
   execution `controlled`), `category` (the slot name), and for the ranger Role the fixed
   `weapon_contract` (other Roles omit it). Defaults: `kind: atomic`, `scratch_root: none`,
   `default: false`.
5. **Provenance only from an explicit file.** `--provenance` populates `upstream_repo`,
   `upstream_skill_path`, `upstream_version`, `upstream_commit`, `upstream_content_digest` and
   `license`. Without it they are omitted, never invented. This is the delivery contract a
   Skill-for-Hire supplier can meet.
6. **Deterministic output:** fixed key order, LF endings, no timestamps, a header comment with the
   generator version and an input digest over `SKILL.md` and the declared flags. Only
   `<package-dir>/strategist.yaml` is written; without `--force` an existing file is never
   overwritten. `--check` regenerates in memory and exits non-zero on any difference.
7. **Failure codes, all fail-closed:** `skill_md_missing`, `skill_md_invalid`, `runtime_unsupported`, `version_missing`, `role_unknown`,
   `slot_unknown`, `role_slot_mismatch`, `runtime_bundle_missing`, `runtime_declaration_missing`,
   `sidecar_exists_differs`, `sidecar_drift`, `provenance_invalid`.

### Versioned catalog layout

8. **Catalog identity is `(id, version)`**; the same pair twice is a duplicate.
9. **`active.yaml` slot value is `id@version`**, written by the Wizard wherever a plain id
   would be ambiguous, that is, when the Role has several compatible versions of that id. A
   plain id stays valid, and is what the Wizard and the silent install write, while exactly one
   version of it exists; with several it is an error naming the options, never a default.
   Every reader accepts both spellings and pins the version when one is given.
10. **Ranked bindings are per certified version**, looked up by `(role, slot, id, version)`. A
    version without a certification digest is not offered as Ranked; it can only be Custom.
11. **Payload layout is `skills/<id>@<version>/`**, one flat directory, read with
    `ReadEmbeddedWeaponPayload(id, version)`.
12. **Raw input keeps one directory per package** in `external-skills-source/`; identity comes from
    the `SKILL.md` name and version, not from the directory name.

## Consequences

- The generator removes hand authoring without giving stage 2 any Role knowledge.
- The layout change touches every mirror path and the embedded FS, and the lock format changes
  again; migration and rollback go through `strategist upgrade` and `strategist install`
  (runbook `embedded-weapon-ingestion-migration-rollback`).
- Certification is needed per version to be Ranked, which bounds how many versions a build ships.

## Rejected Alternatives

- **Separate `version:` field per slot in `active.yaml`:** two places to keep consistent.
- **`skills/<id>/<version>/`:** two levels for no gain over the flat directory.
- **Uncertified versions selectable as Ranked:** would move certification into the Wizard.
- **Requiring `--runtime` always:** friction for the common prompt-bridge package.

## Scope Boundary

This ADR records decisions. The phases in the analysis document (P1-P6) implement them: the
generator, catalog identity, payload layout, Ranked lookup, Wizard and `active.yaml` references,
and the `strategist upgrade` migration with rollback (runbook
`embedded-weapon-ingestion-migration-rollback`).
