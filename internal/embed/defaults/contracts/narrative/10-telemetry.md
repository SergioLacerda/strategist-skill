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

`model`, `effort` and `level_source` (`manual` | `host` | `policy`) identify the level a role
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

Resolution order is manual configuration, then host-reported values, then the
LEVELING policy. Manual values come from the `leveling:` block of `active.yaml`
(`mode: manual | automatic`; for manual, a per-role `model`/`effort` map); an
absent block means automatic. LEVELING data is read on demand: the `leveling:`
block first, and `leveling.yaml` plus the install authority only when a model or
effort is still missing in automatic mode. A complete manual map, or a complete
host report, never reads the policy.

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

## Chest Event Naming

`treasure_chest_loaded` and `treasure_chest_found` are two intentionally distinct events, not
naming drift:

- `treasure_chest_loaded` (DEBUG, `machine/context-enrichment.yaml`): fires on every slot
  chest-consult step, including the empty case (`treasure_chest_loaded none`). Internal
  machine/debug observability signal, `render_policy: debug_bypass`.
- `treasure_chest_found` (INFO, `schemas/progress-contract.yaml` user-facing signal): fires
  only when a non-empty chest list is passed to a slot. Visible persona/chat output.

Both are kept as-is. No rename or consolidation without an explicit approval-gate extension.
