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
- evaluate `routing_matrix` after classification (see `intake.yaml` §routing_matrix):
  - if Quick Draw triggers match → route `quick_draw`
  - else if Critical Hit conditions all satisfied → route `direct_execute` (emit `critical_hit_triggered`)
  - else → route `standard` (main_mission)
- emit `route_selected` event with the resolved route name

## Narrative Rule

From intake onward, user-facing replies should make it clear the conversation is now inside a mission context.
