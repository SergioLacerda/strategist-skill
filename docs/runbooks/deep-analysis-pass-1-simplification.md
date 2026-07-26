# Runbook — Deep Analysis Pass 1: Main-Flow Simplification & Reuse

Walk the canonical pipeline end to end and hunt for duplication, dead weight, and
consolidation opportunities. Constructive lens: "what could be simpler or shared" —
defects found en route are **flagged and routed to Pass 2**, not analyzed here.

## Trigger

Pass 0 inventory complete. Inputs: the inventory (duplication metrics, restated-rule
map, orphan lists) + the pipeline's narrative contracts, machine contracts, schemas,
and the runtime state machine implementation.

## Steps

### 1. Build the pipeline map

One table row per pipeline step: **step | owner | contract(s) | schema(s) | artifact |
FSM state**. Read the narrative contracts in load order; extract inputs/outputs/behavior
per phase. The map itself is a deliverable — gaps in the FSM column are findings.

### 2. Hunt duplication (from Pass 0 evidence)

For each restated rule / copy-pasted block:
- identify the natural **normative home** (one contract owns the rule);
- classify the fix: pointer-ization (cheap) vs compile-time projection (structural);
- check whether a machine-readable form could replace prose (checkable > readable-only).

### 3. Hunt structural simplifications

- **Dead states/events:** any FSM state with no inbound transition; any event used for
  multiple unrelated meanings (overloading); legacy items kept "for compatibility".
- **Parallel vocabularies:** two naming systems for the same concept (e.g. artifact
  frontmatter statuses vs FSM states) with implicit mapping → propose one canonical
  table with projections.
- **Pointer chains:** doc A → tombstone B → template C → generated D; collapse.
- **Changelog prose in normative text:** history belongs in ADRs.

### 4. Hunt cost-vs-value on short routes

Estimate the startup read cost (lines/tokens) for the cheapest route (e.g. quick
capture). If a lazy per-phase loading mechanism exists (contract index `by_phase`),
verify the entry docs actually use it. Token-economy wins here apply to every mission.

### 5. Rank

Impact × effort; note which findings a structural fix (single-sourcing) would make
obsolete by construction — do not double-count them.

## Decision Point

Done when `<base_path>/pending/<slug>/01-simplification.md` exists: pipeline map +
findings table (`S<n> | location | pattern | proposal | effort`) + ranking + flags
routed to Pass 2/4.

## Stop Conditions

- A finding without a concrete proposal and effort estimate is not done.
- Do not propose removing a guardrail to simplify — safety rules may be restated
  deliberately; the fix is projection from one source, not deletion.
- Defects found en route are routed to Pass 2, never analyzed or fixed here.

## Reference

- `deep-analysis-workflow.md` (master); `deep-analysis-pass-0-inventory.md` (input).
- Worked example: `.analysis/pending/strategist-deep-analysis/01-simplification.md` (2026-07-26).
