# Design: Engineer OpenSpec Output + Conditional Gate
**Date:** 2026-05-28

---

## Change 1 — SKILL.md §5e: Engineer output becomes OpenSpec subdirectory

**File:** `strategist/SKILL.md`, around line 225

**Replace:**
```
- Primary artifact path: `<base_path>/refined/<mission_id>-plan.md`
- Secondary artifact scope: `<base_path>/` (Engineer may create additional `.md` summaries here)
```

**With:**
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

## Change 2 — SKILL.md §6: Gate reads tasks.md, not plan content

**File:** `strategist/SKILL.md`, around line 238

**Replace** the three static cases:
```
**If the plan requires no Hunter execution** (purely analytical mission, no writes outside
`<base_path>/`): emit `[Strategist] phase=approval_gate status=plan_only`, return mission
result with `status: plan_only`. Do NOT present the gate — the mission is complete.

**If the plan requires Hunter to write only inside `<base_path>/`** (e.g., moving files
to `done/`, creating a report): present the gate once with the full plan visible.

**If the plan requires Hunter to write outside `<base_path>/`** (code, git, config, system):
present the gate with an explicit external-scope warning.
```

**With:**
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

Also update the artifact reference in the approval_prompt template line from
`{artifact_path}` (single file) to `{artifact_dir}` (subdirectory) and point it at
`<base_path>/refined/<mission_id>/tasks.md` for the gate display.

---

## Change 3 — SKILL.md Drift: new pattern route_plan_creation_to_hunter

**File:** `strategist/SKILL.md`, Drift Self-Correction section (last line, ~line 339)

**Append** after the last existing pattern:
```
- `route_plan_creation_to_hunter`: You are about to ask Hunter to create a document,
  spec, analysis, or implementation plan. → Stop. Document authoring is Engineer's work
  (contract: `write_analysis`). Return to phase 5e and invoke the refinement slot.
```

---

## Change 4 — skill.yaml: refinement stage artifact_path becomes directory

**File:** `strategist/skill.yaml`, pipeline stage `refinement` (~line 74–77)

**Replace:**
```yaml
  - stage: refinement
    slot: refinement
    input: [discovery_artifact, side_quest_report]
    artifact_path: "<base_path>/refined/<mission_id>-plan.md"
```

**With:**
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

## Change 5 — skill.yaml: add delegate_analysis_creation_to_hunter to forbidden_behaviors

**File:** `strategist/skill.yaml`, `forbidden_behaviors` block (~line 110–122)

**Append** after the last entry:
```yaml
  - delegate_analysis_creation_to_hunter
```

---

## What does NOT change

- Slot contracts (`write_pending`, `write_analysis`, `controlled`)
- housekeeping_scan logic and side quest pipeline
- Scout's output format (`pending/<mission_id>-discovery.md`)
- Bootstrap scripts (`install.sh`, `bootstrap.sh`)
- Release workflow
- Learning phase
- Mission result schema
