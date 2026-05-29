# Strategist Skill Registration — Design Spec
**Date:** 2026-05-28  
**Status:** pending implementation  
**Topic:** Multi-agent slash command registration via self-contained `.strategist/` install

---

## Problem Statement

After running `install.sh`, the `/strategist` slash command does not appear in any agent terminal. The install script currently:
- Generates `active.yaml` in the skill source root
- Scaffolds workspace dirs (`<base_path>/todo`, `pending`, `refined`, `done`)

It does NOT register the skill with any agent runtime, so no agent knows it exists.

---

## Goal

After `install.sh` completes, `/strategist` appears as a slash command in all four supported agent runtimes (Claude Code, Gemini CLI, Gemini Antigravity, Codex CLI), pointing to a self-contained, per-project skill installation.

---

## Architecture

### Two-artifact install

`install.sh` produces two artifacts:

**1. `.strategist/` in the target project** — self-contained skill runtime  
All configuration and instructions live here. This is the canonical runtime for this installation.

```
<target-project>/
└── .strategist/
    ├── SKILL.md              ← full skill instructions (copied from source)
    ├── active.yaml           ← generated config (wizard or silent)
    ├── personas/             ← copied from skill source
    ├── roles/                ← copied from skill source (wizard may patch default.yaml)
    ├── schemas/              ← copied from skill source
    ├── knowledge.index.yaml  ← copied from skill source
    ├── memory/               ← empty, runtime writes here
    ├── index.yaml            ← internal domain index
    └── identity/             ← internal domain files (drift-patterns, etc.)
```

**2. Agent shims** — thin pointer files in each agent's global skills dir

Each shim is a `SKILL.md` file that declares `skill_root` and instructs the agent to load the full instructions from there. The agent reads the shim on startup and resolves all relative paths from `skill_root`.

---

## Agent Shim Registration

For each supported agent runtime, `install.sh` creates:

```
~/.claude/skills/strategist/SKILL.md
~/.gemini/skills/strategist/SKILL.md
~/.gemini/antigravity/skills/strategist/SKILL.md
~/.codex/skills/strategist/SKILL.md
```

**Shim format:**
```markdown
---
name: strategist
description: "Multi-phase mission orchestrator. Coordinates discovery, refinement, and execution through three pluggable slots."
skill_root: /absolute/path/to/target/.strategist
---

# Strategist

**SKILL_ROOT:** `/absolute/path/to/target/.strategist`

Read full instructions from: `{skill_root}/SKILL.md`

All config paths (active.yaml, personas/, roles/, schemas/) resolve from skill_root.
```

**Safety rule:** A shim is only written if the parent agent directory exists. If `~/.codex/` does not exist, the Codex shim is skipped with a notice (not an error).

**Idempotency:** Each install overwrites existing shims without prompting. Shims are generated artifacts, never manually edited.

---

## Changes to `install.sh`

### New: `copy_skill_runtime()` function

Copies source files from `$SKILL_ROOT` into `<target>/.strategist/`:
- `SKILL.md`, `knowledge.index.yaml`
- Directories: `personas/`, `roles/`, `schemas/`, `templates/domain/` (→ `identity/` and domain files)
- Creates empty `memory/` dir
- Does NOT copy `install.sh`, `skill.yaml`, `skills/` (sub-skills), or `active.yaml` (generated separately)

### New: `install_agent_shims()` function

Iterates over known agent shim targets:
```bash
SHIM_TARGETS=(
  "${HOME}/.claude/skills"
  "${HOME}/.gemini/skills"
  "${HOME}/.gemini/antigravity/skills"
  "${HOME}/.codex/skills"
)
```
For each: if parent exists → create `<target>/strategist/` → write shim SKILL.md.

### Updated: `write_active_yaml()`

Writes `active.yaml` to `<target>/.strategist/active.yaml` instead of `$SKILL_ROOT/active.yaml`.

### Updated: `scaffold_workspace()`

Remains unchanged — scaffolds `<base_path>/todo`, `pending`, `refined`, `done`.

### Updated: main flow

```
run_silent() / run_wizard()
  → copy_skill_runtime()     ← new
  → write_active_yaml()      ← updated path
  → scaffold_workspace()     ← unchanged
  → install_agent_shims()    ← new
```

---

## Changes to `SKILL.md` (source)

Add to the top of the **Bootstrap** section:

> **Skill root resolution:** If invoked from an agent shim, `skill_root` is declared in the frontmatter. Resolve all relative paths — `active.yaml`, `personas/`, `roles/`, `schemas/` — from `skill_root`. If `skill_root` is not present, assume the directory containing this file is the skill root.

This makes the skill self-documenting when read via the shim.

---

## What Does NOT Change

- `install.sh` interface (`--wizard`, `--target`, `--mode` flags)
- `skill.yaml` (SDD schema descriptor, stays in skill source root)
- Wizard TUI questions and slot configuration
- Mission artifact paths (`<base_path>/pending`, `refined`, `done`)
- Sub-skills in `strategist/skills/` (engineer, etc.)
- The `--target` flag behavior: target repo root is still where workspace is scaffolded; `.strategist/` is placed at `<target-repo-root>/.strategist/`

---

## Edge Cases

| Case | Behavior |
|------|----------|
| `.strategist/` already exists | Overwrite skill runtime files; preserve `memory/` contents |
| Agent dir does not exist | Skip shim silently, print notice |
| `--target` flag used | `.strategist/` goes into `--target` dir, shim `skill_root` points there |
| Skill source moved after install | Shims still work (point to `.strategist/`, not skill source) |
| Multiple projects installing strategist | Each gets its own `.strategist/`; shims point to last install |

**Note on multiple projects:** Since shims are global and point to a single `skill_root`, installing in project B overwrites the shim's `skill_root` from project A. This is a known limitation. Future work: per-project shim management or shell-based resolution.

---

## Out of Scope

- Uninstall command (`rm -rf ~/.claude/skills/strategist` for manual cleanup)
- Multiple simultaneous project installations with agent-level switching
- SDD registry integration (skill is standalone)
