# ADR-0018 — `strategist eval harvest`: Reuse Existing Scan, Copy Whole Fixtures, Correct a Prior Framing

**Status:** Accepted  
**Date:** 2026-08-04  
**Context:** `20260804-eval-harvest`

---

## Context

ADR-0017 (`20260804-eval-fake-provider`, DEC-2) named "harvest regression
fixtures from real missions" as a prerequisite for part of `internal/eval`
Phase 2's deferred scenarios (B1–B3 recorded-decision sub-cases), and
described it loosely as harvesting a "recorded `route_decision`."

This mission (`20260804-eval-harvest`, refining `SQ-002`) designed that
missing capability. Discovery found that `internal/treasure/scan.go`
already implements exactly the mission-discovery half of "harvest":
`ScanMissions(basePath)`/`ScanMissionsInDir(dir)` walk
`<base_path>/refined/` and `<base_path>/done/`, locate mission directories
containing a `tasks.md`, and return parsed `ScannedMission` records. This is
live, tested infrastructure backing `strategist treasure-chest index`'s
scan phase (`internal/treasurecli/treasure_chest_scan.go`), not a design sketch.

Discovery also found that no `route_decision` record is ever persisted to
disk anywhere in this codebase — Scout emits it as log/conversational text
only (`.strategist/contracts/machine/scout-routing.yaml#emit`). The prior
ADR's framing was therefore imprecise: there is no `route_decision`
artifact anywhere in mission history to harvest.

## Decision

**DEC-1 (granularity):** `harvest` copies whole source files verbatim, no
section extraction, defaulting to `analysis.md` with `--include
design,proposal,tasks,adr,report` for more.

**DEC-2 (no code generation):** `harvest` writes only `.md` fixture files
under `tests/evals/regression/<mission_id>/`. It never generates `.go` test
code — scenario authorship stays a separate, deliberate human/agent step.

**DEC-3 (single + batch modes):** `strategist eval harvest <mission_id>`
for one mission; `strategist eval harvest --all` reuses
`treasure.ScanMissions(basePath)` for every mission. No implicit batch mode
on a bare invocation.

**DEC-4 (idempotent overwrite):** re-running `harvest` overwrites the
existing fixture copy; no versioning scheme — git history already covers
that need.

**DEC-5 (correction):** ADR-0017's DEC-2 is corrected: no `route_decision`
fixture is harvestable, because none is ever persisted. `harvest` unblocks
D1/D4-shaped fixtures (real `analysis.md`/report content) instead — a real,
useful subset of what DEC-2 hoped for, but not literally recorded route
decisions. Persisting `route_decision` records is a distinct, unscoped
future decision about Scout's own behavior, not part of this mission.

**Discovery mechanism:** `harvest` calls `treasure.ScanMissions(basePath)`
exactly as `treasure_chest_scan.go` already does (via
`resolveDojoRoots(root)` for `basePath`), performing zero modification to
`internal/treasure/scan.go`. `ScannedMission` carries no file path, so
`harvest` resolves each selected mission's directory itself
(`refined/<id>/` or `done/<id>/`, whichever exists) for the copy step; `adr`/
`report` artifact types resolve from `archived/<id>-adr.md` /
`archived/<id>-report.md` per the existing artifact contract.

### Alternatives Considered and Rejected

- **Extract named sections instead of whole files** (DEC-1). Rejected:
  no per-artifact-type section-parsing exists yet; whole-file copy is
  lossless and sufficient for the existing content-assertion harness.
- **Auto-generate scenario test stubs** (DEC-2). Rejected: risks
  low-value/incorrect generated tests and further blurs the Scope
  Invariant; fixture supply and scenario authorship stay decoupled.
- **Batch-only harvesting** (DEC-3). Rejected: most real usage is
  single-mission and deliberate; forcing a full scan every run wastes time
  and risks noisy bulk fixture dumps.
- **Versioned fixture copies per harvest run** (DEC-4). Rejected: git
  already tracks fixture history; a bespoke versioning scheme would need
  its own retention policy for no added benefit.
- **Leave the prior ADR's "harvest a route_decision" framing as-is**
  (DEC-5). Rejected: a future mission re-reading that ADR without this
  correction would expect a fixture type that structurally cannot exist,
  wasting discovery effort re-deriving the same finding.

## Consequences

- `strategist eval harvest` becomes buildable without any change to
  `internal/treasure/**` — pure reuse, minimal risk surface.
- `tests/evals/regression/` gains a defined write-target convention
  (`<mission_id>/<artifact>.md`) that Phase 2's `TargetArtifactCheck`-based
  scenarios can consume identically to the hand-authored fixtures already
  added.
- The B1–B3 recorded-`route_decision` sub-cases from ADR-0017 remain
  blocked after this mission — not by this design's limitation, but because
  the source data was never captured. Future missions should not expect
  `harvest` to produce this fixture type until a separate decision is made
  to persist `route_decision` records.
- Implementation (T2–T6's code items) remains `implementation_handoff` —
  outside this Strategist mission's execution scope, per the Scope
  Invariant.
