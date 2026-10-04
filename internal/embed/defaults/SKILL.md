# Strategist — Agent Instructions

Strategist is an analysis and documentation orchestrator. It coordinates the
fixed pipeline through the configured discovery, refinement, approval, and
execution contracts. It does not mutate source code, tests, hooks, locks, or
other repository implementation artifacts during analysis or refinement.

## Explicit Invocation Boundary

Use this skill only after the user explicitly invokes Strategist through this
dedicated skill, a registered host slash command, or a `strategist mission` CLI
operation. The presence of `.strategist/` and a `strategist check --json` result
make Strategist available for inspection, but do not invoke it, start a mission,
or change how an ordinary direct request is handled.

A reference to a path under `.analysis/refined/` does not activate a Strategist
mission. A local execution context is only a precondition; it is not proof of
mission activation or approval. The parent-agent role lock begins only after an
explicit Strategist invocation transitions the request into an active mission.

## ENTRYPOINT — after explicit invocation

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

The parent agent MUST resolve the route through Strategist and relay work
through the Ranger → Archivist → Approval Gate → Sniper pipeline. For a Ranked
binding, it MUST NOT load a same-named skill from `.strategist/skills/`, a global
skill directory, `external-skills-source`, or a host loader. It MUST create the
mission state with `strategist mission start --mission-id <id>`, then submit `bootstrap_done`,
record the Scout route, and then submit `intake_done`. For a Ranked binding with
`runtime.kind: embedded`, it obtains the resolved slot envelope with `strategist mission invoke --mission-id <id> --role <role> --slot <slot> --json`, executes the exact `payload` once as the current-host adapter under `input.execution_contract` and `input.output_contract`, and returns exactly one raw completion object to `strategist mission complete --request-id <id> --json`. This bounded adapter execution is the configured Weapon invocation; the parent MUST NOT add its own workflow, provider, or conclusions. A Ranked `runtime.kind: openspec_root` instead runs
its compiled private OpenSpec runtime and publishes the completed change through
`strategist mission normalize-openspec`; `mission invoke` is not its executor.
`--host codex|claude --context "<original user request>"` is a standalone-shell
convenience for an operator whose child process has working network and auth. A
managed Codex or Claude session MUST NOT recursively spawn the same host; it uses
the current-host adapter above.
The parent agent MUST
NOT perform discovery, refinement, or execution directly except for that scoped
host-adapter execution of the emitted payload. It MUST NOT perform
Scout's route classification or skip Scout. It MUST NOT replace a missing
provider with its own built-in capabilities, or treat preflight as source
mutation authorization.

Discovery subtypes are selected by Scout and executed under the fixed Ranger role.
Ranger must invoke the configured discovery Weapon and normalize its untrusted
result. There is no fallback: an unavailable or incompatible Weapon is a
role-invocation failure.

## Canonical Taxonomy Vocabulary

Use these seven public families consistently: Roles, Weapons, Feats, Tools,
Mechanisms, Stages, and Artifacts. Roles are agent personas that own
responsibilities; Weapons are external skill packages employed by pluggable
Roles; Feats are contextual, judgment-based behavior; Tools perform operations;
Mechanisms enforce deterministic rules; Stages frame governed workflows; and
Artifacts preserve generated results. Routes are selection outcomes, not a family.
An item with both a judgment part and a deterministic part is one item with two
facets (PRECISE-SHOT, Opportunity Attack).

`LEVELING` is an immutable operational resolver consumed by the INITIATIVE Feat;
it is a Tool. LEVELING is not a Role, Weapon, provider, or execution authority.
`INITIATIVE` is a Feat, not a Mechanism. Its contextual advice cannot mutate the
LEVELING resolution, provider, gate, or implementation authorization.
`origin` and `extensibility` are independent Role properties; “internal role” and
“external role” are historical compatibility wording only. Pathfinder,
Cartographer, Jeweler, and Jewelcrafter remain inactive proposals.

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
- `.strategist/skills/<provider_id>/` is a generated compatibility/provenance
  projection; it is not the runtime source for Ranked Embedded invocation.
- Ranked Embedded runtime reads the compiled catalog and payload through
  `mission invoke`; only an explicitly typed Custom binding may use its
  recorded external connector or host loader.
- Ranked `openspec_root` runtime uses the compiled private runtime rooted at
  `.strategist/openspec`; its provider output is published only through
  `mission normalize-openspec`.
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
