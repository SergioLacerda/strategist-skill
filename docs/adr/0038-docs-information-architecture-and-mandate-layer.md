# ADR-0038 — Docs Information Architecture and Mandate Layer

**Status:** Accepted
**Date:** 2026-09-15
**Mission:** `20260915-adr-hierarchy-docs-reorg`

## Context

`docs/` grew to 16 top-level files, `docs/adr/` (37 ADRs, flat per
[ADR-0015](0015-adr-index-by-theme-not-subfolders.md)), `docs/runbooks/` (75 files), and
other subtrees without a stated information architecture.
`docs/README.md` (last touched 2026-06-06) predates
[ADR-0025](0025-generated-documentation-anti-drift.md)'s generated/authored split and
[ADR-0034](0034-role-and-skill-taxonomy.md)'s Role/Skill/Weapon taxonomy, so it no longer
reflects the system it documents.

Separately, a hierarchy proposal was supplied for this project's documentation
(philosophy → principles → mandates → policies → ADR → implementation), drafted without
checking the current implemented architecture. Discovery for this mission found it does
not match how this repository actually works in several concrete ways:

- ADR-0015 already rejected physical `docs/adr/` subfolders (would break ≥8 hardcoded
  cross-references and the flat filename-numbering scan in
  `contracts/narrative/07-adr.md` § Canonical Destination Resolution).
- No `docs/philosophy/`, `docs/governance/mandates/`, or `docs/architecture/` directory
  existed prior to this mission. `GOVERNANCE.md` is an informal maintainer-decision
  document, not a mandate registry.
- `docs/` does not split uniformly into "runtime-fed" vs. "operational directive" as
  initially assumed — only `docs/runbooks/*.runbook.yaml` sidecars are read by the
  runtime at mission time (`internal/treasurecli/runbook_select.go`,
  `.strategist/treasure-chests.yaml`, `.strategist/knowledge.index.yaml`). Every other
  `docs/` subtree is directive-only for humans and agents.
- ADR-0034 already formalizes the Role/Skill/Weapon ("Papel"/"Arma") vocabulary that the
  supplied proposal material independently re-derives.
- `.strategist/knowledge.index.yaml` already contained a commented-out example entry
  anticipating a `docs/architecture` path, never implemented.

## Decision

### D1 — Group scattered architecture docs under `docs/architecture/`

Move the five existing architecture-purpose files into a new `docs/architecture/`
directory: `architecture.md` → `docs/architecture/overview.md` (renamed to avoid a name
collision with the new directory), and `mental-model.md`, `strategist-concepts.md`,
`skill-internals.md`, `c4-diagrams.md` (unchanged names). `docs/adr/` is not moved,
renamed, or sub-foldered — ADR-0015's decision stands unchanged.

### D2 — One explicit governance-hierarchy map page; `.strategist/contracts/` remains the sole mandate layer

Add `docs/architecture/governance-hierarchy.md`,
mapping the philosophy → principles → mandates/policies → ADR → implementation ladder to
where each layer concretely lives in this repository today. Mandates/policies map to
`.strategist/contracts/` (machine + narrative contracts, generated from this project's
own `internal/embed/defaults/contracts/`) — no new `docs/governance/` prose layer is
introduced, and no content is duplicated out of the contracts or out of existing ADRs.
`.strategist/`'s own directory structure is not changed by this decision.

## Rejected Alternatives

- **A one-line mandate-layer mention inside this ADR's own prose, with no separate map
  page.** Rejected: satisfies traceability but not a clean, explicit view of the whole
  ladder — a reader would have to find and read this ADR to recover it, instead of
  seeing it in one place.
- **A new `docs/governance/mandates/` directory**, distilling rules already embedded in
  individual ADRs (e.g. [ADR-0003](0003-approval-gate-obrigatorio.md),
  [ADR-0014](0014-monorepo-and-toolchain-policy.md)) into standalone mandate files.
  Rejected: it would duplicate content already enforced in `.strategist/contracts/`,
  creating two rule surfaces that can drift apart with no mechanism to keep them in
  sync — the same maintenance-burden reasoning ADR-0015 used to reject a per-ADR
  `theme:` frontmatter tag.
- **Moving `docs/adr/` under `docs/architecture/adr/` for symmetry with the new
  directory.** Rejected: this is exactly what ADR-0015 already rejected once, at smaller
  ADR volume; this decision does not reopen it.
- **Reorganizing `.strategist/` (the runtime) or the workspace (`base_path` in
  `active.yaml`, `.analysis/` only its suggested default) into the nested
  `runtime/weapons/governance/knowledge` or `sessions/<id>/...` shapes also proposed in
  the source material.** Deferred as a separate future mission — both would touch
  hardcoded paths across contracts, schemas, and Go code (`.strategist/` mirrors
  `internal/embed/defaults/` 1:1 via `go:embed`) for a benefit not yet justified at
  current scale.

## Consequences

### Positive

- `docs/architecture/` gives the scattered architecture docs one discoverable home.
- The governance-hierarchy map page answers "where does each layer of the philosophy→
  implementation ladder actually live" without inventing a new rule surface or
  duplicating `.strategist/contracts/`.
- `docs/README.md`'s navigation table (updated in the same mission) reflects ADR-0025
  and ADR-0034, closing a real staleness gap independent of this decision.
- Philosophy and Principles are named as a real, currently-unformalized gap rather than
  silently left undiscoverable or fabricated to fill the ladder.

### Negative

- Five files changed path; any link to their old `docs/<name>.md` locations from outside
  this repository (an external site, a GitHub issue, etc.) cannot be found by a repo-wide
  grep and may break. Internal references were audited and updated as part of this
  mission's execution.
- Three non-documentation files (`cmd/strategist/handoff_verify.go`,
  `internal/dojo/checker_pipeline.go`, `tests/spec/cicd_residual_contract_test.go`)
  already reference the current `docs/architecture/strategist-concepts.md` /
  `docs/architecture/overview.md` paths — this paragraph originally flagged them as
  a residual, code-authorized follow-up, but that follow-up was already completed
  (corrected 2026-09-15 during `.analysis/refined/20260915-legacy-behavior-audit/`).
- Philosophy/Principles remain unformalized; closing that gap is left as an explicit
  future decision, not assumed here.

## References

- `.analysis/refined/20260915-adr-hierarchy-docs-reorg/` — mission that produced this decision
- [ADR-0015](0015-adr-index-by-theme-not-subfolders.md) — precedent for recording
  documentation-structure decisions as ADRs, and for rejecting `docs/adr/` subfolders
- [ADR-0025](0025-generated-documentation-anti-drift.md) — generated/authored `docs/`
  split this decision builds on
- [ADR-0034](0034-role-and-skill-taxonomy.md) — Role/Skill/Weapon taxonomy referenced
  by the governance-hierarchy page
- `docs/architecture/governance-hierarchy.md` — the map page this decision introduces
