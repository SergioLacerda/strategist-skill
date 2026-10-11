# Telemetry interoperability contract: mission lifecycle and usage

**Date:** 2026-10-11

**Status:** Accepted (ADR-0069, Wave 4); implemented for the lifecycle and usage evidence of a mission.
Scope: how Strategist exposes that evidence to other tools, and what a consumer may rely on.
This contract does not make Strategist depend on any of them.

## 1. Principles

1. **The ledgers are the source of truth.** `memory/mission-lifecycle.jsonl` and
   `memory/mission-usage.jsonl` are written first. Events are an export of records that already
   exist; an export failure never removes or changes a record.
2. **Strategist runs alone.** No PROVIDENCE, collector or external bridge is required. Every
   export path is optional and off by default.
3. **No content.** Events carry ids, counts, phases, roles, models and qualities. There is no
   field for a prompt, a response or a transcript.
4. **Unobserved is not zero.** A token count that was not observed is an absent attribute, never
   `0`.
5. **Dependency direction.** Consumers (a telemetry addon, an audit addon, PROVIDENCE) read the
   contract below. Strategist does not import them and does not call them to decide anything.

## 2. Transports

| Transport | Enable | Notes |
|---|---|---|
| structured log (slog) | default | local, no setup |
| OpenTelemetry | `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_INSECURE`, `OTEL_SERVICE_NAME` | optional; OTLP/gRPC; events travel as **log records** (body = event name, one attribute per field), spans only exist for install, validate, compile, check-stale and sync-governance |
| JSONL event stream | `STRATEGIST_TELEMETRY_JSONL=<file>` | append-only, one event per line; fan-out next to the sink above |
| external governance bridge | explicit adapter | wraps the selected sink; never auto-detected |

Failure policy: lenient by default (`STRATEGIST_TELEMETRY_STRICT` unset): an export failure is
logged and the command continues. With `STRATEGIST_TELEMETRY_STRICT=true` the failure is returned
by the command **after** the ledger was written, with the text "telemetry export failed", so CI
and staging can catch a broken exporter.

## 3. Events

All events use the envelope of `docs/observability-contract.md` (`name`, `timestamp`, severity,
`attributes`, `run_id`, `sequence`, `schema_version`). `run_id` and the attribute
`strategist.correlation_id` equal the mission id. The event timestamp is the instant the record
was observed, so a stream replays the ledger faithfully.

| Event name | Contract id | Written when |
|---|---|---|
| `strategist.mission.started` | `mission-lifecycle/v1` | the mission is created |
| `strategist.mission.phase.started` | `mission-lifecycle/v1` | a canonical phase starts |
| `strategist.mission.phase.ended` | `mission-lifecycle/v1` | a canonical phase ends |
| `strategist.mission.ended` | `mission-lifecycle/v1` | the mission reaches `DONE` |
| `strategist.usage.recorded` | `mission-usage/v1` | one usage observation is recorded |

Attributes common to both contracts: `strategist.mission_id`, `strategist.correlation_id`,
`strategist.phase` (canonical), `strategist.contract_id`, `strategist.authority`.

Lifecycle attributes: `strategist.lifecycle.marker_id` (idempotency key),
`.phase_run_id`, `.end_reason`, `.stage`, `.observed_at`, `.schema_version`.

Usage attributes: `strategist.usage.record_id` (idempotency key), `.event_id`, `.phase_run_id`,
`.role`, `.invocation_id`, `.attempt`, `.scope`, `.session_id`, `.record_kind`
(`incremental` | `cumulative_snapshot`), `.tokens_input`, `.tokens_output`, `.tokens_cache`,
`.tokens_reasoning` (each only when observed), `.provider`, `.model`, `.measurement_source`,
`.measurement_quality`, `.dedup_unavailable`, `.calls`, `.observed_at`, `.schema_version`.

## 4. Semantics a consumer must honor

- **Idempotency.** `marker_id` and `record_id` identify a record. Delivery is at-least-once from
  the consumer's point of view (a stream may be re-read or concatenated): count each id once.
  The same id with different counts is a conflict, not an update.
- **Quality travels with the number.** `measurement_quality` is one of `provider_measured`,
  `host_measured`, `reported_unverified`, `tokenizer_estimated`, `heuristic_estimated`,
  `unavailable`. An aggregate over several qualities is `mixed`, never one of them. Never relabel
  an estimate.
- **Snapshots.** A `cumulative_snapshot` is a running counter of one session; only the delta
  between two snapshots of the same session and counter is consumption. A counter that goes
  backwards yields no interval.
