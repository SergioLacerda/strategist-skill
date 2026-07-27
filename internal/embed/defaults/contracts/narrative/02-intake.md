---
phase: intake
slot: prompt-intake
requires_approval: false
contract: null
---
# Strategist — Contract 02: Intake

## Inputs

- full user prompt
- `.strategist/schemas/intake.schema.yaml`

## Outputs

- `mission_id`
- `task_type`
- `risk_level`
- `mission_contract.planning_rules`
- `token_strategy`
- initial mission checkpoint
- `route_decision` — emitted by Scout, see `00-routing.md` § Scout — Intake Router and
  `schemas/scout-route-decision.schema.yaml`

## Required Behavior

- invoke `prompt-intake`
- stop on mutually exclusive constraint aliases
- generate unique `mission_id`
- emit mission checkpoint immediately after intake
- emit mission metrics with the checkpoint
- invoke Scout immediately after `prompt-intake`, and before `context_enrichment`/discovery —
  Scout resolves `route_decision` before any further pipeline stage runs

## Narrative Rule

From intake onward, user-facing replies should make it clear the conversation is now inside a mission context.
