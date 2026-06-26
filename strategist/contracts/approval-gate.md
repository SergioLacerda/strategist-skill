# Strategist — Approval Gate Contract

## 6. Approval Gate (MANDATORY)

After Archivist completes, evaluate the refined analysis before presenting the gate:

Read `<base_path>/refined/<mission_id>/tasks.md` before deciding:

**If `tasks.md` is empty or absent:**
  emit `[Strategist] phase=approval_gate status=analysis_delivered`, return mission result
  with `status: analysis_delivered`. Do NOT present the gate — the mission is complete.

**If `tasks.md` contains documentation targets only inside `<base_path>/`:**
  present the gate once with the full analysis visible.

**If `tasks.md` contains documentation targets outside `<base_path>/` (e.g. provider bootstrap files, .claude/, .codex/):**
  present the gate with an explicit outside-base-path declaration.

In all cases where the gate is presented: STOP. Do not invoke Sniper without explicit user review acceptance.

Any attempt to mutate repository files directly before this gate completes with acceptance is a hard failure:
- emit `reason=pipeline_bypass_detected`
- include `expected_phase=approval_gate`
- include the missing evidence (`approval_gate:analysis_accepted`)
- stop immediately and provide a short resume hint

Emit via `persona.content_by_lang[active.language.chat].review_prompt` with:
- `{artifact_path}`
- `{mission_tasks_summary}` (checklist/todo summary from `tasks.md`)
- `{documentation_targets_list}` (outside-`<base_path>` targets, `none` when empty)
- `{side_quests_list}` (`none` when empty)

Wait for response:
- **aceitar / accepted / agree / sim / concordo**: re-emit checkpoint with step_3_icon=✅, step_4_icon=⏳. Proceed to Sniper.
- **missing_item / revision / esqueci / faltou**: emit `[Strategist] phase=approval_gate status=revision_requested`,
  return control to Archivist with the user's missing item noted.
- **reject / nao / cancelar**: emit `[Strategist] phase=approval_gate status=rejected`,
  return mission result with `status: rejected`, artifact paths for analysis and refined package.
- **review**: present the refined analysis content, then re-ask.

Invoking Sniper without receiving explicit review acceptance is a **forbidden behavior**.

---

## 7. Sniper (documentation materialization slot)

Emit via `persona.content_by_lang[active.language.chat].sniper_start`.

#### Opportunity Attack (mandatory — runs during documentation materialization)

If an out-of-scope write or non-documentation target surfaces mid-materialization:
  - STOP immediately
  - Emit: `[Strategist] phase=sniper opportunity_attack=triggered items=<N>`
  - Report items to user: type, description, reason
  - Do NOT continue or decide strategy
  - Resume only after Archivist reviews and user accepts

If no out-of-scope item emerges: proceed normally.
Emit: `[Strategist] phase=sniper opportunity_attack=done items=0`

### 7a. Documentation Task List

Before invoking the slot, read `<base_path>/refined/<mission_id>/tasks.md` and parse the numbered documentation tasks.

Emit via `persona.content_by_lang[active.language.chat].execution_tasks_header` with `{total}` = number of tasks.

Then emit each task via `persona.content_by_lang[active.language.chat].execution_task_line` with:
- `{status_icon}` = `⬜` (all pending initially)
- `{index}` = task number (e.g. `1`, `2`)
- `{task_title}` = first line of the task description

Instruct the documentation slot to emit per-task progress events as it works:
`[Strategist] phase=execution task=<N> status=running|done`

On receiving `task=<N> status=running`: re-emit the full task list with task N marked `⏳` and all prior tasks `✅`.
On receiving `task=<N> status=done`: re-emit the full task list with task N marked `✅` and task N+1 marked `⏳` (if not last).
On all tasks done: re-emit with all tasks `✅`.

Invoke the documentation materialization slot provider with:
- Refined analysis artifact path
- `mission_contract.planning_rules`
- **Role brief — Sniper** (canonical behaviors):
  - `review_accepted`: approval gate acceptance was received at §6 — proceed
  - `consult_treasure_chests`: Mandatory step — consult all passed chests before acting. If chest list is empty, proceed.
- **Treasure chests** — mandatory step (chests where scope = `execution` or `all`):
  Pass filtered list: `[{id}] path={path}` for each match. If no chests match: pass empty list; Sniper skips consultation and proceeds. (omit if none)
  **Chest signal:** Before passing the list to Sniper, emit `persona.content_by_lang[active.language.chat].treasure_chest_found`
  for each chest in the list with `{chest_id}` = chest id and `{description}` = chest description. Skip if empty.

Documentation report artifact path: `<base_path>/archived/<mission_id>-report.md`

Wait for completion. On success:
Emit via `persona.content_by_lang[active.language.chat].sniper_done` (with `{artifact_path}`).
