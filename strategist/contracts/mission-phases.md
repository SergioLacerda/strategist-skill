# Strategist — Mission Phases Contract

## 5. Mission Phases

Pipeline: Ranger (+ opportunity_attack) → Archivist (+ opportunity_attack + side_quest gate) → approval gate → Sniper (+ opportunity_attack)

## Invariant — No Direct Mutation Outside The Pipeline

No mission may bypass Ranger, Archivist, or the approval gate by performing a direct repository mutation.

This applies to:
- code changes
- documentation edits
- config updates
- any other write outside phase-authorized artifact paths

If a direct mutation is attempted before the route has produced its required evidence, Strategist MUST stop and emit `reason=pipeline_bypass_detected` with the missing phase and resume hint.

### Invariant — Opportunity Attack

Opportunity attack runs as a mandatory routine INSIDE each role — Ranger, Archivist, Sniper.
It is not a standalone stage. Each role section (§5.1, §5.2, §7) has an explicit
Opportunity Attack subsection that MUST be executed and emitted.

Required emissions per role:
- Ranger: `[Strategist] phase=ranger opportunity_attack=done items=<N>`
- Archivist: `[Strategist] phase=archivist opportunity_attack=done side_quests=<N>`
- Sniper: `[Strategist] phase=sniper opportunity_attack=done items=0` OR `triggered items=<N>`

This invariant applies even for narrow prompts (single-file/single-target refinement).
"Single-target focus" is NOT a valid reason to skip opportunity attack.

If a role cannot run opportunity attack due to technical error, emit:
`[Strategist] phase=<role> opportunity_attack=failed reason=<why>`
This is non-blocking — log and continue. Do not stop the pipeline.

### 5.0 Quick Draw Execution (conditional)

When §3.1 matched, run:

Ranger (organize only) → Archivist (theme/path/counts) → quick_draw gate → Sniper append

#### 5.0a Ranger (quick_draw)

- Input: original quick note prompt
- Output: one normalized line, preserving context:
  - `idea: <normalized text without expanding scope>`
- Ranger must not add requirements, milestones, or implementation details.

#### 5.0b Archivist (quick_draw)

- Determine theme bucket based on `active.language.chat`:
  - en:    `architecture` | `security` | `analysis` | `general`
  - pt-BR: `arquitetura` | `seguranca` | `analise` | `geral`
- Resolve destination path: `<base_path>/todo/<bucket>.md`
  - en example: `.analysis/todo/architecture.md`
  - pt-BR example: `.analysis/todo/arquitetura.md`
- Inspect existing file content (if present) and compute:
  - `total_ideas`: total idea entries in the destination theme file
  - `similar_ideas`: ideas in the same theme with textual similarity to the normalized idea

#### 5.0c Quick Draw Gate (mandatory)

STOP. Show exactly (using `content_by_lang[active.language.chat].quick_draw_gate` template):

```text
idea: <normalized text>
add idea? (yes / no)
```

Accepted input tokens: `yes` / `no` (English) or reserved tokens `sim` / `nao` (see `internal/i18n/reserved.go`).

Wait for response:
- `yes` / `sim`: proceed to Sniper append.
- `no` / `nao`: return without writing.

Before append, evaluate guarded transition group `finalize_analysis` with effective policy.
Emit canonical event:
`[Strategist] phase=policy_eval status=<allowed|blocked> mission=<id> mode=<mode> can_execute=<bool> transition_group=finalize_analysis`.
If blocked, stop with `reason=policy_blocked`.

#### 5.0d Sniper (quick_draw append)

- Append a new entry to `<base_path>/todo/<bucket>.md`.
- Entry includes timestamp + normalized idea.
- Emit `content_by_lang[active.language.chat].quick_draw_success` with:
  - `{destination_path}`: the bucket file path
  - `{total_ideas}`: total idea count in the file
  - `{similar_ideas}`: ideas with textual similarity to the appended idea

### 5.1 Ranger (discovery slot)

Emit via `persona.content_by_lang[active.language.chat].ranger_start` (substitute `{provider}` with the slot provider skill id).

Invoke the discovery slot provider with:
- User prompt
- `mission_contract.planning_rules`
- Dossier from context enrichment
- Artifact path: `<base_path>/pending/<mission_id>-analysis.md`
- **Role brief — Ranger** (canonical behaviors, always included):
  - `find_unexpected_items`: Surface anything outside the declared mission scope as an addendum
  - `consult_treasure_chests`: Mandatory step — consult all passed chests before generating the artifact. If chest list is empty, proceed.
  - Output format: single transient analysis handoff artifact at the artifact path above
  - Mandatory section in artifact: `Mission Checklist and Phase Roles` with entries for Ranger, Archivist, Sniper using status markers `[x]` (done), `[ ]` (pending), `[-]` (not applicable/no evidence yet)
- **Treasure chests** — mandatory step (chests where scope = `discovery` or `all`):
  Pass filtered list: `[{id}] path={path} — {description}` for each match.
  If no chests match this scope: pass empty list. Ranger skips the consultation step and proceeds without blocking.
  **Chest signal:** Before passing the filtered list to Ranger, for each chest in the list
  emit `persona.content_by_lang[active.language.chat].treasure_chest_found` with `{chest_id}` = chest id and
  `{description}` = chest description. Skip if the list is empty.

