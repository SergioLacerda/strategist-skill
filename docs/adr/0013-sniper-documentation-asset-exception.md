# ADR-0013 — Narrow documentation-asset exception to Sniper's code-file prohibition

**Status:** Accepted
**Date:** 2026-07-15
**Context:** `20260715-sniper-doc-target-scope-conflict`, precedent: ADR-0005 (slot write contracts)

---

## Context

Sniper's contract (`.strategist/roles/sniper.yaml`, `.strategist/internal_skills/sniper/SKILL.md`)
forbade writing `.go`/`.ts`/`.py`/`.js`/`.sh`/etc. files unconditionally, with no
exception for files Archivist had explicitly declared `task_type:
documentation_target`. `.strategist/contracts/narrative/06-execution.md` already
carried an inconsistency of its own: its "Pre-Materialization Scan" section only
blocked source-code-extension files "not declared as `documentation_target`
assets," implying an exception, while its "Write Scope" section a few lines
below restated the flat prohibition without that qualifier.

Mission `20260712-docs-landing-updates-treasure-scout` surfaced the conflict in
practice: its Archivist-refined, gate-accepted `tasks.md` tagged
`web/landing/src/pages/*.astro`, `web/landing/src/styles/global.css`,
`web/design/ui_kits/{landing,console}/i18n.js`,
`web/landing/src/pages/__tests__/landing-copy.test.ts`, and
`web/landing/src/components/__tests__/CopyButton.test.tsx` as `task_type:
documentation_target` (T8–T12, SQ-001/T15) — a legitimate landing/site
documentation scope. Sniper's flat rule blocked all of them, forcing a stop and
report instead of materializing legitimately-approved work.

## Decision

Add one identically-worded exception to all three locations where Sniper's
code-file prohibition is stated (`roles/sniper.yaml`, `SKILL.md`,
`06-execution.md`):

> Sniper may write `.astro` and `.css` files, and `.js`/`.ts`/`.tsx` files, when
> — and only when — the exact file path is listed in the mission's
> `approved_scope`/`documentation_targets` **and** tagged `task_type:
> documentation_target` in the gate-accepted `tasks.md`. Any other
> `.go`/`.ts`/`.py`/`.js`/`.sh`/etc. file, or any file of these extensions not
> so declared and accepted, remains forbidden.

This is narrower than a blanket code-write grant: it requires both an explicit
Archivist declaration and Approval Gate acceptance of that specific path before
Sniper may touch it, and it names only the five extensions actually needed by
the landing/site mission that surfaced the gap. It follows the precedent set by
ADR-0005 (slot write contracts) of stating Sniper's write boundary precisely
rather than leaving it to per-mission improvisation.

## Non-Goals

- No blanket permission for Sniper to write arbitrary code files.
- No change to Sniper's Git-mutation prohibition, or to any other `must_not`
  clause.
- No expansion beyond `.astro`/`.css`/`.js`/`.ts`/`.tsx` — a future mission
  needing another extension requires a separate ADR/proposal.

## Consequences

- Sniper can now materialize an Archivist-declared, gate-accepted
  `.astro`/`.css`/`.js`/`.ts`/`.tsx` `documentation_target` without stopping on
  a contract conflict.
- Sniper still stops and reports for any code file that is undeclared,
  mistagged, or of a different extension.
- `roles/sniper.yaml`, `SKILL.md`, and `06-execution.md` now state the same
  rule consistently; the previous internal inconsistency in `06-execution.md`
  is resolved.
- Downstream: Sniper's earlier stop-and-report on
  `20260712-docs-landing-updates-treasure-scout`'s T8–T12/SQ-001(T15) can now
  be resumed in a separate, later Sniper claim on that mission package.
