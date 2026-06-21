# Strategist — Contract 00: Routing

## Purpose

Resolve the route before any mission work starts.

## Routes

- **Quick Draw** — only for explicit quick capture / note append requests
- **Main mission** — every other request

## Main Mission Sequence

`bootstrap → preflight → intake → discovery → refinement → approval_gate → execution? → adr? → learning`

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
- No execution without explicit approval
- No slot work performed by Strategist itself
- External governance (any adapter) may block or permit execution — it does NOT replace the canonical pipeline sequence once Strategist is invoked
- `execution_gate=allowed` from governance never substitutes the persona gate (explicit user approval)
