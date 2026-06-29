---
phase: routing
slot: null
requires_approval: false
contract: null
---
# Strategist — Contract 00: Routing

## Purpose

Resolve the route before any mission work starts.

## Routes

- **Quick Draw** — only for explicit quick capture / note append requests
- **Critical Hit** — fast path for moving `.md` analysis artifacts between `pending/`, `refined/`, `archived/` (see `critical-hit.yaml`)
- **Implementation Short Route** — for already-refined implementation/materialization requests
- **Main Mission** — every other request

## Route Selection Order

1. Quick Draw keywords detected → Quick Draw
2. Critical Hit conditions satisfied (see `critical-hit.yaml`) → Critical Hit
3. Implementation Short Route conditions satisfied → Implementation Short Route
4. Default → Main Mission

**When in doubt → Main Mission. Conservatism is the safe default.**

## Main Mission Sequence

`bootstrap → preflight → intake → discovery → refinement → approval_gate → execution? → adr? → learning`

## Critical Hit Sequence

`bootstrap → preflight → intake → critical_hit_gate → execution → learning`

## Implementation Short Route

For requests that are already framed as implementation/materialization and arrive with sufficient context:

`bootstrap → preflight → intake → implementation_context_validation → approval_gate → execution_provider_resolution → execution/materialization → learning`

Short route conditions (ALL must hold):
- request explicitly asks for implementation/materialization
- local context or user prompt provides enough scope to avoid full discovery
- documentation/materialization targets are clear
- no unresolved ambiguity
- code mutation is not required
- Git mutation is not required

This route skips full Ranger/Archivist expansion only when context is already refined enough. It does NOT skip the Strategist Approval Gate. If any condition fails, fall through to Main Mission.

Still delegates execution to the resolved provider. Direct execution by the Strategist shell is never permitted.

## Contract Lookup

When operating inside the main mission, consult contracts in this order:

1. `01-bootstrap.md`
2. `02-intake.md`
3. `03-discovery.md`
4. `04-refinement.md`
5. `05-approval-gate.md`
6. `06-execution.md`
7. `07-adr.md`
8. `08-learning.md`
9. `09-response.md`
10. `10-telemetry.md`

## Invariants

- No direct repository mutation without canonical pipeline evidence
- No execution without explicit Strategist Approval Gate acceptance
- No slot work performed by Strategist itself
- The invoking local context (any adapter, orchestrator, or harness) may block or permit execution — it does NOT replace the canonical pipeline sequence once Strategist is invoked
- `execution_gate=allowed` from local context never substitutes the Strategist Approval Gate (explicit user approval)
- The Strategist Approval Gate is required on all routes: Quick Draw, Critical Hit, Implementation Short Route, and Main Mission
- A missing or uncallable resolved execution provider is a blocked state — never a reason for direct execution
