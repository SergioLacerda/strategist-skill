# Runbook: Confidence observe-mode review before blocking enforcement

## Trigger

Someone wants confidence enforcement to move from `advisory` to `blocking`
(`machine/confidence-governance.yaml#rollout`), or asks whether that is
justified. Enforcement stays advisory until this review exists.

## Steps

1. Run the pipeline in observe mode (`enforcement: advisory`) for enough
   missions that each producing boundary has records. Producers persist through
   `strategist metrics record`; gate outcomes through
   `strategist metrics gate-outcome`.
2. Inspect coverage per agent with `strategist metrics confidence` and, per
   mission, `strategist metrics confidence --mission <id>`. Check that
   `missing_records`, `rejected_records` and `duplicate_records` are explained,
   and that every agent's denominators reconcile with its own sample size.
3. Confirm calibration is not claimed without evidence: `calibration_status`
   must be `no_sample` or `uncalibrated` unless ground truth (human gate
   outcomes, reviewed labels) supports more.
4. Replace the inadmissible baseline at
   `docs/evidence/confidence-observe-review.yaml` with a human-reviewed record:

   ```yaml
   schema_version: "1"
   status: ready
   reviewed_by: <human reviewer>
   reviewed_at: 2026-09-20T12:00:00Z
   compatibility_evidence_ref: <link or path to the compatibility evidence>
   agent_denominators_reconciled: true
   samples_per_agent: {scout: 4, ranger: 4, archivist: 4}
   ```

5. Verify admissibility:

   ```bash
   strategist metrics rollout-check --enforcement blocking \
     --review-file docs/evidence/confidence-observe-review.yaml
   ```

## Decision Point

- **Admissible:** the command prints `enforcement blocking admissible`. Changing
  the contract is still a reviewed change to `internal/embed/defaults/`.
- **Refused:** it exits non-zero with the reason and enforcement stays
  advisory. Do not edit the review to satisfy the check; collect the evidence.

## Rollback

`strategist metrics rollout-check --enforcement advisory` is always admissible.
Rolling back sets `rollout.enforcement` to `advisory`; it preserves
`confidence-records.jsonl` and `ground-truth-labels.jsonl` and never bypasses
or auto-accepts the human Approval Gate.

## Stop Conditions

- The reviewer cannot explain rejected or missing records.
- Any agent's denominator does not reconcile.
- Blocking would authorize execution: confidence is advisory and cannot invoke
  Sniper or replace the Approval Gate.

## Reference

- `internal/embed/defaults/contracts/machine/confidence-governance.yaml`
- `docs/adr/0050-confidence-governance-contract.md`