The skill decides HOW to use each chest — Strategist only passes the path and description.

Ranger writes the artifact directly (contract: `write_analysis`). Strategist does not
intermediate the write — it only waits for completion and emits the done event.

On success:
Emit via `persona.content_by_lang[active.language.chat].ranger_done` (with `{artifact_path}`).

On failure: emit `[Strategist] phase=ranger status=blocked reason=ranger_failed`, present partial artifact if any.

#### Opportunity Attack (mandatory — runs after artifact is written)

Scan `<base_path>/`:

| Dir      | Check                                                   | Type      |
|----------|---------------------------------------------------------|-----------|
| pending/ | Does this spec have a corresponding plan in refined/?   | file_move |
| refined/ | Does this plan have a corresponding report in archived/? | file_move |
| todo/    | Does this spec have an implementation commit in git?    | file_move |

**Heuristic for file_move:** git log contains a commit referencing the spec slug (date + topic keyword) OR spec lists features that exist as code in the repo. When uncertain, list as a candidate — the user decides.

Also check: treasure_chests not yet consulted for this mission.

Produce opportunity manifest: list of items with `type`, `origin_path`, `destination`, `reason`.

Then:
- Emit: `[Strategist] phase=ranger opportunity_attack=done items=<N>`
- If N > 0: include manifest summary in the analysis artifact AND surface in response
- If N = 0: emit `[Strategist] phase=ranger opportunity_attack=done items=0`, continue to Archivist

Ranger surfaces items only — does not decide strategy for side_quests.
If manifest is non-empty: present to user via `persona.content_by_lang[active.language.chat].opportunity_detected`
with `{count}` = N and `{items_brief}` = one line per item (`→ <slug> reason: <reason>`).
Wait for user response before proceeding to §5.2 (Archivist):
- **yes**: proceed to Archivist (file moves deferred to Sniper after main gate)
- **no**: discard manifest, proceed to Archivist with workspace as-is
- **select**: user picks items by number; defer selected items only

### 5.2 Archivist (refinement slot)

Emit via `persona.content_by_lang[active.language.chat].archivist_start` (with `{provider}`).

Invoke the refinement slot provider with:
- Analysis artifact path
- Side quest report (if present) — injected as context with instruction:
  > "Items listed under 'Items excluded from main analysis' are resolved. Do not treat them as pending. Base your analysis on the post-cleanup workspace state."
- `mission_contract.planning_rules`
- Dossier
- **Role brief — Archivist** (canonical behaviors):
  - `artifact_language`: Use `active.language.docs` for artifact body content
    (proposal.md, design.md, tasks.md). If `language.docs` differs from `language.chat`,
    write artifacts in `language.docs` and all user-facing communication in `language.chat`.
    If both are equal, no distinction is needed.
  - `consult_treasure_chests`: Mandatory step — consult all passed chests before analyzing. If chest list is empty, proceed.
  - Output format: `proposal.md` + `design.md` + `tasks.md` in the artifact subdirectory
- **Treasure chests** — mandatory step (chests where scope = `refinement` or `all`):
  Pass filtered list: `[{id}] path={path} — {description}` for each match. If no chests match: pass empty list; Archivist skips consultation and proceeds.
  **Chest signal:** Before passing the list to Archivist, emit `persona.content_by_lang[active.language.chat].treasure_chest_found`
  for each chest in the list with `{chest_id}` = chest id and `{description}` = chest description. Skip if empty.
- Artifact path: `<base_path>/refined/<mission_id>/` (subdirectory)
  - `analysis.md` — promoted Ranger analysis artifact with mission frontmatter preserved
  - `proposal.md` — what and why (fed by Ranger's analysis artifact)
  - `design.md` — how (architecture, affected components, decisions)
  - `tasks.md` — numbered implementation steps (Sniper's input contract)

**Rules:**
- Archivist NEVER produces a standalone `.md` in `refined/` — always the mission subdirectory
- Archivist promotes the transient Ranger artifact from `pending/` into `analysis.md` inside the mission subdirectory
- If `tasks.md` is empty or absent after Archivist completes, Sniper is not invoked
- Archivist writes all three files directly (contract: `write_analysis`), no gate

#### Opportunity Attack (mandatory — runs during refinement, before approval gate)

Detect side_quests: work adjacent to the declared mission scope that emerged
during discovery or analysis but is out of scope for this mission.

For each side_quest detected:
  - Classify: `backlog` | `split_mission` | `defer`
  - Add to the approval gate presentation under a **Side Quests** section

Emit: `[Strategist] phase=archivist opportunity_attack=done side_quests=<N>`

Side quest strategy is Archivist's decision. Sniper never decides side quest strategy.
If N = 0: emit with `side_quests=0`, proceed to approval gate.

Archivist writes artifacts directly (contract: `write_analysis`). Strategist does not
intermediate the write — it only waits for completion and emits the done event.

On success:
Emit via `persona.content_by_lang[active.language.chat].archivist_done` (with `{artifact_path}`).
