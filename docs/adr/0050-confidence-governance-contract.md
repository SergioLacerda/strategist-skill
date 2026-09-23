# ADR-0050 — Versioned Confidence Governance Contract

**Status:** Accepted
**Date:** 2026-09-20
**Related:** ADR-0047, ADR-0048

> 2026-09-22 reinforcement: the Approval Gate stopped inlining the full
> cross-agent calibration payload (policy version, distribution, per-agent
> metrics, evidence coverage, calibration status, missing/rejected/duplicate
> counts) in the chat prompt. That payload remains this contract's
> materialization of `LoadConfidenceGateReview`, unchanged in shape — it is
> now read via the pre-existing `strategist metrics confidence --mission <id>`
> / `strategist mission view --json` commands instead of being duplicated
> into gate prose. The gate itself shows a per-item summary (task assertions
> with `confidence_percent`, open questions with no percent, side quests with
> `confidence_percent` or an honest `investigation_required` state) so the
> low-confidence-must-be-a-question principle below extends visibly to side
> quests, which previously had no confidence signal at all. See
> `contracts/narrative/05-approval-gate.md` and
> `.analysis/pending/20260922-approval-gate-confidence-summary/`.

## Context

The workspace carried several unrelated confidence-like signals: the
`low`/`medium`/`high` labels on Decision and Evidence, `route_confidence`,
intake and routing thresholds, critic scores, Mission Quality and handoff
pass rates. They shared no percentage semantics and no rule separating a
question from an evidence-backed assertion, so a percentage could be read as
a calibrated probability and empty metrics could look like a measured 0%.

## Decision

`machine/confidence-governance.yaml` (policy `v1`) is the single contract:

- `confidence_percent` is an integer in [0, 100]; levels are derived
  (low 0–59, medium 60–84, high 85–100). A label that contradicts its
  percentage is rejected.
- `claim_kind` is `question` or `assertion`. Questions stay questions through
  every handoff (correlated by `correlation_key`) until explicitly resolved.
- Assertions require resolvable evidence with a supporting class. Low-confidence
  assertions are invalid; medium ones are provisional and cannot authorize
  high-risk or destructive work; high ones need explicit evidence or two
  independent corroborated sources and no unresolved contradiction.
- `sample_size` and `calibration_status` (`no_sample`, `uncalibrated`,
  `observed`, `calibrated`) keep policy percentages distinct from empirical
  calibration. An empty history is `no_sample`; `calibrated` needs at least
  three reviewed samples and an explicit ground-truth reference.
- Confidence is advisory gate input. It never replaces the Strategist
  Approval Gate or the local execution gate, never invokes Sniper, and
  `implementation_handoff` items stay separate from `documentation_target`.
- Route confidence, critic score, Mission Quality, throughput, timing and
  handoff pass rate remain independent metrics owned by their producers.

Enforcement starts as `advisory`. Blocking requires reviewed compatibility
evidence; rollback is a switch back to advisory that preserves confidence
records and does not bypass or auto-accept the gate.

## Consequences

- Validation lives in `internal/domain` (claims, summaries, handoff
  comparison) and `internal/telemetry` (records, metrics, gate review).
- Legacy `evidence` is read as an alias of `evidence_ids` only when values do
  not conflict; legacy records never imply ground truth or calibration.
- Authoring defaults under `internal/embed/defaults/` are the source; the
  `.strategist/` runtime mirrors them and parity is checked by tests.
