# Tasks: Engineer OpenSpec Output + Conditional Gate
**Date:** 2026-05-28
**Scope:** external — writes to strategist/ outside base_path

---

## T1 — SKILL.md §5e: replace flat plan path with OpenSpec subdirectory

**File:** `strategist/SKILL.md`

Find and replace the two lines:
```
- Primary artifact path: `<base_path>/refined/<mission_id>-plan.md`
- Secondary artifact scope: `<base_path>/` (Engineer may create additional `.md` summaries here)
```

Replace with:
```
- Artifact path: `<base_path>/refined/<mission_id>/` (subdirectory)
  - `proposal.md` — what and why (fed by Scout's discovery artifact)
  - `design.md` — how (architecture, affected components, decisions)
  - `tasks.md` — numbered implementation steps (Hunter's input contract)

**Rules:**
- Engineer NEVER produces a standalone `.md` in `refined/` — always the three-file subdirectory
- If `tasks.md` is empty or absent after Engineer completes, Hunter is not invoked
- Engineer writes all three files directly (contract: `write_analysis`), no gate
```

---

## T2 — SKILL.md §6: gate reads tasks.md instead of evaluating plan inline

**File:** `strategist/SKILL.md`

Find and replace the three static case blocks starting with:
```
**If the plan requires no Hunter execution**
```
through:
```
present the gate with an explicit external-scope warning.
```

Replace with:
```
Read `<base_path>/refined/<mission_id>/tasks.md` before deciding:

**If `tasks.md` is empty or absent:**
  emit `[Strategist] phase=approval_gate status=plan_only`, return mission result
  with `status: plan_only`. Do NOT present the gate — the mission is complete.

**If `tasks.md` contains tasks scoped only to `<base_path>/`:**
  present the gate once with the full plan visible.

**If `tasks.md` contains tasks that write outside `<base_path>/` (code, git, config, system):**
  present the gate with an explicit external-scope warning.
```

---

## T3 — SKILL.md Drift: append route_plan_creation_to_hunter

**File:** `strategist/SKILL.md`

Find the last line of the Drift Self-Correction section:
```
- `housekeeping_scan_as_slot`: You are about to delegate the housekeeping scan to Scout or another slot. → Stop. Execute the scan directly as Strategist (deterministic, internal phase).
```

Append immediately after:
```
- `route_plan_creation_to_hunter`: You are about to ask Hunter to create a document, spec, analysis, or implementation plan. → Stop. Document authoring is Engineer's work (contract: `write_analysis`). Return to phase 5e and invoke the refinement slot.
```

---

## T4 — skill.yaml: refinement stage artifact_path becomes directory with produces

**File:** `strategist/skill.yaml`

Find:
```yaml
  - stage: refinement
    slot: refinement
    input: [discovery_artifact, side_quest_report]
    artifact_path: "<base_path>/refined/<mission_id>-plan.md"
```

Replace with:
```yaml
  - stage: refinement
    slot: refinement
    input: [discovery_artifact, side_quest_report]
    artifact_path: "<base_path>/refined/<mission_id>/"
    produces:
      - "<base_path>/refined/<mission_id>/proposal.md"
      - "<base_path>/refined/<mission_id>/design.md"
      - "<base_path>/refined/<mission_id>/tasks.md"
```

---

## T5 — skill.yaml: add delegate_analysis_creation_to_hunter to forbidden_behaviors

**File:** `strategist/skill.yaml`

Find the last entry in `forbidden_behaviors`:
```yaml
  - engineer_writes_non_md
```

Append after it:
```yaml
  - delegate_analysis_creation_to_hunter
```

---

## Verification

After all edits:

1. `grep "plan.md" strategist/SKILL.md strategist/skill.yaml` → zero results in §5e and pipeline.refinement
2. `grep "tasks.md" strategist/SKILL.md` → present in §6 gate logic
3. `grep "route_plan_creation_to_hunter" strategist/SKILL.md` → present in Drift section
4. `grep "delegate_analysis_creation_to_hunter" strategist/skill.yaml` → present in forbidden_behaviors
5. `grep "produces:" strategist/skill.yaml` → present under refinement stage
