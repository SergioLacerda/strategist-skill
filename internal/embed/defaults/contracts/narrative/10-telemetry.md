---
phase: telemetry
slot: null
requires_approval: false
contract: null
---
# Strategist — Contract 10: Telemetry

## Goal

Keep the human narrative and the structured telemetry aligned.

## Canonical Event Payload

Structured telemetry should preserve, when available:

- `phase`
- `status`
- `component`
- `mission_id`
- `artifact_path`
- `selected_skill`
- `runtime_mode`
- `output_profile`
- `gate.type`
- `gate.status`
- `gate.response`
- `transition_group`
- `reason`
- `role`
- `route`
- `route_reason`
- `route_confidence`
- `evidence_state`
- `discovery_subtype`
- `provider`
- `model`
- `effort`
- `level_source`
- `ability`
- `initiative.advice_id`
- `initiative.policy_version`
- `initiative.policy_digest`
- `initiative.trigger`
- `initiative.alignment`
- `initiative.confidence_ceiling`
- `initiative.result_status`
- `initiative.evidence_refs`
- `initiative.outcome_ids`
- `initiative.observed_model`
- `initiative.observed_provider`
- `initiative.observed_effort`
- `initiative.observed_level_source`
- `initiative.recommended_capability`
- `initiative.recommended_effort`
- `initiative.advice_reused`
- `initiative.supersedes`
- `initiative.deviation_ids`
- `initiative.challenge_reasons`
- `initiative.source_role`
- `handoff_challenge.status`
- `handoff_challenge.critical_failures`
- `handoff_challenge.types`
- `confidence.policy_version`
- `confidence.event_id`
- `confidence.agent`
- `confidence.correlation_key`
- `confidence.claim_kind`
- `confidence.confidence_level`
- `confidence.confidence_percent`
- `confidence.evidence_class`
- `confidence.evidence_ids`
- `confidence.evidence_classes`
- `confidence.sample_size`
- `confidence.calibration_status`
- `confidence.ground_truth_ref`
- `confidence.ground_truth_kind`
- `confidence.ground_truth_outcome`
- `confidence.coverage_status`
- `confidence.missing_reason`
- `confidence.violation`

## Role Level Fields

`model`, `effort` and `level_source` (`host` | `policy`) identify the level a role
runs at and back the role-line level label. They are recorded for every role
scope (Scout, Ranger, Archivist, Sniper, transport) and are `null` when no level
is known — a missing level never blocks a mission. Within one phase every event
carries the same tuple unless an escalation is recorded with its `reason`.

At each role's phase start, run the role's `on_start` commands (declared in
`roles/<id>.yaml`; the default is `strategist leveling label --role {role}
--mission {mission_id}`, pass `--host-model`/`--host-effort` when the host
reports them). A repeated role in one mission, such as an Archivist revision
loop, passes `--run <n>` so each run keeps its own level. The command records
the tuple in `.strategist/memory/role-levels.jsonl`, and later calls for the
same mission and role reuse it; record an escalation with `--reason escalated`.
Two forms show the level. Content templates (`<role>_start`, `<role>_done`,
`<role>_task_done`, the gate prompt) use the stacked `{role_level_header}`; the
short narration lines (`phase_announcements`) name the role with the inline
`{role_level_tag}`, the `tag` field of `strategist leveling label --json`
(`(Sonnet-High)`, empty when the level is unknown), for example
`🎯 **Ranger(Sonnet-High):** ...`. Scout has a narration line (`scout_done`); the
gate is not a role and carries no level. The per-role content keys are generated
at compile time from generic `role_start`, `role_done` and `role_task_done`
templates plus a phrase table, one set per registered role, so a new role needs
no new template strings and the old keys stay valid as aliases. Each newly recorded
tuple also emits the `role_level_resolved` event (DEBUG) with `role`, `model`,
`effort`, `level_source` and, for an escalation, `reason`; the Archivist's tuple
is repeated on its `handoff-metrics.jsonl` line.

The `leveling:` block of `active.yaml` (`mode: manual | automatic`; absent means
automatic) selects the resolution. LEVELING is an internal ability and the
install wizard leaves this optional block absent for new installations.
Explicit modes remain runtime compatibility settings. Manual is host passthrough: only
host-reported values are used, the LEVELING policy is never read, and a value
the host does not report stays unknown. Automatic uses host-reported values,
then the LEVELING policy; `leveling.yaml` plus the install authority are read
only when a model or effort is still missing. A complete host report never
reads the policy.

## Mission View

`strategist mission view --mission-id <id> [--run <id>] [--json]` is a
read-only projection over mission state, role registry, confidence history,
Approval Gate labels, and the LEVELING ledger. It keeps the role journey and
the Gate as distinct entries; confidence is always advisory evidence and never
authorizes execution. JSON uses `strategist-mission-view/v1` and represents
missing secondary data explicitly as `unavailable`, `unknown`, or
`not_applicable`.

