# Proposal: Engineer OpenSpec Output + Conditional Gate
**Date:** 2026-05-28
**Status:** refined — awaiting execution
**Source:** `.analysis/pending/2026-05-28-engineer-openspec-output-design.md`

---

## What

Change Engineer's output contract from a single flat `<mission_id>-plan.md` file to an
OpenSpec subdirectory (`proposal.md` + `design.md` + `tasks.md`). Simultaneously make
the approval gate conditional on `tasks.md` content, and add a drift pattern that
prevents routing plan-creation work to Hunter.

## Why

Three behavioral problems emerged during live missions:

1. **Wrong output format.** Engineer writes `refined/<mission_id>-plan.md` — a flat file.
   The existing skill set (openspec-explore) produces structured OpenSpec subdirectories.
   The contract and the tool are misaligned.

2. **Gate fires for analytical missions.** Section 6 evaluates the plan inline and fires
   the gate even when no code changes are planned. A mission that only creates docs inside
   `<base_path>/` should complete silently as `plan_only`.

3. **Plan creation leaks to Hunter.** Without an explicit drift pattern, Strategist can
   be coaxed into routing "write a spec" work to the execution slot. Hunter's contract is
   `controlled` (code, git, config) — document authoring belongs to Engineer.

## Scope

Two files modified. No new files. No behavioral change to Scout, housekeeping_scan,
side quest pipeline, learning phase, or install/release scripts.

| File | Sections touched |
|------|-----------------|
| `strategist/SKILL.md` | §5e (Engineer output), §6 (Approval Gate), Drift Self-Correction |
| `strategist/skill.yaml` | `pipeline.refinement` stage, `forbidden_behaviors` |

§4 (wizard context hints) was already delivered in commit `61ebb8e` — excluded from scope.