- **Phases.** Canonical phases are the pipeline phases. The Approval Gate has duration and never
  tokens. Display labels are a presentation concern.
- **Windows.** Host-observed records (`host_counter`) are attributed by time window only.
- **Evolution.** New attributes may be added to `v1`; consumers ignore unknown attributes. A
  change of meaning, a removed attribute or a changed key requires a new contract id (`v2`).

## 5. Reconstruction and verification

| Need | Command |
|---|---|
| Rebuild the report from a stream | `strategist mission usage --mission-id <id> --from-events <file>` |
| Check ledgers and stream for gaps, duplicates and conflicts | `strategist mission verify-telemetry --mission-id <id> [--events <file>] [--json]` |

`verify-telemetry` is read-only and advisory. It exits non-zero only when the evidence
contradicts itself (an `error` finding). Finding codes:

| Code | Severity | Meaning |
|---|---|---|
| `unreadable_lines` | warning | lines that are not records; the readers skip them silently |
| `lifecycle_record_invalid` | error | a record fails its own validation (schema, phase, timestamp) |
| `lifecycle_duplicate_marker` | error | the same marker id twice |
| `lifecycle_time_regression` | warning | a marker older than the previous one in file order |
| `lifecycle_end_without_start` | error | a phase ended with no open run |
| `lifecycle_end_before_start` | error | an end earlier than its start |
| `lifecycle_overlapping_run` | error | a run started while another of the same phase was open |
| `lifecycle_run_id_not_contiguous` | warning | `r1`, `r3` |
| `lifecycle_open_run_at_mission_end` | error | a phase still open when the mission ended |
| `lifecycle_engine_mismatch` | warning | the ledger disagrees with the engine state (a lost marker) |
| `usage_record_invalid` | error | schema, phase, counts or a quality stronger than the source allows |
| `usage_duplicate_record` | warning | the same record twice; aggregation counts it once |
| `usage_conflicting_counts` | error | one key with different counts |
| `usage_unknown_invocation` | warning | an invocation id this mission never issued |
| `usage_phase_never_started` | warning | usage in a phase with no lifecycle run |
| `usage_counter_reset` | warning | a cumulative counter went backwards; that interval is unavailable |
| `usage_dedup_unavailable` | info | records with no invocation or event id |
| `usage_unattributed` | info | records outside any phase |
| `event_missing_for_record` | warning | a ledger record with no exported event |
| `event_without_record` | error | an event the ledger does not have |
| `event_duplicate` | warning | the stream repeats an event |
| `event_conflicting_duplicate` | error | the stream repeats a usage event with different counts |
| `event_conflicts_with_record` | error | the event and the ledger carry different counts |

## 6. Degradation scenarios

| Scenario | Ledger | Event stream | Observable as |
|---|---|---|---|
| collector or file sink down, lenient | written | not exported | warning log; `event_missing_for_record` on verify |
| same, strict | written | not exported | the command returns "telemetry export failed" |
| lifecycle ledger not writable | not written | not exported | `warning=mission_lifecycle_marker_write_failed`; `lifecycle_engine_mismatch` or a `gap` row |
| usage ledger not writable | not written | not exported | the command fails |
| stream concatenated or re-read | unchanged | repeated events | `event_duplicate`; consumers dedupe by id |
| host transcript lagging (`observe-usage`) | window recorded | exported | a re-run with more data in the same window is a conflict, never a double count |

## 7. Migration towards separate addons (FORJA / SMITH)

This repository stops at the contract above. A separate telemetry addon and a separate audit
addon can be built on it without changes here:

1. **Telemetry addon:** consumes the JSONL stream or the OTel export, owns retention, fan-out and
   long-term storage, and may add its own sinks. It must preserve ids, qualities and the absence of
   unobserved counts.
2. **Audit addon:** consumes the same stream and the `verify-telemetry --json` report, decides what
   a finding means for governance, and never feeds a decision back into the mission flow through
   Strategist.
3. **PROVIDENCE:** may read the stream through its own adapter. Its token-source vocabulary
   (`env`, `estimated`, `unknown`) does not include a measured source; mapping to the qualities
   above is the adapter's job and must not upgrade an estimate.
4. **Order of work:** (a) keep this contract versioned; (b) let one consumer read the stream in a
   test environment; (c) move retention and cross-mission views out of Strategist only when a
   consumer needs them; (d) never make Strategist call an addon to decide something.

## 8. Not covered

Cross-mission aggregation, retention policy, price tables, budgets and any decision to block a
mission. The usage ledger carries no money values. The only Wave 5 surface is the advisory
per-stage volume block of `strategist handoff criteria`, which reads the ledger and adds no
event or attribute.