New LEVELING ledger lines may include provider, model/effort sources,
capability, fallback metadata, and policy identity. These fields are additive:
legacy JSONL remains readable and its missing provenance is shown as unknown,
never reconstructed from the current policy. Confidence records may likewise
carry an optional explicit `run`; records without it remain mission-wide.

INITIATIVE has a separate append-only ledger at
`.strategist/memory/initiative-records.jsonl`, correlated by `mission_id`,
`role`, `run_id`, and `advice_id`. It never writes `role-levels.jsonl` and never
replaces the LEVELING tuple. A role resolves one advice envelope at entry and
reuses its `advice_id` for local actions. Re-evaluation triggers create a new
record with `supersedes`; prior advice remains immutable. `recommended_effort`
and `recommended_capability` are advisory names only, not execution selectors.
Missing evidence is represented as `unknown` or `unavailable`, and a blocked
obligation may lower the confidence ceiling or challenge the handoff without
authorizing or rejecting the Approval Gate.

The runtime emits the corresponding diagnostic events to
`.strategist/memory/initiative-events.jsonl`. These events carry INITIATIVE
observations and recommendations under `strategist.initiative.*`; they never
overwrite the LEVELING `model`, `provider`, `effort`, or `level_source` fields.
This local JSONL stream is the authoritative default INITIATIVE event sink;
hosts may inject another `EventSink`, but delivery failures are surfaced to the
role boundary rather than silently reported as complete advisory evidence. The
mission-start adapter consults the declared Scout hook, while advisory
handoffs consume the declared downstream hook before the mission transition is
applied.

## Scout Event

Scout's route-decision events are distinguished from Ranger's discovery-result
events by `component`:

- `component: scout`, `phase: intake` — route classification (`role: Scout`,
  `route`, `route_reason`, `route_confidence`, `evidence_state`,
  `discovery_subtype`, `provider`). See `contracts/machine/scout-routing.yaml`.
- `component: ranger`, `phase: discovery` — discovery results, including
  `evaluation_verdict` when `discovery_subtype: evaluation`.

These are always separate events — a Scout route decision is never merged into a
Ranger discovery-result payload, and vice versa.

## OTel Rule

- INFO/WARN/ERROR/FATAL are rendered via output profile
- DEBUG/TRACE remain structured telemetry
- profile rendering must not rewrite structured telemetry payloads
- CLI commands that create spans must start from the Cobra command context via
  `commandContext(cmd)`, not from a fresh `context.Background()`.
- Install, compile, check, validate, and sync-governance spans must preserve
  the incoming mission context so `MissionRun` counters and child spans stay
  connected to the current invocation.
- Telemetry setup and shutdown may use `context.Background()` because they run
  before or after a command invocation context exists.
- When OTLP is disabled, the no-op tracer provider must keep the same context propagation behavior and must not open network connections.

## Handoff Challenge Event

When Archivist -> Sniper `handoff_verification` is evaluated, telemetry should preserve:

- `strategist.handoff_challenge.status` (`required`, `skipped`, `passed`, `failed`)
- `strategist.handoff_challenge.critical_failures`
- `strategist.handoff_challenge.types`

These attributes are diagnostic. They never imply Approval Gate acceptance and never
authorize Sniper materialization.

INITIATIVE result attributes are equally diagnostic. They correlate diligence,
alignment, evidence references, deviations, and outcome observations, but they
cannot authorize `implementation_handoff` or bypass the independent Approval Gate.

Confidence telemetry is comparable across agents only through the shared envelope
above. Scout's `route_confidence`, critic scores, Mission Quality, timing, and
handoff rates retain their own meanings and must not be converted into claim
confidence. Calibration accuracy requires an explicit human revision, handoff
validation, or downstream verification label; unlabeled records remain
`uncalibrated`.

Confidence history is owned by the Strategist runtime at
`.strategist/memory/confidence-records.jsonl`. The v1 calibrated state requires
at least three reviewed outcomes; observe-mode may report `no_sample`,
`uncalibrated`, or `observed` without blocking the human gate.

## Coverage Policy

- if a field is not yet emitted by runtime code, document the gap explicitly
- contract updates should keep `internal/telemetry/schema.go` in sync
- the current Go runtime exposes the INITIATIVE domain and contract fields; the
  production mission-start and handoff adapters invoke declared lifecycle hooks,
  while absent provider evidence remains explicit rather than being synthesized
  as execution-level data

## Chest Event Naming

`treasure_chest_loaded` and `treasure_chest_found` are two intentionally distinct events, not
naming drift:

- `treasure_chest_loaded` (DEBUG, `machine/context-enrichment.yaml`): fires on every slot
  chest-consult step, including the empty case (`treasure_chest_loaded none`). Internal
  machine/debug observability signal, `render_policy: debug_bypass`.
- `treasure_chest_found` (INFO, `schemas/progress-contract.yaml` user-facing signal): fires
  only when a non-empty chest list is passed to a slot. Visible persona/chat output.

Both are kept as-is. No rename or consolidation without an explicit approval-gate extension.
