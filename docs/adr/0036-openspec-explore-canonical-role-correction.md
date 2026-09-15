# ADR-0036 — openspec-explore's Canonical Role: Ranger, Not Archivist

**Status:** Accepted
**Date:** 2026-09-14
**Context:** `20260914-openspec-explore-canonical-role-correction`

## Context

ADR-0029 (2026-08-20) assigned `openspec-explore` `canonical_role:
archivist`. That ADR's own context section is explicit about the basis for
every assignment it made: the two embedded mirrors that existed at the time
were "Strategist-authored capability mirrors... [that] do not include the
upstream `SKILL.md` content" — the role assignments were made from
descriptions, not from the real upstream packages.

`external-skills-source/openspec-explore/SKILL.md` reflected that gap
directly: it was a 15-line, Strategist-authored "metadata mirror" stub,
inconsistent with `brainstorming/SKILL.md` and `openspec-propose/SKILL.md`
(both full verbatim copies of their real upstream packages). Its
description read *"Refinement provider for Strategist. Consumes discovery
output, structures the implementation approach, and writes analysis
artifacts."*

Mission `20260914-openspec-explore-canonical-role-correction` obtained the
real upstream package (`Fission-AI/OpenSpec`'s `openspec-explore/SKILL.md`,
342 lines, previously staged and archived at
`.analysis/archived/20260914-skills-native-to-import-superseded/`). Its
actual description: *"Enter explore mode - a thinking partner for exploring
ideas, investigating problems, and clarifying requirements. Use when the
user wants to think through something before or during a change."* Its body
reinforces this explicitly: *"Explore mode is for thinking, not
implementing... you must NEVER write code or implement features."*

This is a discovery/exploration activity — investigating a problem space,
clarifying requirements, thinking before a change exists — not a
refinement/proposal-authoring one. `openspec-propose` (a separate package,
already correctly assigned `canonical_role: archivist`) is the one that
authors the structured proposal/design/tasks package; `openspec-explore`
was never that.

## Decision

Correct `openspec-explore`'s `canonical_role` from `archivist` to `ranger`,
and its `category` from `refinement` to `discovery`, at the source
(`external-skills-source/openspec-explore/strategist.yaml`), and:

1. Replace the stub `SKILL.md` with a verbatim copy of the real upstream
   package, so `external-skills-source/openspec-explore/` matches the same
   portable-package-plus-adapter pattern every other embedded weapon
   already uses (ADR-0033's package layer must be an accurate, portable
   copy — a project-authored substitute description violates that).
2. Regenerate `internal/embed/defaults/plugins/catalog.yaml` and
   `internal/embed/defaults/skills/openspec-explore/skill.yaml` via `make
   embed-skills` from the corrected source — never hand-edit generated
   files.
3. Update Strategist's own capability manifest
   (`internal/embed/defaults/skill.yaml`) to list `openspec-explore` as a
   discovery weapon.

`openspec-explore` becomes a secondary, opt-in **discovery** weapon —
mirroring its prior status as a secondary, opt-in refinement weapon before
`docs/adr/0035-embedded-weapon-fallback-policy.md` DEC-004. It does not
join DEC-001's permanent embedded-weapon roster: `brainstorming` remains
the sole permanent discovery default. This correction changes no runtime
behavior — discovery already unconditionally resolves to native Ranger
regardless of the configured/tagged weapon
(`00-routing.md` § Discovery Weapon Resolution by Subtype), so the fix is
metadata/taxonomy-only.

This ADR supersedes ADR-0029's specific `canonical_role: archivist` claim
for `openspec-explore` only. ADR-0029's broader decision (the adapter-first
provider lifecycle, the canonical provider catalog, explicit readiness
tiers, and the rest of its content) stands unchanged. ADR-0027, ADR-0028,
and `docs/design/soft-profile-orka-mapping.md` remain accurate historical
records of decisions and incidents made while the old assignment was in
effect and are not rewritten.

## Consequences

### Positive

- `external-skills-source/openspec-explore/` now matches the same
  verbatim-package convention every other embedded weapon uses, closing a
  drift the earlier stub introduced.
- The taxonomy now reflects what the package actually does, which matters
  for any future tooling or documentation that reasons about "which weapon
  fits which role" from `canonical_role`.
- No runtime behavior changes, so this correction carries none of the
  installer-default risk ADR-0027/ADR-0028 documented for a *configured*
  refinement default that turns out uninvocable — `openspec-explore` was
  never the configured refinement default at the time of this correction
  (DEC-004 had already demoted it to secondary).

### Negative

- A fourth ADR (this one) is now needed to fully understand
  `openspec-explore`'s history, alongside ADR-0027/0028/0029/0035 — the
  cost of never rewriting historical ADRs.

## Rejected Alternatives

- **Edit ADR-0029 directly.** Rejected — this project's established
  convention (see ADR-0035's own precedent) is that historical ADRs are
  never rewritten; a correction gets its own ADR.
- **Keep `canonical_role: archivist` and only add a new `ranger` tag
  alongside it.** Rejected by the user: the assignment was factually wrong
  (based on a stub description that misrepresented the real package), not
  merely incomplete — an additive tag would have left the wrong claim
  standing.

## Scope Boundary

This ADR records the correction only. Implementation (the source/generated
file edits, the capability manifest update, and the documentation fix) is
mission `20260914-openspec-explore-canonical-role-correction`'s own refined
package, materialized in the same mission this ADR was written in.
