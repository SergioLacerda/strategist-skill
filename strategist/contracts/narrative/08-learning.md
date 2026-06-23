---
phase: learning
slot: learning-curator
requires_approval: false
contract: null
---
# Strategist — Contract 08: Learning

## Inputs

- mission result
- critic evaluation
- `task_type`

## Outputs

- optional learning checkpoint
- outcome line appended to `.strategist/memory/outcomes.tmp`

## Required Behavior

- learning is non-blocking
- learning-curator must present a checkpoint before any memory write
- outcome recording keeps `outcomes.jsonl` as source of truth
- semantic retrieval remains optional and rebuildable
