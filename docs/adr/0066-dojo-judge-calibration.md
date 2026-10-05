# ADR-0066 — DOJO judge calibration

**Status:** Accepted (implemented in waves; see the package status note)
**Date:** 2026-10-04
**Mission:** `20261004-dojo-calibration-training` (refined package, amendments 001 and 002)
**Related:** [ADR-0065](0065-provider-integration-mechanism.md), [ADR-0018](0018-eval-harvest.md), [ADR-0050](0050-confidence-governance-contract.md)

## Context

The provider pre-check approves a handoff when the provider's probability of accept is above a
threshold. The default (0.80) came from one live look. The DOJO already owns durable scenarios, a
run mode for the agent and durable learning, but it can only assert structure: it cannot express a
judgment, a rate or a calibration. Labeled data is almost all positive, so a calibration needs
generated negatives.

## Decision

1. **The DOJO is the surface; a pure evaluator does the work.** A new scenario kind,
   `judge_calibration`, runs a case. The evaluation lives in `internal/calibration` with the
   provider behind a port; `eval` is only a corpus and content-assertion source. No separate
   subsystem and no third scenario format.
2. **Vocabulary.** Cases, fixtures, evaluation, calibration. Nothing is trained.
3. **Cases.** A package is a `candidate` until a reviewed promotion makes it `golden` with
   provenance and an oracle; threshold selection reads golden cases only. Negatives are the
   amendment `before` snapshots, deterministic mutations (paired with their parent, with graded
   levels) and cases near the threshold; neutral perturbations measure false rejection. A real
   mission where the provider and the outcome disagree becomes a regression candidate.
4. **Observations.** Every call is one content-free line; rates, intervals and thresholds are
   recomputed from them. A passing run produces evidence; only a failing run produces a lesson.
5. **Threshold as a product.** Fitness gates run in order (separation, monotonic response,
   stability); a threshold is recommended only for a fit judge, under an explicit risk policy, by
   the upper confidence bound of false approval, and only when the sample supports it.
6. **The handoff looks it up.** A calibration record (`strategist-calibration/v1`) is bound to
   provider, resolved model, question profile and version, corpus, catalog, seed and risk policy.
   Operator configuration pins an id (never "latest"); the pre-check uses it only when valid and
   otherwise uses the configured default and reports `não calibrado`.
7. **Consent.** A run that sends real package fields needs an explicit per-run consent flag, a
   dry run shows the exact fields and bytes first, and observations carry no content.

8. **External calls only through the DOJO.** The DOJO is the sanctioned path to call a provider for a check. A run prints a
   consent term (what leaves, to where, how much), needs `--consent`, can bind the assent to the exact plan, and keeps the term in
   its evidence. `dojo check` never calls out; a synthetic case (`dojo generate --synthetic`) calibrates over fixtures that are
   golden by construction; a live case (`--layer live`) is the provider smoke check. No make target, test or CI workflow calls a
   provider by default: the Go live targets are outside `.PHONY` and every dependency chain, and need `STRATEGIST_LIVE_CONSENT=yes`.

9. **Question profiles.** `handoff-v1` projects only the fields an artifact has; `handoff-v2` also shows a missing allowed field as
   `(absent)`. The first calibration found that the judge ignores defects that remove something (a missing acceptance check scored
   higher than the intact package), which v2 is meant to test. A case picks its profile independently of the provider's configured
   one; a calibration is valid only for its profile and version, and the v1 version is unchanged.

10. **Laboratory.** Stored observations can be re-evaluated offline (`--recompute`), compared across cases (`dojo compare`) and
    evaluated without chosen operators as a labelled what-if that persists nothing and never yields a record. The monotonic gate
    scales its tolerance with the measured repetition noise and names the operator that inverts, and a record keeps the operator
    list its case restricted itself to.

11. **Gate rule gates-2, local checks and the probability contract.** After the first four pilots: a margin gate, an inversion rate
    under an exact bound, correlated mutants treated at their effective sample, golden positives counted per parent, and the AUC printed
    with its difference to the limit. Deterministic local checks are measured like the judge and an operator the judge cannot see can be
    delegated to them, with a coverage gate. A calibrated threshold is on the probability of accept and the confidence is never
    substituted for it. A requirement-pair layer asks atomic questions per dimension to decide which dimensions the provider may judge.
    The first pilots stay reported under `gates-1`.

12. **Gate rule gates-3 and the atomic package profile.** The first lab runs (32 synthetic packages) showed a single held-out split of
    seven positives confirms nothing: it hid the one defect the judge approved. `gates-3` reads the whole corpus by a five-fold
    cross-validation grouped by parent, judged on upper bounds, adds a threshold-band gate against the repetition noise and centers the
    threshold in its band; it is pre-registered before the real corpus and `gates-2` stays the default of a case without `gate_rule`.
    The profile `handoff-atomic-v1` asks one noul per criterion and takes the lowest answer, to be measured against `handoff-v2` on the
    same packages; the handoff runs it when the provider is configured with it (`profile: handoff-atomic-v1`); the local rule derives the aggregate from the lowest criterion answer, so a pinned calibration of it decides the threshold like any other, and a missing criterion answer never approves.

13. **Shadow mode.** Before a calibration is trusted on real handoffs, `integrations shadow on` keeps the provider call but takes the
    decision away from it: each reading is recorded in the call ledger with what it would have decided, the handoff proceeds on the main
    path with no delegation, no hint and no confidence claim, and a shadow reading is never reused as a delegation. `integrations shadow
    report` summarizes the readings (numbers only) and `dojo candidate --mission` compares a mission's reading with its gate outcome.

14. **What the first real run taught.** On real packages the aggregate could not be certified (not_fit twice), because the reviewed golden packages did not
    carry the schema's `acceptance_checks` and `approved_scope` and some lacked a verification per task; the judge itself separated the defects per criterion. The
    DOJO therefore gained a tolerant task grammar for its local checks and operators, a human `dojo demote` (promotions stay append-only), an offline `dojo corpus`
    screen, and a per-criterion calibration (rule criteria-1) that says which questions are reliable enough to raise a hint. The Archivist gap is a pending demand.

## Consequences

- The default threshold stays an uncalibrated starting point until a valid record is pinned.
- The DOJO grows a case vertente with its own configurable output paths; existing scenarios and
  results are unchanged.
- Certifying other providers and the deferred PRECISE-SHOT and SHARPSHOOTER roles stays out of
  scope; the evaluator port keeps it possible.
