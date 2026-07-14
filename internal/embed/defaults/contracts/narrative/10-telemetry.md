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
