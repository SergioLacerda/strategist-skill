# Proposal: Role Standardization — Ranger / Archivist / Sniper
**Date:** 2026-05-28
**Status:** refined — awaiting execution
**Source:** `.analysis/pending/2026-05-28-role-standardization-discovery.md`

---

## What

Establish a clean two-layer naming system for Strategist's slots:

| Layer | Names | Where used |
|-------|-------|-----------|
| **Internal** (Strategist identity) | Ranger · Archivist · Sniper | SKILL.md prose, personas, install.sh, templates |
| **External** (slot keys, event labels) | discovery · refinement · execution | roles YAML, skill.yaml, progress events |

Simultaneously fix a latent bug in install.sh where the wizard writes
`scout:`/`engineer:`/`hunter:` as role keys instead of the expected
`discovery:`/`refinement:`/`execution:`.

## Why

Three layers of naming coexist today with no canonical contract:
1. `roles/*.yaml` slot keys: `discovery / refinement / execution` (called "internal" in a comment)
2. SKILL.md prose and epic persona: `Scout / Engineer / Hunter` (treated as canonical)
3. pragmatic persona phase labels: `analysis / refinement / execution` (matches slot keys)

No file defines which layer is authoritative or how they relate. The personas
use different conventions. The wizard generates structurally invalid roles files.

The rename replaces the legacy internal names (Scout/Engineer/Hunter) with
Ranger/Archivist/Sniper across all prose and labels, while preserving the external
slot key vocabulary unchanged.

## Scope

| File | Change type |
|------|------------|
| `strategist/SKILL.md` | Rename ~31 occurrences Scout→Ranger, Engineer→Archivist, Hunter→Sniper |
| `strategist/skill.yaml` | Update forbidden_behavior names + slot description prose |
| `strategist/personas/epic.yaml` | phase_labels + prose |
| `strategist/personas/pragmatic.yaml` | approval_prompt prose (1 line) |
| `strategist/install.sh` | Variable rename + **bug fix** in roles YAML write |
| `strategist/skills/engineer/` | Full directory rename → `skills/archivist/` + id update |
| `strategist/roles/default.yaml` | `refinement: engineer` → `refinement: archivist` |
| `strategist/schemas/progress-contract.yaml` | Examples + note |
| `strategist/templates/domain/identity/what-i-am.yaml` | Prose (3 lines) |

`protocol.md` — no changes needed (already uses external names throughout).
`roles/mission.yaml` + `roles/spec-driven.yaml` — no changes (use `_injected_by_sdd`, not `engineer`).

## Ordering constraint

This changeset overlaps with `.analysis/refined/2026-05-28-engineer-openspec-output/`
on SKILL.md §5e and §6. Apply **this rename first**, then apply the openspec-output
changes, to avoid merge conflicts.
