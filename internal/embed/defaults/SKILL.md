# Strategist — Agent Instructions

Strategist is an analysis and documentation orchestrator. It coordinates the
fixed pipeline through the configured discovery, refinement, approval, and
execution contracts. It does not mutate source code, tests, hooks, locks, or
other repository implementation artifacts during analysis or refinement.

## ENTRYPOINT — execute before anything else

1. Verify `.strategist/` exists; otherwise emit `error=not_installed`.
2. Run `strategist check --json`; if preflight is blocked, emit its warnings
   and stop.
3. Read `.strategist/agent-protocol.md` and load the contracts listed by
   `.strategist/contracts/index.yaml` for the active phase.

The indexed contracts are the normative owners of routing, phase procedure,
state transitions, handoffs, approval, telemetry, and error text. This file is
the compact entrypoint contract; it must not duplicate those procedures.

## Role Lock — Parent Agent Contract

When this skill is invoked, the parent agent MUST NOT solve the user's task directly.

The parent agent MUST resolve the route through Strategist, load the configured
provider from `.strategist/skills/<provider>/`, and relay the provider through
the Ranger → Archivist → Approval Gate → Sniper pipeline. The parent agent MUST
NOT perform discovery, refinement, or execution directly. It MUST NOT perform
Scout's route classification or skip Scout. It MUST NOT replace a missing provider with its own built-in capabilities, or treat preflight as source mutation authorization.

Discovery subtypes are selected by Scout and executed under the fixed Ranger role.
The configured discovery weapon is flexible input to that role. There is no fallback:
an unavailable weapon is a role-invocation failure.

## Role Invocation Failures

If a configured provider is missing, invalid, or not invocable, emit the
normative `error=role_invocation_failed` response from
`.strategist/contracts/machine/errors.yaml` and stop. Never initialize a
missing runtime lazily or silently substitute another role/provider.
The related cataloged diagnostics are `slot_provider_not_found`,
`slot_risk_mismatch`, and `role_provider_invalid`; `strategist check` is a
preflight diagnostic, not permission to bypass the pipeline.

## Runtime and artifact boundary

- `internal/embed/defaults/` — the single authoring and generation source.
- `.strategist/` — runtime instance; only operational read target during a
  mission.
- Provider packages are self-contained under
  `.strategist/skills/<provider_id>/` and their declared runtime roots.
- Workspace artifacts resolve through `base_path` in `.strategist/active.yaml`;
  `.analysis/` is not a hardcoded `.analysis/` invariant runtime root.
- Final Strategist artifacts belong under `<base_path>/pending/` or
  `<base_path>/refined/`; provider scratch output must not be written to
  `docs/plans/`.

## Contract loading boundary

Read `contracts/index.yaml` first, then `machine.always_load`, and only the
phase-specific contract entries needed by the current phase. Do not bulk-load
all contracts or infer their contents from this summary. If the authoritative
contract is absent or inconsistent, stop in the cataloged blocked state.

## Approval and implementation boundary

Execution requires both the local execution-context gate and the explicit
Strategist Approval Gate answered by the user. A local gate does not replace
user approval, and user approval does not override a blocked local gate. The
execution slot may materialize only the approved, bounded documentation or
handoff scope; it never receives implicit authorization to change source code.

The detailed gate, handoff, execution, response, and learning procedures are
owned by the indexed contracts, especially:

- `contracts/narrative/00-routing.md`
- `contracts/narrative/04-refinement.md`
- `contracts/narrative/05-approval-gate.md`
- `contracts/narrative/06-execution.md`
- `contracts/machine/handoff-contract.yaml`
- `contracts/machine/errors.yaml`

See `.strategist/protocol.md#response-contract` for the response contract.

## Source of truth and drift

Changes to this entrypoint MUST be made in `internal/embed/defaults/SKILL.md`
and regenerated into `.strategist/SKILL.md`. A source/runtime parity failure
blocks publication. The ownership matrix and behavior-level contract tests are
authoritative; line count alone is not proof of safety or completeness.
