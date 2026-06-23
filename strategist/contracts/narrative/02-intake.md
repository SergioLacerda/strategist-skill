---
phase: intake
slot: prompt-intake
requires_approval: false
contract: null
---
# Strategist — Contract 02: Intake

## Inputs

- full user prompt
- `strategist/schemas/intake.schema.yaml`

## Outputs

- `mission_id`
- `task_type`
- `risk_level`
- `mission_contract.planning_rules`
- `token_strategy`
- initial mission checkpoint

## Required Behavior

- invoke `prompt-intake`
- stop on mutually exclusive constraint aliases
- generate unique `mission_id`
- emit mission checkpoint immediately after intake
- emit mission metrics with the checkpoint

## Narrative Rule

From intake onward, user-facing replies should make it clear the conversation is now inside a mission context.
