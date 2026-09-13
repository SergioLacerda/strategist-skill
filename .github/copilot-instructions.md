# GitHub Copilot Governance Bootstrap

You are operating in a workspace governed by **Spec Driven Development (SDD)**.

## Critical Instruction

Read and adhere to the canonical governance rules in:
```
.providence/agent-instructions.md
```

This file is the **single source of truth** for all governance policies in this workspace.

## Commands And Skills (Source Of Truth)

1. Commands source of truth: `.providence/commands`.
2. Skills source of truth: `.providence/skills`.
3. On startup, load:
   - `.providence/commands/registry.json`
   - `.providence/skills/registry.json`
4. For each active command/skill in registries, read canonical files:
   - Commands: `.providence/commands/<command-id>/command.yaml`
   - Skills: `.providence/skills/<skill-name>/skill.yaml`
5. Precedence rule:
   - Local path (`.github/prompts/*`) is for context and ergonomics.
   - `.providence` is authoritative for routing/policy and wins conflicts.

## Quick Reference

- **Mandate enforcement**: Non-negotiable rules (M001-M010, M015)
- **Governance status**: Run `providence runtime status` to check workspace health
- **Validation**: Run `providence governance validate` before finalizing changes
- **Activation**: Governance activates automatically on project load via `.providence/seedlings/`

## Governance Documentation

All governance documentation lives in `.providence/source/`:
- `mandates/mandates.md`  Mandate descriptions and enforcement rules
- `guidelines/`  Customizable guidelines by category (if any)
- `README.md`  Onboarding guide for agents

## Operating Rules

- Do not bypass mandatory mandates.
- Prefer generated templates and `.providence/*` canonical governance over improvised structure.
- When the workspace state is unclear, run `providence runtime status` first.

## Expected Validation Commands

```bash
providence governance validate
providence runtime status
```

## Notes

This bootstrap is intentionally a redirector. Do not rely on framework-external paths.

## Safe Fallback

If registries or canonical files are missing/inconsistent, register bootstrap drift and continue in safe fallback mode without inventing missing rules.
