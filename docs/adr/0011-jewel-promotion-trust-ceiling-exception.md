# ADR-0011 — Jewel promotion: trust-tier ceiling replaces human pre-approval

**Status:** Accepted
**Date:** 2026-07-13
**Context:** `bau-tesouro-sq003-004-007` (Gate revision), formalized by `bau-tesouro-doc-drift-closure`

---

## Context

The `20260711-bau-tesouro-upgrade` mission's `forbidden` list includes:
- "Any generated jewel promotion without explicit review/approval"
- "Any automatic source promotion into governed chests"

The original jewels design (`bau-tesouro-sq003-004-007`) followed this: a jewel would
move through `candidate → reviewed → active`, requiring a human to review before
activation.

At that mission's Approval Gate, this was revised: requiring human review on every
jewel would make jewels impractical to generate at the volume chests produce them.
The alternative — no safeguard at all — was rejected as violating the grandparent
mission's forbidden clause outright.

## Decision

Jewels are generated and activated **immediately** by the analyzing agent
(`reviewed_by: agent`, status `active` on creation), with a **trust-tier ceiling** as
the safeguard instead of a pre-approval step: a jewel's `trust` field can never exceed
its parent chest's `trust.tier`. This is enforced at read time by `ValidateJewelTrust`
(`internal/domain/jewel_grade.go`) and at contract level by the `jewel_generation`
block in `internal/embed/defaults/contracts/machine/context-enrichment.yaml`.

This relaxes the `20260711-bau-tesouro-upgrade` forbidden clause on jewel promotion
**for jewels only, and only via this mechanism.** It does not relax:
- "Any automatic source promotion into governed chests" (chests themselves still
  require the existing governed-layer promotion path)
- Any other forbidden behavior in that or any other mission

A jewel cannot carry more trust than the chest it was extracted from already carries —
so the ceiling bounds the blast radius of skipping human review to "no worse than the
chest's own already-approved trust level."

## Consequences

**Positive:**
- Jewels can be generated at the pace chests are analyzed, without a human review
  queue becoming a bottleneck.
- The trust ceiling means a jewel can never smuggle in a higher trust claim than its
  source already has — bounding risk without requiring a human in the loop per jewel.
- `deprecated` status remains available for correction (manual, or automatic on
  chest removal via `treasure-chest remove` — itself a human-initiated action).

**Negative:**
- No human reviews an individual jewel before it becomes `active` — a bad extraction
  is live immediately, correctable only after the fact via manual `deprecated` marking.
- Future readers of the grandparent mission's "forbidden: jewel promotion without
  review" clause could mistake it as still fully in force; this ADR exists precisely
  to prevent that misreading — the exception is narrow (jewels, via the trust
  ceiling) and does not extend to chest-level promotion.

## Reference

- Originating mission: `bau-tesouro-sq003-004-007` (Gate revision note)
- Enforcement: `internal/domain/jewel_grade.go` (`ValidateJewelTrust`)
- Runtime: `bau-tesouro-sq009-jewels-runtime`
- Docs: [cli-reference.md § Jewels](../cli-reference.md#jewels)
