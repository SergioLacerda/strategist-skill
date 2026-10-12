# Runbook: Producing a Runbook

## Trigger

A mission produced, or an operator verified, a reusable agent procedure that should outlive the mission, and
someone has to turn it into a runbook that the selector can offer to later missions. Typical signals: a repeated
failure with a known recovery sequence, a runtime or governance diagnostic an agent should be able to run
without broad discovery, an executable decision procedure, or a lookup rule that prevents stale behavior. An
`OA-RUNBOOK-<mission_id>` side quest appearing at the Approval Gate is the usual entry point.

Not a trigger: a generic idea with no procedure, a one-off implementation task, a feature request, a claim
whose correctness was never validated, or an architecture trade-off (that is an ADR).

## Steps

1. **Check for duplicates.** Run `strategist runbook select --format json --requester archivist
   --signal <signal> [--signal ...]` with the mission's task type and salient keywords. The command
   queries the admitted external Treasure Chest through JEWELER; inspect the returned source paths.
   If a runbook already covers the procedure, extend it instead of creating another.
2. **Confirm the procedure is verified.** Every command, path and expected output the runbook will state must
   have been run or read in the mission. An unverified step is marked as unverified in the text, never smoothed
   over.
3. **Let the Opportunity Attack surface the candidate.** After refinement the Archivist evaluates the runbook
   criteria (`.strategist/contracts/machine/opportunity-attack.yaml#on_runbook_criteria_met`). When they hold,
   `internal/tools/prepare_runbook` prepares a reviewable candidate bound to the package digest. Its status is
   `prepared`, `duplicate`, `insufficient_evidence` or `needs_review`; none of them is a canonical write.
   The typed candidate has the same shape as a `.runbook.yaml` sidecar.
4. **Decide at the Approval Gate.** Accept or decline the side quest with
   `strategist mission accept-side-quest --mission-id <id> --side-quest OA-RUNBOOK-<id>` or
   `strategist mission decline-side-quest ...`. Declining keeps the procedure as backlog.
5. **Know what acceptance materializes.** At execution entry the runbook resolver prepares a group in the
   configured root `runbooks.canonical_path` of `.strategist/active.yaml` (the wizard default is
   `<base_path>/runbooks`, here `.analysis/runbooks`; without a configured root the legacy path
   `docs/runbooks/<mission_id>-runbook.md` is used): the markdown `<mission_id>-runbook.md` and the typed sidecar
   `<mission_id>-runbook.runbook.yaml`, whose `runbook_id` is the markdown stem and whose `source_doc` is the
   markdown path. Sniper completion parses the sidecar with the same parser the selector uses and refuses a
   mission whose sidecar would be skipped.
6. **Review and promote (manual step).** A human reviews the materialized markdown. Rename it to
   `<slug>.md`, where the slug is a short stable name, and write the typed sidecar `<slug>.runbook.yaml` next to
   it. The sidecar schema and the authoring guidance are in `docs/runbooks/README.md` (section "Typed
   sidecars"). Required fields: `schema_version`, `runbook_id` (equal to the markdown stem), `runbook_type`
   (`analytical` or `operational`), `source_doc`, `objective` and a non-empty `applies_when`; classify each
   `checks[].level` from the document's own language.
7. **Place and index it through JEWELER.** The external Treasure Chest owns the provider index and its
   maintenance lifecycle. Register or refresh the provider's runbook index through the authorized JEWELER
   maintenance path; do not make the local sidecar selector the production source of truth.
8. **Verify visibility.** Run `strategist runbook select --format json --requester archivist
   --signal <a signal from applies_when>` and expect the new runbook in the result. A missing, stale,
   malformed or unhealthy provider binding is an error and must be repaired before closing; no local
   fallback is permitted.
9. **Chest and index.** If the runbook lives under a root that no registered chest covers, register it with
   `strategist treasure-chest add <path> --id <id> --trust-tier T2 --scope all` only after the operator
   approves; the Opportunity Attack never registers a chest, and `strategist treasure-chest doctor` shows
   the drift when a registration is incomplete.

## Decision Point

Three design questions were open when this runbook was written. The operator decided D1 and D2 at the
Approval Gate of mission `20261011-runbook-sidecar-pattern-and-generation` (2026-10-11); all three are now
implemented.

| Id | Decision | State |
|---|---|---|
| D1 | A runbook is a markdown plus a typed `.runbook.yaml` sidecar; the `.meta.json` placeholder is gone. | implemented: the runbook resolver emits `<stem>.runbook.yaml` (kind `runbook_sidecar`) and completion parses it |
| D2 | The local selector reads the path configured in the wizard (`runbooks.canonical_path`) first and the legacy `docs/runbooks` as a fallback; the configured root wins a duplicate `runbook_id`. | implemented in the local corpus loader behind `strategist runbook select --allow-local-staging` (the production lookup goes through JEWELER, which owns its own index); an unreadable `active.yaml` degrades to the legacy root with a `runbooks_root_unavailable` warning |
| D3 | The materialized markdown name, `runbook_id` and `source_doc` agree without a manual rename. | implemented: `runbook_id` is the markdown stem, `source_doc` its path |

Promotion to a short slug (step 6) is now optional polish, not a repair: a materialized runbook already
satisfies the sidecar contract.

## Stop Conditions

- The procedure cannot be verified against real command output or current file content: do not write it.
- A runbook with the same `runbook_id` already exists in the root: extend it, do not duplicate it.
- The admitted provider does not offer the runbook for its own signal, or JEWELER reports an unavailable
  binding/readiness state: fix before closing.
- The same gate lists `OA-CHEST-<mission_id>`: do not accept it while the accepted-chest defect is open (an
  accepted chest leaves the mission without a way forward; see `.analysis/refined/20261011-refinar-oa-adr-derived-target-bridge/oa-chest-dead-end-handoff.md`).
- An operator declines the gate: leave everything where it is.

## Reference

- `docs/runbooks/README.md`: canonical roots, typed sidecar schema, authoring guidance, lookup order,
  candidates versus accepted runbooks.
- `.strategist/contracts/machine/opportunity-attack.yaml`: `runbook_worthy` signals, `on_runbook_criteria_met`,
  `preparation`.
- `internal/runbook.ParseSidecar`: the single sidecar parser; `internal/tools/prepare_runbook`: candidate
  preparation; `internal/mission/execution_resolver.go`: what acceptance materializes;
  `treasure-chest/cli/runbook_select.go`: the selector corpus.
- `.analysis/refined/20261011-runbook-sidecar-pattern-and-generation/`: the diagnostic and the implementation
  handoffs 3.1 to 3.3 that retire the Decision Point.
- `docs/runbooks/verifying-implemented-demands.md`: the closest procedural runbook, as a shape example.
