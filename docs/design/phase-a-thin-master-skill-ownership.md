# Phase A — Thin Master Skill Ownership Matrix

This matrix is the migration authority for reducing the master entrypoint. A
row may be relocated or removed only when its canonical owner and observable
regression evidence are recorded.

| Current area | Disposition | Canonical owner | Regression evidence |
|---|---|---|---|
| What Strategist does and explicit invocation | retain | master entrypoint + routing contract | `tests/spec/thin_master_skill_contract_test.go` |
| Bootstrap and preflight | retain summary | `contracts/narrative/01-bootstrap.md`, `contracts/machine/preflight.yaml` | preflight contract tests |
| Parent-agent role lock | retain | master entrypoint + `agent-protocol.md` | pipeline-bypass/evaluation contract tests |
| Provider failure behavior | retain summary | `contracts/machine/errors.yaml`, provider-fallback contract | provider readiness tests |
| Source/runtime path model | retain summary | `agent-protocol.md`, path-hygiene contract | path-hygiene tests |
| Phase loading procedure | relocate | `contracts/index.yaml` and phase contracts | contract-index and phase contract tests |
| Pipeline routing and Critical Hit detail | relocate | `contracts/narrative/00-routing.md`, `11-critical-hit.md` | routing/evaluation contract tests |
| Invocation-mode and local-context mechanics | relocate | `contracts/narrative/06-execution.md`, `agent-protocol.md` | approval and execution-boundary tests |
| Footprint and artifact destinations | retain summary | `agent-protocol.md`, handoff contracts | path-hygiene and handoff tests |
| Dual approval gate | retain summary | approval-gate and execution contracts | approval-boundary tests |
| Response and telemetry mechanics | relocate | response/telemetry contracts | response and telemetry contract tests |
| Drift/self-correction catalog | relocate | identity drift-patterns and preflight contract | preflight identity tests |

## Migration checkpoint and rollback

Before changing the authoring entrypoint, capture the source revision and the
generated runtime checksum. Regenerate `.strategist/SKILL.md` from
`internal/embed/defaults/SKILL.md`, then run the parity and contract gates. If
any retained control or parity check fails, do not publish the reduced
entrypoint; restore the captured source revision and regenerate the runtime
artifact.

This matrix covers the hardened profile only. ORKA/soft-profile packaging is a
separate implementation handoff and is not created or removed by Phase A.
