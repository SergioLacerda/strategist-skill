# Strategist Provider Discovery Contract

This document is the canonical reference for provider integration with the Strategist runtime.
It is read by provider bootstrap adapters, not by Strategist itself.

## Runtime Authority

`.strategist/` is the only valid Strategist runtime source. Every provider integration
must resolve this directory before activating Strategist behavior.

## Mandatory Runtime Files

When a local Strategist runtime is present, a provider MUST load:

1. `.strategist/SKILL.md` — agent instructions and path model
2. `.strategist/skill.yaml` — pipeline definition and slot/contract mapping

If either file is absent, the provider MUST stop and emit the documented absence condition.
It MUST NOT improvise a substitute runtime from any other source.

## Resolution Order

1. Provider loads its own minimal local bootstrap (e.g. `.codex/commands.md`, `.claude/claude-instructions.md`).
2. Bootstrap checks whether `.strategist/` exists locally.
3. If present, provider loads `.strategist/SKILL.md`.
4. Provider loads `.strategist/skill.yaml`.
5. Provider applies governance context from `.sdd/` as already expected by its bootstrap.

## Governance Relationship

`.sdd/` is the governance authority. It may constrain or enrich execution context,
but it is NOT the operational runtime source for Strategist.

- Load `.sdd/agent-instructions.md` for governance bootstrap.
- Load `.strategist/SKILL.md` and `.strategist/skill.yaml` for Strategist runtime.
- These two loads are separate concerns and must not be conflated.

## Forbidden Behaviors

Providers MUST NOT:

- Treat `.sdd/` as a substitute for `.strategist/` when activating Strategist.
- Load files from the source tree `strategist/` (without the leading dot) during runtime.
- Silently activate a global Strategist equivalent when `.strategist/` is absent.
- Re-explain the full Strategist runtime contract in provider-local bootstrap files.

## Failure Behavior

When `.strategist/` is absent or its mandatory files are missing:

- Emit: `error=not_installed` or the equivalent provider-local absence signal.
- Do not fall back to `strategist/` (source tree) or any other Strategist substitute.
- Do not activate governance-only behavior as if that were Strategist runtime.

## What Provider Bootstrap Files Must Not Do

- Duplicate the full Strategist pipeline definition.
- Override slot behavior or contract semantics.
- Load runtime files from paths other than `.strategist/`.
- Treat absence of `.strategist/` as a soft warning — it must be a hard stop.
