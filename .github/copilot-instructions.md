# GitHub Copilot Governance Bootstrap

You are operating in a workspace governed by **Spec Driven Development (SDD)**.

## Critical Instruction

Read and adhere to the canonical governance rules in:
```
.sdd/agent-instructions.md
```

This file is the **single source of truth** for all governance policies in this workspace.

## Commands And Skills (Source Of Truth)

1. Commands source of truth: `.sdd/commands`.
2. Skills source of truth: `.sdd/skills`.
3. On startup, load:
   - `.sdd/commands/registry.json`
   - `.sdd/skills/registry.json`
4. For each active command/skill in registries, read canonical files:
   - Commands: `.sdd/commands/<command-id>/command.yaml`
   - Skills: `.sdd/skills/<skill-name>/skill.yaml`
5. Precedence rule:
   - Local path (`.github/prompts/*`) is for context and ergonomics.
   - `.sdd` is authoritative for routing/policy and wins conflicts.

## Quick Reference

- **Mandate enforcement**: Non-negotiable rules (M001-M010, M015)
- **Governance status**: Run `sdd runtime status` to check workspace health
- **Validation**: Run `sdd governance validate` before finalizing changes
- **Activation**: Governance activates automatically on project load via `.sdd/seedlings/`

## Governance Documentation

All governance documentation lives in `.sdd/source/`:
- `mandates/mandates.md` — Mandate descriptions and enforcement rules
- `guidelines/` — Customizable guidelines by category (if any)
- `README.md` — Onboarding guide for agents

## Operating Rules

- Do not bypass mandatory mandates.
- Prefer generated templates and `.sdd/*` canonical governance over improvised structure.
- When the workspace state is unclear, run `sdd runtime status` first.

## Expected Validation Commands

```bash
sdd governance validate
sdd runtime status
```

## Notes

This bootstrap is intentionally a redirector. Do not rely on framework-external paths.

## Safe Fallback

If registries or canonical files are missing/inconsistent, register bootstrap drift and continue in safe fallback mode without inventing missing rules.
