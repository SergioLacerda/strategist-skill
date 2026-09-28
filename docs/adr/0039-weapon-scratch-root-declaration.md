# ADR-0039 — Weapon Scratch-Root Declaration

**Status:** Accepted
**Date:** 2026-09-15
**Missions:** `20260915-openspec-store-scoping`, `20260915-weapon-scratch-root-taxonomy`

## Context

Mission `20260915-openspec-store-scoping` found the `openspec-propose` weapon's
underlying `openspec` CLI writing its own working/scratch files (`openspec/changes/<id>/`)
into the repository root and `.analysis/` (the curated workspace domain) instead of the
runtime domain (`.strategist/`). That mission's fix was a one-off: hand-written prose in
`.strategist/roles/archivist.yaml`'s `custom_brief`, covering only `openspec-propose`,
living only in the installed runtime copy — not the shared `internal/embed/defaults/`
authoring source, so it does not travel to a fresh install and does not generalize to any
future weapon with the same problem.

`20260915-weapon-scratch-root-taxonomy` (analysis/design mission) confirmed via direct
code inspection (`rg 'exec\.Command|os/exec' internal/`) that no Go code invokes a
weapon's own CLI/tooling — the parent LLM agent does, driven entirely by whatever
brief/manifest content is surfaced to it. There is therefore no Go-level interception
point; the durable fix has to be a declared, generically-read manifest field plus a
generic role-brief instruction.

## Decision

### D1 — New sidecar field `scratch_root: runtime | none`

Each external skill's existing project-adapter sidecar
(`external-skills-source/<id>/strategist.yaml`, [ADR-0033](0033-orka-skill-package-and-project-adapter-boundary.md)'s
two-layer contract) may declare `scratch_root: runtime` if the weapon creates its own
working files. `workspace` is deliberately not a legal value — a weapon's own working
format is implementation detail, never a curated mission artifact. Absent field behaves
as `none`, preserving prior behavior for every skill that doesn't need one.

### D2 — Ingestion carries the field through as data

`internal/install/embedded_weapon_ingestion.go`'s `externalSkillAdapter` struct gained a
`ScratchRoot` field, validated against the two-value enum (fail closed — reject anything
else, matching this file's existing validation style). It flows into
`pluginCatalogProvider` (`internal/install/plugin_catalog.go`), the embedded catalog
(`internal/embed/defaults/plugins/catalog.yaml`), and both generated mirrors
(`internal/embed/defaults/skills/<id>/skill.yaml`, and after install,
`.strategist/skills/<id>/skill.yaml`) the same way `risk_score`/`roles` already do.

### D3 — One generic canonical behavior, added directly to both role sources

`resolve_weapon_scratch_root` was added to the `canonical:` block of
`internal/embed/defaults/roles/ranger.yaml` and `archivist.yaml`, instructing: before
invoking a weapon that declares `scratch_root: runtime`, resolve and `cd` into
`.strategist/weapon-runtime/<weapon-id>/` first. The equivalent entry was also added to
`skill.yaml`'s `slots.discovery.canonical`/`slots.refinement.canonical`, for
documentation consistency with the existing slot-level convention — but that alone does
not reach either role.

**Correction to the originating design**, found during implementation: the design assumed
`roles/ranger.yaml`/`archivist.yaml`'s `canonical:` block is generated from `skill.yaml`
at compile time. It is not — both role files are their own hand-authored static files
(confirmed: none of Ranger's existing entries like `search`, `select_runbook`,
`scope_observation` appear in `skill.yaml` at all). The canonical entry had to be added
directly to both role files or neither role would ever read it.

### D4 — Recorded as a new ADR, not an edit to ADR-0033

ADR-0033 governs the portable-SKILL.md-vs-project-adapter boundary; this is new decision
content built on top of that boundary (a specific new field and a new generic role
behavior), not a correction to it.

### D5 — `custom_brief` prose removed only after the generic mechanism was verified

The prior mission's `custom_brief` prose (installed-runtime-only, never in
`internal/embed/defaults/roles/archivist.yaml` — confirmed that file's own `custom_brief`
was `""` throughout) was removed from `.strategist/roles/archivist.yaml` only after D1–D3
were implemented, `prepare-embedded --check` reported no drift, and the new ingestion
tests passed — avoiding a window where two sources of truth (prose and generic mechanism)
could disagree.

## Rejected Alternatives

- **A boolean `creates_own_working_files: true` with an assumed-always-runtime default**,
  instead of an explicit `scratch_root` enum. Rejected: an explicit enum names the
  destination and leaves room to reject `workspace` deliberately, rather than encoding an
  implicit assumption a future reader has to rediscover.
- **A per-weapon `custom_brief` special case for every future weapon that needs this.**
  Rejected: this is exactly the pattern being replaced — it does not survive
  `strategist compile`/`install` regeneration and requires an agent to read and follow
  per-installation prose correctly every time, with no fail-closed validation.
- **A runtime Go interception point** (e.g. wrapping `exec.Command`). Rejected: no Go code
  invokes weapon CLIs today (confirmed by repo-wide search) — there is no call site to
  intercept.

## Consequences

### Positive

- Any future weapon that creates its own working files declares `scratch_root: runtime`
  once, at ingestion time, with fail-closed validation — no hand-written per-weapon prose,
  and the declaration survives `strategist compile`/`install` regeneration.
- `openspec-propose` now declares `scratch_root: runtime` on its own sidecar
  (`external-skills-source/openspec-propose/strategist.yaml`), and the generic mechanism
  supersedes the one-off prose fix from the prior mission.

### Negative

- The generic role-brief instruction (D3) has no code-level enforcement — same trust model
  every other canonical behavior in `roles/*.yaml` already relies on (e.g.
  `consult_treasure_chests`), not a new category of risk.
- `skill.yaml`'s own `slots.*.canonical` entries (added for documentation consistency)
  are not literally read by either role at invocation time — only the entries directly in
  `roles/ranger.yaml`/`archivist.yaml` are. A future reader relying on `skill.yaml` alone
  to know what a role actually does would be misled; this ADR and the role files'
  "canonical behaviors — read-only, source in skill.yaml" comments should eventually be
  reconciled, but that reconciliation is not done by this decision.

## References

- `.analysis/refined/20260915-openspec-store-scoping/` — mission that first hit the
  problem and applied the one-off fix this decision supersedes
- `.analysis/refined/20260915-weapon-scratch-root-taxonomy/` — analysis/design mission
  this decision implements
- [ADR-0033](0033-orka-skill-package-and-project-adapter-boundary.md) — the two-layer
  `SKILL.md`/`strategist.yaml` contract this decision extends
