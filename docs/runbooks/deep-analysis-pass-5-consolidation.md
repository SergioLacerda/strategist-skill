# Runbook — Deep Analysis Pass 5: Consolidation & Plan

Merge all findings into one ranked, deduplicated plan and present it at an approval
gate.

## Trigger

Passes 0–4 complete (all five findings artifacts exist).

## Steps

### 1. Merge & cluster

Group findings into **workstreams** (`W<n>`), not one task per finding — a workstream
bundles findings sharing a root cause (e.g. "canonical status vocabulary" absorbs the
schema, checklist, filter, and orphan-token findings). Each workstream lists exactly
which finding IDs it resolves.

### 2. Dedupe against the backlog

Read every entry in `<base_path>/todo/` (including dated files and long-lived analysis
files). For each overlap: **link, don't restate**, and record the link in the
consolidated analysis. A proposal that feeds a pending architectural decision becomes a
`documentation_target` design-input note, not an implementation workstream.

### 3. Classify every workstream

Per the handoff taxonomy:
- `documentation_target` — workspace-doc updates only;
- `implementation_handoff` — anything touching source trees (contracts, schemas, code,
  tests); **analysis mode never executes these**;
- `analysis_artifact` — the findings files themselves.

Flag operational actions (state-changing CLI commands like `compile-domain`) as
ready-to-run suggestions for the human — never execute them from the analysis flow.

### 4. Rank & sequence

Impact × effort, then a sequencing block: cheap-and-safe first ("now"), root-cause fixes
next, structural work after (noting which earlier fixes it folds in), new capabilities
last. Mark dependencies between workstreams explicitly (e.g. gate wiring depends on the
FSM revision event).

### 5. Package & gate

- Write `<base_path>/refined/<date>-<slug>/analysis.md` (thesis, findings index,
  critical findings, dedupe links) + `tasks.md` (workstreams, classification, sequence).
- Frontmatter must be honest about origin: if this was not a full Strategist pipeline
  mission, say so — do **not** stamp pipeline `mission_status` values the pipeline did
  not produce.
- Present the approval gate: artifact paths, workstream count, explicit warning that
  acceptance does not execute `implementation_handoff` items.
- Record the gate decision back into the package frontmatter (accepted / revision /
  rejected + date).

## Decision Point

The approval gate itself. Done means: every finding ID from Passes 0–4 appears in
exactly one workstream (or an explicit "won't fix / superseded" list); zero unlinked
overlaps with the pre-existing backlog; gate presented and decision recorded.

## Stop Conditions

- Gate acceptance never authorizes executing `implementation_handoff` items —
  implementation proceeds as separate authorized tasks per workstream, in the accepted
  sequence.
- Never stamp Strategist pipeline `mission_status` values on a package the pipeline did
  not produce.
- Operational state-changing CLI commands are suggested to the human, never executed
  from the analysis flow.

## Reference

- `deep-analysis-workflow.md` (master); inputs: all five pass artifacts.
- Worked example: `.analysis/refined/2026-07-26-strategist-deep-analysis/` (2026-07-26).
- Closure-with-evidence doctrine: `verifying-implemented-demands.md`.
