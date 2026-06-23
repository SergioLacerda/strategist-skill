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

## OTel Rule

- INFO/WARN/ERROR/FATAL are rendered via output profile
- DEBUG/TRACE remain structured telemetry
- profile rendering must not rewrite structured telemetry payloads

## Coverage Policy

- if a field is not yet emitted by runtime code, document the gap explicitly
- contract updates should keep `internal/telemetry/schema.go` in sync
