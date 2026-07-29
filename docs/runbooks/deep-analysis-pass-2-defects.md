# Runbook — Deep Analysis Pass 2: Defects, Ambiguities, Consistency Gaps

Adversarial cross-check of every normative surface against every other. The question is
never "is this doc good?" but "**do these two sources agree, and what breaks when they
don't?**"

## Trigger

Pass 0 inventory complete (routed flags are this pass's starting checklist). Pass 1 is
not required but its routed flags feed this pass when available.

## Severity scale

- **S1 — contradiction:** two normative sources disagree; following either violates the other.
- **S2 — ambiguity / broken reference:** one source is unresolvable or two readings exist.
- **S3 — gap / orphan:** defined but never consumed, or consumed but never defined.

Every finding needs a **concrete failure scenario** (inputs/state → wrong outcome). No
scenario, no finding.

## Cross-check matrix (run each cell)

| Check | Technique |
|---|---|
| Status/state vocabulary | Grep every status token corpus-wide (`grep -rhoE 'mission_status: [a-z_]+|status \`[a-z_]+\`'`); build writer/reader table per token; any token with writers in one contract set and readers expecting a different set is S1 |
| Narrative ↔ FSM | For each documented outcome (approve/decline/revision/timeout/retry), find the FSM transition; documented loops with no transition are S1 |
| Narrative ↔ machine contracts | Does every narrative MUST have an enforcing machine/code counterpart? Aspirational rules are S3 |
| Docs ↔ deployed runtime | Do files the docs claim are "loaded at X" actually exist in the running workspace? Check degraded modes fire loudly; silent degradation is S1-operational |
| Reference resolution | Every path/anchor mentioned in entry docs must resolve in the runtime tree; check dotless-path violations of the skill's own path model; check **generated** files too (template may regenerate the defect) |
| Quick-reference ↔ authoritative list | Diff every "quick ref" against its declared source of truth (e.g. drift-pattern IDs) |
| Registry completeness | From Pass 0 §5: unreferenced artifacts, dangling refs, test files whose `subject:` doesn't resolve |
| Language policy | `grep -rln 'é |çã|ções'` (or equivalent) over mandatory-English surfaces |
| Orphan vocabulary | Every enum value / documented field: find at least one writer and one reader; else S3 |
| Test coverage of error paths | From Pass 0 §3: zero-test error families guarding critical boundaries are S3 with S1 consequences |

## Gate-semantics edge cases (walk explicitly)

- revision vs decline vs timeout — distinct outcomes? representable end-to-end?
- dual-gate ordering (policy gate vs user gate) — can one be mistaken for the other?
- delegated invocation with missing provider but implementation intent present;
- resume/re-entry: does the pre-creation checklist cover every status a prior session
  could have left behind?

## Decision Point

Done when `<base_path>/pending/<slug>/02-defects.md` exists: findings `D<n>` grouped by
severity, each with evidence table + failure scenario; severity roll-up; fix-order
recommendation; errata for earlier passes if any claim was invalidated.

## Stop Conditions

- No concrete failure scenario → not a finding; drop or downgrade it.
- Verify before asserting: a "broken" reference may be a documented degraded mode —
  downgrade severity accordingly, but flag silent degradation itself.
- Do not fix anything in this pass, even one-line typos — the fix plan is Pass 5's job.

## Reference

- `deep-analysis-workflow.md` (master); `deep-analysis-pass-0-inventory.md` (input).
- Worked example: `.analysis/pending/strategist-deep-analysis/02-defects.md` (2026-07-26).
