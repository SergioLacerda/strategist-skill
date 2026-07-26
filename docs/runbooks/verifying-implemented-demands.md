# Runbook: Verifying a Possibly-Already-Implemented Demand

## Trigger

A request to implement a demand from `<base_path>/refined/<mission_id>/`, especially when
phrased as "if it's already finished, move it to done" — or any time you're about to start
implementation work and a refined package already exists for it.

This is also the procedure to follow whenever the mandatory bootstrap stale scan
(`.strategist/contracts/narrative/01-bootstrap.md`, `11-critical-hit.md` § Stale Card
Detection Trigger 3) flags a candidate.

## Pre-Refinement Duplicate Check

Before generating a new refined package, inventory the configured analysis lifecycle
directories when they exist: `pending/`, `refined/`, `done/`, and `archived/`.
Search for related artifacts by exact `mission_id`, slug variants, demand title, and
scope keywords.

If a matching refined package exists, inspect its four canonical files:
`analysis.md`, `proposal.md`, `design.md`, and `tasks.md`. Treat
`mission_status: archivist_done` or equivalent complete package evidence as
already refined unless the package documents a material gap.

Report and reuse the existing package instead of overwriting it. Create a residual
pending demand only when the existing package is materially incomplete or stale.
This check only decides whether refinement already exists before generation; the
steps below verify whether a refined demand has already been implemented.

## Steps

1. Read the package's `tasks.md` front-matter: `approved_scope` and `acceptance_checks`. These
   define what "done" means for this mission — not general impressions of the work.
2. Inspect the current state of every file listed in `approved_scope`. Read them; don't infer
   from file names or commit messages.
3. Re-run verification: the mission's own "Suggested Verification Command" section if `tasks.md`
   has one; otherwise this repo's standard — `go build ./...`, `go test ./...`, `go vet ./...`,
   `go test -tags spec ./tests/spec/...`. If `.strategist/contracts`, `.strategist/schemas`, or
   `.strategist/personas` are in `approved_scope`, diff them against their
   `internal/embed/defaults/` source (the single authoring tree) to confirm they match.

## Decision Point

**All acceptance_checks verifiably met** (against real command output and file content, not
assumption):
1. Write `completion-report.md` into the package: `what_was_completed`, `evidence_supplied`,
   `checks_run`, `unresolved_residuals`. Be explicit about anything not run or only partially
   verified — an honest residual is not a blocker, a silent gap is.
2. Mark per-task `status: completed` in `tasks.md`, but only for tasks the supplied evidence
   actually covers.
3. Present the Critical Hit closure gate exactly as defined in
   `.strategist/contracts/machine/critical-hit.yaml#inline_gate.closure_move` — evidence summary,
   then "Confirm? (sim / nao)". Stop and wait; this gate cannot be bypassed.
4. On `sim`: move `<base_path>/refined/<mission_id>/` to `<base_path>/done/<mission_id>/` — `git
   mv` if the analysis workspace is git-tracked in this repo, a plain move otherwise (check with
   `git ls-files <base_path>/` first).
5. On `nao` or no response: leave the package in `refined/` untouched.

**Not all acceptance_checks are met:** this is not a closure candidate. Treat it as a normal
implementation task and do the missing work — do not write a completion report for partial work
and do not present the closure gate.

## Stop Conditions

- Any acceptance check can't be verified against actual command output or current file
  content — do not proceed to closure, and never invent or assume an `evidence_summary`
  (`critical-hit.yaml`'s own invariant: evidence "MUST NOT be invented by Strategist").
- Explicit decline (`nao`) at the gate — leave the package exactly where it was.

## Reference

- `.strategist/contracts/machine/critical-hit.yaml` — the authoritative closure_move contract
  (trigger conditions, gate display, evidence fields, invariants). This runbook shows how to
  arrive at a well-formed evidence summary; it does not redefine the contract.
