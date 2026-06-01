# Token Economy Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the Token Economy design across four phases: mode inference in intake, treasure chest governance, role contracts with Opportunity Attack as a cross-cutting routine, and compression pipeline contracts.

**Architecture:** Most changes are YAML contract files — `intake.schema.yaml`, `prompt-intake/skill.yaml`, role files, `skill.yaml`, and new schema/manifest files. Phase 4 evolves the `context-enrichment` and `dossier-builder` skill contracts. No new Go binaries in phases 1–3.

**Tech Stack:** YAML (contracts/schemas), Go (existing test harness in `strategist/tests/`), Gherkin features, `gopkg.in/yaml.v3`

**Spec:** `.analysis/refined/token_economy/2026-06-01-token-economy-design.md`

---

## Phase 1 — Token Strategy + Economic Intake

### Task 1: Add `token_strategy` field to `intake.schema.yaml`

**Files:**
- Modify: `strategist/schemas/intake.schema.yaml`

**Step 1: Read the current file**

```bash
cat strategist/schemas/intake.schema.yaml
```

**Step 2: Add the `token_strategy` field block after the existing `urgency` field**

Add at the end of `fields:`:

```yaml
  token_strategy:
    description: >
      Execution mode inferred by prompt-intake from task_type and uncertainty signals.
      Never set by the user. Determines context budget, chest limits, and triage gate.
    inferred: true
    fields:
      mode:
        values: [lean, balanced, deep]
        default: balanced
      uncertainty_level:
        values: [low, medium, high]
        default: medium
      pressure_score:
        type: integer
        description: Sum of uncertainty signals triggered. Auditable.
      signals_triggered:
        type: list
        description: Which signal rules fired during inference.
      context_policy:
        fields:
          max_sources:
            lean: 2
            balanced: 4
            deep: 8
          max_excerpt_tokens: 450
          require_source_justification: true
      triage_gate:
        fields:
          blocked:
            type: boolean
            default: false
          blocking_question:
            type: string
            description: Present only when blocked=true.
```

**Step 3: Validate YAML parses**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/schemas/intake.schema.yaml'))" && echo OK
```

Expected: `OK`

---

### Task 2: Update `prompt-intake/skill.yaml` with mode inference behavior

**Files:**
- Modify: `strategist/skills/prompt-intake/skill.yaml`

**Step 1: Read the current file**

```bash
cat strategist/skills/prompt-intake/skill.yaml
```

**Step 2: Add `token_strategy` to the output block**

In `output:`, add after `urgency`:

```yaml
  token_strategy:
    mode: string            # lean | balanced | deep
    uncertainty_level: string
    pressure_score: integer
    signals_triggered: list
    context_policy: object
    triage_gate: object
```

**Step 3: Add mode inference algorithm to `behavior:`**

Append to the `behavior:` list:

```yaml
  - Infer token_strategy.mode using this deterministic algorithm:
      Step 1 — base mode from task_type:
        bugfix, docs → lean
        feature, refactor, general → balanced
        architecture_analysis, epic → deep
      Step 2 — apply uncertainty pressure signals (cumulative score):
        +2 if acceptance criteria absent from prompt
        +1 if prompt mentions multiple systems or scopes
        +1 if ambiguous language detected (maybe, something like, not sure, algo assim)
        +1 if prompt word count < 15 AND task_type != bugfix
        -1 if acceptance criteria explicitly present
        -1 if scope clearly declared
      Step 3 — upgrade mode if pressure_score >= 2:
        lean → balanced
        balanced → deep
      Step 4 — economic triage gate:
        Check in priority order: acceptance_criteria_present > scope_bounded > request_clear
        If any critical item missing: set triage_gate.blocked=true
        Generate ONE unblocking question for the highest-priority missing item only
        Do NOT open discovery while triage_gate.blocked=true
  - Record all signals_triggered and pressure_score in token_strategy output (auditable).
  - triage_gate.blocking_question templates by priority:
      acceptance_criteria: "I need to confirm one thing before opening the mission: what should be different when this is done — and how will we know it worked?"
      scope: "I need to confirm one thing before opening the mission: should this change be limited to [inferred scope], or does it touch other systems too?"
      clarity: "I need to confirm one thing before opening the mission: what exactly should be different when this task is complete?"
```

**Step 4: Validate YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/skills/prompt-intake/skill.yaml'))" && echo OK
```

Expected: `OK`

---

### Task 3: Add global stop conditions and token_strategy reference to `skill.yaml`

**Files:**
- Modify: `strategist/skill.yaml`

**Step 1: Add `token_strategy` to the intake stage output**

Find the `intake` pipeline stage and add to its `produces:` or add a note that `mission_contract` now includes `token_strategy`.

Find:
```yaml
  - stage: intake
    skill: prompt-intake
    produces: mission_contract
```

Replace with:
```yaml
  - stage: intake
    skill: prompt-intake
    produces: mission_contract    # now includes token_strategy block (mode, pressure_score, triage_gate)
```

**Step 2: Add triage gate enforcement after intake**

After the intake stage, add:

```yaml
  - stage: triage_gate
    type: conditional_pause
    condition: mission_contract.token_strategy.triage_gate.blocked == true
    description: >
      Present the single unblocking question from token_strategy.triage_gate.blocking_question.
      Do not advance to context_enrichment until user answers. Update mission_contract with answer.
```

**Step 3: Add global token economy stop conditions**

After `budget_policy:`, add:

```yaml
token_economy_stop_conditions:
  - id: ambiguous_task_no_criteria
    trigger: task_is_ambiguous and acceptance_criteria_missing
    action: generate_one_minimum_unblocking_question
    blocks: discovery
  - id: source_no_justification
    trigger: source_has_no_relevance_justification
    action: do_not_load_source
  - id: plan_exceeds_scope
    trigger: execution_plan_exceeds_declared_scope
    action: split_into_separate_missions
  - id: hunter_context_gap
    trigger: sniper_needs_context_not_in_handoff
    action: return_to_archivist
    blocks: re_reading_full_discovery
  - id: unrelated_side_quest
    trigger: side_quest_is_unrelated
    action: record_in_backlog_only
    blocks: refine_in_current_mission
```

**Step 4: Pass `token_strategy` to context_enrichment**

Find the `context_enrichment` stage and add:

```yaml
  - stage: context_enrichment
    skill: context-enrichment
    condition: "!quick_draw_detection.matched"
    input:
      task_type: mission_contract.task_type
      token_strategy: mission_contract.token_strategy    # NEW — drives budget limits
    produces: dossier
```

**Step 5: Add `triage_gate` to `forbidden_behaviors`**

Add to the `forbidden_behaviors:` list:

```yaml
  - skip_triage_gate
  - open_discovery_while_triage_blocked
```

**Step 6: Validate YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/skill.yaml'))" && echo OK
```

Expected: `OK`

---

### Task 4: Add feature spec and fixture for triage gate

**Files:**
- Create: `strategist/tests/specs/token-economy.feature`
- Create: `strategist/tests/fixtures/triage-gate-blocked.yaml`

**Step 1: Write the feature file**

```gherkin
Feature: Token Economy — Mode Inference and Triage Gate
  Invariant: prompt-intake infers token_strategy before discovery opens.
  Source: skill.yaml token_economy_stop_conditions, prompt-intake/skill.yaml behavior.

  Scenario: triage_gate_blocked — discovery blocked when acceptance criteria missing
    Given a user prompt with task_type=feature and no acceptance criteria
    When prompt-intake runs the mode inference algorithm
    Then token_strategy.triage_gate.blocked is true
    And token_strategy.triage_gate.blocking_question is present
    And the pipeline does not advance to context_enrichment

  Scenario: mode_upgraded_by_pressure — balanced upgraded to deep by uncertainty signals
    Given a user prompt mentioning multiple systems with ambiguous language
    When prompt-intake runs the mode inference algorithm
    Then pressure_score is >= 2
    And token_strategy.mode is deep

  Scenario: lean_bug_no_gate — lean bugfix bypasses triage gate
    Given a user prompt classified as task_type=bugfix with clear scope
    When prompt-intake runs the mode inference algorithm
    Then token_strategy.mode is lean
    And token_strategy.triage_gate.blocked is false
```

**Step 2: Write the fixture**

```yaml
scenario: triage_gate_blocked
expected_event: "[Strategist] phase=intake status=blocked reason=triage_gate"
```

**Step 3: Validate YAML fixture**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/tests/fixtures/triage-gate-blocked.yaml'))" && echo OK
```

Expected: `OK`

---

## Phase 2 — Treasure Chest Governance

### Task 5: Create `treasure-chests.yaml` manifest with schema

**Files:**
- Create: `strategist/treasure-chests.yaml`

**Step 1: Write the manifest file**

```yaml
# Treasure Chest Manifest
# Extends knowledge.index.yaml — adds trust, routing, and budget metadata per source.
# Sources declared here MUST also be registered in knowledge.index.yaml.
# Format version: 1

schema_version: "1"
description: >
  Governed offline knowledge sources. Each chest is a trusted, human-reviewed source
  with routing rules, trust tier, and token budget. Discovered sources (found during
  Ranger/Discovery) are NOT listed here until promoted via learning checkpoint.

trust_tiers:
  T0:
    name: canonical
    examples: [mandates, skill.yaml, active.yaml, protocol.md]
    behavior: always_prefer_when_relevant
  T1:
    name: project_docs
    examples: [architecture.md, adr/, configuration.md]
    behavior: load_selectively
  T2:
    name: examples
    examples: [good examples, previous missions]
    behavior: require_task_type_match
  T3:
    name: discovered
    examples: [docs found by search, old notes]
    behavior: use_as_hint_only_until_promoted

budget_defaults_by_mode:
  lean:
    max_sources: 2
    max_total_tokens: 900
  balanced:
    max_sources: 4
    max_total_tokens: 2200
  deep:
    max_sources: 8
    max_total_tokens: 5000

stop_conditions:
  - id: no_task_type_match
    action: do_not_load
  - id: relevance_reason_missing
    action: do_not_load
  - id: source_stale_conflicts_T0
    action: prefer_T0
  - id: context_budget_exceeded
    action: summarize_or_drop_lowest_trust
  - id: discovered_conflicts_trusted
    action: mark_conflict_send_to_archivist
  - id: more_than_2_chests_disagree
    action: stop_and_report_ambiguity

chests: []

# Chest schema (each entry in chests: must follow this structure):
# ---
# id: string                        # unique identifier
# title: string                     # human-readable name
# path: string                      # absolute or relative path
# trust:
#   tier: T0 | T1 | T2 | T3
#   reviewed_by: human | auto
#   last_reviewed: YYYY-MM-DD
# routing:
#   task_types: [list of task_type values]
#   keywords: [list of keywords]
# budget:
#   max_chunks: int
#   max_tokens_per_chunk: int       # default: 450
#   max_total_tokens: int
# retrieval:
#   strategy: selective | full
#   require_relevance_reason: true  # MANDATORY — always true
#   allow_full_load: false          # default: false
```

**Step 2: Validate YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/treasure-chests.yaml'))" && echo OK
```

Expected: `OK`

---

### Task 6: Create source card schema

**Files:**
- Create: `strategist/schemas/source-card.schema.yaml`

**Step 1: Write the schema**

```yaml
schema_version: "1"
description: >
  Source card — the retrieval unit delivered to the LLM. Never a raw chunk.
  Enforces the pattern: evidence → interpretation → impact.

fields:
  id:
    type: string
    format: "<chest-id>#<section-slug>"
    example: "architecture-guides#runtime-slots"
    required: true
  chest:
    type: string
    description: ID of the source chest this card came from
    required: true
  trust:
    type: string
    values: [T0, T1, T2, T3]
    required: true
  excerpt:
    type: string
    description: Verbatim or minimal paraphrase from the source
    required: true
  relevance:
    type: string
    description: Why this excerpt matters for this specific mission
    required: true
  interpretation:
    type: string
    description: What this means for the plan (inferred by Ranger)
    required: true
  impact:
    type: string
    description: What the Archivist must validate based on this card
    required: true
```

**Step 2: Validate YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/schemas/source-card.schema.yaml'))" && echo OK
```

Expected: `OK`

---

### Task 7: Update `skill.yaml` treasure chest section

**Files:**
- Modify: `strategist/skill.yaml`

**Step 1: Replace the minimal `treasure_chest_schema` block**

Find:
```yaml
treasure_chest_schema:
  fields: [id, path, scope, description]
  scope_values: [all, discovery, refinement, execution]
```

Replace with:
```yaml
treasure_chest_schema:
  manifest: "strategist/treasure-chests.yaml"    # full governance manifest
  source_card_schema: "strategist/schemas/source-card.schema.yaml"
  legacy_fields: [id, path, scope, description]  # still supported in active.yaml
  scope_values: [all, discovery, refinement, execution]
  note: >
    Chests declared in active.yaml use legacy_fields format.
    Chests with trust/routing/budget governance are declared in treasure-chests.yaml.
    Both formats are valid. treasure-chests.yaml sources take precedence on conflict.
```

**Step 2: Validate YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/skill.yaml'))" && echo OK
```

Expected: `OK`

---

## Phase 3 — Role Contracts + Opportunity Attack Routine

### Task 8: Create handoff contract schemas

**Files:**
- Create: `strategist/schemas/handoff-ranger-to-archivist.schema.yaml`
- Create: `strategist/schemas/handoff-archivist-to-hunter.schema.yaml`

**Step 1: Write Ranger → Archivist schema**

```yaml
schema_version: "1"
description: >
  Handoff contract from Ranger (discovery) to Archivist (refinement).
  These fields are REQUIRED in the discovery artifact, regardless of format.
  The implementing skill (brainstorming, openspec, custom) decides the file format.

required_fields:
  mission_id:
    type: string
    required: true
  objective:
    type: string
    required: true
  confidence_score:
    type: float
    range: [0.0, 1.0]
    required: true
  known_facts:
    type: list
    required: true
    item_fields: [id, text, evidence_ref]
  uncertainties:
    type: list
    required: false
    item_fields: [id, text, suggested_resolution]
  treasure_chests:
    type: list
    description: From opportunity_attack routine
    required: false
    item_fields: [id, trust, impact, promote_candidate]
  side_quests:
    type: list
    description: From opportunity_attack routine
    required: false
    item_fields: [id, description, relation, suggested_strategy]
    item_values:
      relation: [related, unrelated, duplicate]
      suggested_strategy: [execute_together, execute_later, separate_mission, discard]
  recommended_refinement_focus:
    type: list
    required: false
```

**Step 2: Write Archivist → Hunter schema**

```yaml
schema_version: "1"
description: >
  Handoff contract from Archivist (refinement) to Hunter (execution).
  These fields are REQUIRED in the refinement artifact, regardless of format.
  The implementing skill decides the file format.

required_fields:
  mission_id:
    type: string
    required: true
  approved_scope:
    type: object
    required: true
    fields:
      allowed:
        type: list
        description: Glob patterns of paths Hunter may touch
      forbidden:
        type: list
        description: Glob patterns Hunter must not touch
  implementation_plan:
    type: list
    required: true
    item_fields:
      id: string
      objective: string
      scope: list         # files or paths
      validation: list    # commands or checks
  side_quests_approved:
    type: list
    description: Decisions made by Archivist, surfaced at approval gate
    required: false
    item_fields: [id, strategy]
    item_values:
      strategy: [execute_together, execute_later, separate_mission, discard]
  acceptance_checks:
    type: list
    required: true
  stop_conditions:
    type: list
    required: true
    known_values: [scope_drift, missing_file, validation_failure, ambiguity]
```

**Step 3: Validate both YAML files**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/schemas/handoff-ranger-to-archivist.schema.yaml'))" && echo ranger-ok
python3 -c "import yaml; yaml.safe_load(open('strategist/schemas/handoff-archivist-to-hunter.schema.yaml'))" && echo archivist-ok
```

Expected: `ranger-ok` then `archivist-ok`

---

### Task 9: Create discovery artifact template

**Files:**
- Create: `strategist/templates/discovery-artifact.md`

**Step 1: Write the template**

````markdown
# Discovery Brief — {{mission_id}}

> **Format note:** This is a reference template. The implementing skill decides the actual file format.
> Required handoff contract fields: see `schemas/handoff-ranger-to-archivist.schema.yaml`.

## Mission

**Objective:** {{objective}}
**Task type:** {{task_type}}
**Mode:** {{token_strategy.mode}}

## Confidence

**Score:** {{confidence_score}} (0.0–1.0)
**Uncertainty level:** {{uncertainty_level}}

**Ambiguities:**
- {{ambiguity_1}}

**Blockers:**
- {{blocker_1}} (or: none)

## Evidence Cards

### E1 — {{title}}

- **Source:** `{{path/to/file}}`
- **Evidence:** {{verbatim or minimal paraphrase}}
- **Interpretation:** {{what this means for the plan}}
- **Impact:** {{what Archivist must validate}}

## Treasure Chests Used

| Chest | Trust | Why loaded | Use in refinement |
|---|---|---|---|
| `{{chest-id}}` | {{T0–T3}} | {{reason}} | {{use}} |

## Side Quests Detected

| ID | Item | Relation | Suggested strategy |
|---|---|---|---|
| SQ-1 | {{description}} | {{related\|unrelated\|duplicate}} | {{strategy}} |

## Recommended Refinement Focus

- {{focus_1}}
- {{focus_2}}
````

---

### Task 10: Update `ranger.yaml` with full contract

**Files:**
- Modify: `strategist/roles/ranger.yaml`

**Step 1: Read the current file**

```bash
cat strategist/roles/ranger.yaml
```

**Step 2: Replace the content with the full contract**

```yaml
role: ranger
slot: discovery

# Canonical behaviors — read-only, source in skill.yaml.
canonical:
  - load_mission_contract                     # read token_strategy.mode for context budget
  - consult_treasure_chests                   # consult T0/T1 chests scoped to discovery
  - opportunity_attack:                       # cross-cutting routine — mandatory
      detect: [treasure_chests, side_quests]
      output: include_in_discovery_artifact   # handoff contract fields
      feedback: surface_in_response           # show to user before handing off to Archivist
  - compress_findings:                        # evidence → interpretation → impact
      format: source_card
  - output_format: "<base_path>/pending/<mission_id>-discovery.md"
  - handoff_contract: "schemas/handoff-ranger-to-archivist.schema.yaml"

# Must / must_not contract
must:
  - separate facts, hypotheses, and ambiguities in the discovery artifact
  - surface opportunity_attack findings in response to user before finishing
  - include all handoff contract fields in the discovery artifact
  - use the discovery artifact template as format reference (schemas/discovery-artifact.md)

must_not:
  - propose a final plan as if approved
  - execute any changes
  - pass raw context to Archivist (compress to evidence cards)
  - hide ambiguities or uncertainties

# Additional instructions for this installation.
custom_brief: ""
```

**Step 3: Validate YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/roles/ranger.yaml'))" && echo OK
```

Expected: `OK`

---

### Task 11: Update `archivist.yaml` with full contract

**Files:**
- Modify: `strategist/roles/archivist.yaml`

**Step 1: Replace content with full contract**

```yaml
role: archivist
slot: refinement

# Canonical behaviors — read-only, source in skill.yaml.
canonical:
  - load_ranger_discovery_artifact            # read handoff contract fields only, not raw history
  - consult_treasure_chests                   # consult chests scoped to refinement
  - opportunity_attack:                       # cross-cutting routine — mandatory
      detect: [side_quests]                   # Archivist finds more side quests during refinement
      decide: side_quest_strategy             # execute_together | execute_later | separate_mission | discard
      output: include_in_refinement_artifact  # handoff contract fields
      feedback: include_in_approval_gate      # shown alongside the plan at approval gate
  - emit_completion_signal                    # must emit has_execution_tasks: true|false
  - output_format:
      - "<base_path>/refined/<mission_id>/proposal.md"
      - "<base_path>/refined/<mission_id>/design.md"
      - "<base_path>/refined/<mission_id>/tasks.md"
  - handoff_contract: "schemas/handoff-archivist-to-hunter.schema.yaml"

must:
  - load only the Ranger discovery artifact (do NOT re-read full discovery history)
  - decide side_quest_strategy for every side quest in the handoff
  - convert ambiguities into decisions or explicit blockers
  - produce numbered tasks with scope and validation steps
  - declare allowed and forbidden file scope in handoff
  - include all handoff contract fields in the refinement output

must_not:
  - invent evidence not present in the Ranger artifact
  - execute tasks
  - expand scope without marking it as an explicit decision

custom_brief: ""
```

**Step 2: Validate YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/roles/archivist.yaml'))" && echo OK
```

Expected: `OK`

---

### Task 12: Update `sniper.yaml` (hunter) with full contract

**Files:**
- Modify: `strategist/roles/sniper.yaml`

**Step 1: Replace content with full contract**

```yaml
role: sniper
slot: execution

# Note: "sniper" is the implementation alias; "hunter" is the spec alias.
# Both names refer to the same role.

# Canonical behaviors — read-only, source in skill.yaml.
canonical:
  - requires_approval_gate                    # enforced by Strategist — slot cannot bypass
  - load_archivist_handoff                    # read handoff contract fields + approved scope
  - consult_treasure_chests                   # consult chests scoped to execution
  - execute_one_task_per_loop                 # never batch multiple tasks
  - opportunity_attack:                       # cross-cutting routine — mandatory
      detect: [side_quests]
      action: stop_and_report                 # Hunter DOES NOT decide — reports only
      feedback: surface_immediately_pause_execution
  - update_checklist_before_advancing
  - handoff_contract_consumed: "schemas/handoff-archivist-to-hunter.schema.yaml"

must:
  - declare the active task at the start of each loop
  - run targeted validation after each task (not broad test suite unless specified)
  - update the task checklist before moving to the next task
  - stop immediately and report if scope drift, ambiguity, or validation failure detected
  - surface opportunity_attack findings immediately and pause

must_not:
  - execute without the approval gate having been granted
  - execute multiple tasks in a single loop
  - modify files outside the approved_scope.allowed list
  - re-open or re-read full discovery history
  - write a new plan — return to Archivist instead
  - decide side quest strategy (report only)
  - use context not in the Archivist handoff (return to Archivist if needed)

custom_brief: ""
```

**Step 2: Validate YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/roles/sniper.yaml'))" && echo OK
```

Expected: `OK`

---

### Task 13: Migrate `opportunity_attack` from pipeline stage to canonical routine in `skill.yaml`

This is the most impactful change. The dedicated `opportunity_attack` stage, `opportunity_gate`, and `opportunity_execution` stages are replaced by the routine in each role.

**Files:**
- Modify: `strategist/skill.yaml`

**Step 1: Read the current opportunity_attack, opportunity_gate, opportunity_execution stages**

```bash
grep -n "opportunity" strategist/skill.yaml
```

**Step 2: Remove the three dedicated stages**

Remove these pipeline stages:
- `opportunity_attack` (internal scan stage after discovery)
- `opportunity_gate` (conditional pause after opportunity_attack)
- `opportunity_execution` (slot execution after opportunity_gate)

**Step 3: Update refinement stage input**

The refinement stage currently takes `[discovery_artifact, opportunity_report]`. Since opportunity_attack is now inside each role, remove `opportunity_report`:

Find:
```yaml
  - stage: refinement
    slot: refinement
    input: [discovery_artifact, opportunity_report]
```

Replace with:
```yaml
  - stage: refinement
    slot: refinement
    input: discovery_artifact
    note: >
      Opportunity Attack runs inside the Ranger and Archivist roles.
      Side quests and chest candidates are included in their handoff contract fields.
      Results are surfaced at the approval_gate — not in a separate gate.
```

**Step 4: Update `forbidden_behaviors`**

Remove the now-obsolete forbidden behaviors:
- `run_opportunity_attack_as_slot` (was preventing misuse of old model; now the routine IS in roles)
- `skip_opportunity_gate` (gate no longer exists as separate stage)
- `invoke_opportunity_sniper_without_approval` (opportunity_execution stage removed)

Add new forbidden behaviors:
```yaml
  - skip_opportunity_attack_routine          # each role MUST run it
  - suppress_opportunity_attack_feedback     # findings MUST be shown to user
  - hunter_decides_side_quest_strategy       # only Archivist decides; Hunter reports
  - open_discovery_while_triage_blocked      # triage_gate.blocked=true blocks discovery
```

**Step 5: Add `opportunity_attack` to slot canonical behaviors in skill.yaml**

In the `slots:` section, add `opportunity_attack` as a canonical behavior to each slot:

```yaml
slots:
  discovery:
    canonical:
      - find_unexpected_items
      - consult_treasure_chests
      - opportunity_attack: {detect: [treasure_chests, side_quests], feedback: surface_in_response}
      - output_format: "<base_path>/pending/<mission_id>-discovery.md"

  refinement:
    canonical:
      - consult_treasure_chests
      - opportunity_attack: {detect: [side_quests], decide: side_quest_strategy, feedback: include_in_approval_gate}
      - emit_completion_signal
      - output_format: [...]

  execution:
    canonical:
      - requires_approval_gate
      - consult_treasure_chests
      - opportunity_attack: {detect: [side_quests], action: stop_and_report, feedback: surface_immediately}
      - output_format: "<base_path>/done/<mission_id>-report.md"
```

**Step 6: Validate YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/skill.yaml'))" && echo OK
```

Expected: `OK`

---

### Task 14: Update feature spec for opportunity_attack as routine

**Files:**
- Modify: `strategist/tests/specs/forbidden-behaviors.feature`
- Modify: `strategist/tests/fixtures/side-quest-bypass.yaml`

**Step 1: Replace the `side_quest_approval_bypass` scenario**

The old scenario described a separate gate that no longer exists. Replace it:

```gherkin
  Scenario: skip_opportunity_attack_routine — role completes without running the routine
    Given the Ranger has completed discovery
    When the Ranger's response does not include an Opportunity Attack section
    Then Strategist detects drift pattern "skip_opportunity_attack_routine"
    And surfaces the missing check to the user
    And requests Ranger to re-run before passing to Archivist

  Scenario: suppress_opportunity_attack_feedback — findings hidden from user
    Given Ranger ran opportunity_attack and detected side quests
    When Strategist advances to Archivist without showing opportunity_attack findings to the user
    Then Strategist detects drift pattern "suppress_opportunity_attack_feedback"
    And stops the handoff
    And presents the opportunity_attack findings to the user

  Scenario: hunter_decides_side_quest — Hunter sets side quest strategy
    Given Hunter detected a side quest during execution
    When Hunter sets side_quest.strategy without returning to Archivist
    Then Strategist detects drift pattern "hunter_decides_side_quest_strategy"
    And voids the Hunter's decision
    And routes the side quest to Archivist for strategy decision
```

**Step 2: Update the fixture**

```yaml
scenario: skip_opportunity_attack_routine
expected_event: "[Strategist] phase=discovery status=drift pattern=skip_opportunity_attack_routine"
```

**Step 3: Validate YAML fixture**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/tests/fixtures/side-quest-bypass.yaml'))" && echo OK
```

Expected: `OK`

**Step 4: Run existing Go tests to check for regressions**

```bash
cd strategist && go test ./tests/... -v 2>&1 | tail -20
```

Expected: tests that depended on the old `side_quest_approval_bypass` scenario may fail — fix by aligning the test to the new scenario name `skip_opportunity_attack_routine`.

**Step 5: Fix any failing test**

In `spec_alignment_test.go`, find the test referencing `side_quest_approval_bypass` and update the scenario name and expected event strings to match the updated feature file and fixture.

---

### Task 15: Add `token-economy` feature spec test to Go harness

**Files:**
- Modify: `strategist/tests/spec_alignment_test.go`

**Step 1: Add test for triage gate fixture**

```go
func TestTriageGateFixtureAlignedWithTokenEconomySpec(t *testing.T) {
    t.Parallel()
    featurePath := filepath.Join("specs", "token-economy.feature")
    fixturePath := filepath.Join("fixtures", "triage-gate-blocked.yaml")

    feature := readFile(t, featurePath)
    fixture := readFixture(t, fixturePath)

    if !strings.Contains(feature, "triage_gate_blocked") {
        t.Fatalf("%s missing scenario triage_gate_blocked", featurePath)
    }
    if fixture.Scenario != "triage_gate_blocked" {
        t.Fatalf("%s scenario must be triage_gate_blocked, got: %q", fixturePath, fixture.Scenario)
    }
    if !strings.Contains(fixture.ExpectedEvent, "blocked") {
        t.Fatalf("%s expected_event must contain blocked, got: %q", fixturePath, fixture.ExpectedEvent)
    }
}
```

**Step 2: Run tests**

```bash
cd strategist && go test ./tests/... -v -run TestTriageGate 2>&1
```

Expected: `PASS`

---

## Phase 4 — Structural Compression Pipeline (Contract Layer)

### Task 16: Evolve `context-enrichment/skill.yaml` to be mode-aware

**Files:**
- Modify: `strategist/skills/context-enrichment/skill.yaml`

**Step 1: Read the current file**

```bash
cat strategist/skills/context-enrichment/skill.yaml
```

**Step 2: Add `token_strategy` to the input contract**

In `contract.input:`, add:

```yaml
    - name: token_strategy
      type: object
      required: false
      description: >
        If present, drives source selection limits.
        token_strategy.mode determines max_sources and max_total_tokens.
        Falls back to defaults if absent.
```

**Step 3: Add mode-aware limits to the contract**

Add a new section:

```yaml
  mode_policy:
    lean:
      max_sources: 2
      max_total_tokens: 900
    balanced:
      max_sources: 4
      max_total_tokens: 2200
    deep:
      max_sources: 8
      max_total_tokens: 5000
    default: balanced
```

**Step 4: Update the output contract to reference source cards**

In `contract.output:`, add:

```yaml
    - name: source_cards
      type: list
      description: >
        Source cards assembled by dossier-builder. Each card follows source-card.schema.yaml.
        Present when treasure-chests.yaml is configured and chests were loaded.
```

**Step 5: Add chest stop conditions**

Add:

```yaml
  chest_stop_conditions:
    - no_task_type_match: do_not_load
    - relevance_reason_missing: do_not_load
    - source_stale_conflicts_T0: prefer_T0
    - context_budget_exceeded: summarize_or_drop_lowest_trust
```

**Step 6: Validate YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/skills/context-enrichment/skill.yaml'))" && echo OK
```

Expected: `OK`

---

### Task 17: Evolve `dossier-builder/skill.yaml` to produce source cards

**Files:**
- Modify: `strategist/skills/dossier-builder/skill.yaml`

**Step 1: Read the current file**

```bash
cat strategist/skills/dossier-builder/skill.yaml
```

**Step 2: Add `source_cards` and `mode` to the input**

In `input:`, add:

```yaml
  source_cards: list             # from context-enrichment, if treasure-chests.yaml configured
  mode: string                   # lean | balanced | deep — drives card limits
```

**Step 3: Add source card assembly to the output**

In `output.dossier:`, add:

```yaml
    source_cards: list           # max cards by mode: lean=5, balanced=10, deep=20
    compression_metrics: object  # sources_seen, sources_selected, tokens_estimated
```

**Step 4: Add source card assembly behavior**

In `behavior:`, append:

```yaml
  - If source_cards is non-empty: assemble dossier source_cards section using source-card.schema.yaml.
    Apply card limits by mode: lean=5, balanced=10, deep=20.
    Prefer cards with higher trust tier (T0 > T1 > T2 > T3).
    Each card must include: id, chest, trust, excerpt, relevance, interpretation, impact.
  - Log compression_metrics: sources_seen (from context-enrichment), sources_selected (cards included),
    tokens_estimated (sum of excerpt lengths).
```

**Step 5: Validate YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/skills/dossier-builder/skill.yaml'))" && echo OK
```

Expected: `OK`

---

### Task 18: Add compression policy and metrics to `skill.yaml`

**Files:**
- Modify: `strategist/skill.yaml`

**Step 1: Add compression policy block**

After `budget_policy:`, add:

```yaml
compression_policy:
  description: >
    Governs how context-enrichment and dossier-builder select and compress sources.
    Driven by token_strategy.mode from mission_contract.
  default_provider: builtin-structural
  modes:
    lean:
      max_sources: 2
      max_cards: 5
      max_dossier_tokens: 900
      external_provider: false
    balanced:
      max_sources: 4
      max_cards: 10
      max_dossier_tokens: 2200
      external_provider: optional
    deep:
      max_sources: 8
      max_cards: 20
      max_dossier_tokens: 5000
      external_provider: preferred
  providers:
    - id: builtin-structural
      type: internal
      required: true
      capabilities: [normalize, deduplicate, rank, filter, chunk, source_cards, dossier]
    - id: semantic-context-server
      type: external
      required: false
      endpoint: "http://localhost:8711"
    - id: caveman
      type: external
      required: false
      command: "caveman compress"
  fallback:
    on_provider_missing: builtin-structural
    on_provider_error: builtin-structural
  security:
    external_provider_may_return: [dossier, source_cards, ranking, metrics, warnings]
    external_provider_may_not: [scope_decisions, execution, side_quest_approval, mission_contract_changes]
  metrics_log: ".strategist/memory/chest-usage.jsonl"
```

**Step 2: Add metrics logging note to the pipeline**

After the `context_enrichment` stage, add:

```yaml
  - stage: compression_metrics
    type: internal
    description: >
      Log dossier-builder compression_metrics to .strategist/memory/chest-usage.jsonl.
      Non-blocking. Format: JSON lines with mission_id, mode, sources_seen, sources_selected,
      cards_created, estimated_tokens, compression_ratio, provider_used.
    blocking: false
```

**Step 3: Validate YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('strategist/skill.yaml'))" && echo OK
```

Expected: `OK`

---

### Task 19: Run full test suite and validate spec alignment

**Step 1: Run all Go tests**

```bash
cd strategist && go test ./tests/... -v 2>&1
```

Expected: All tests pass. If any fail, read the failure output and fix the misalignment (usually a fixture or feature file needing updating).

**Step 2: Validate all modified YAML files**

```bash
for f in strategist/skill.yaml strategist/schemas/intake.schema.yaml strategist/treasure-chests.yaml strategist/schemas/source-card.schema.yaml strategist/schemas/handoff-ranger-to-archivist.schema.yaml strategist/schemas/handoff-archivist-to-hunter.schema.yaml strategist/roles/ranger.yaml strategist/roles/archivist.yaml strategist/roles/sniper.yaml strategist/skills/prompt-intake/skill.yaml strategist/skills/context-enrichment/skill.yaml strategist/skills/dossier-builder/skill.yaml; do
  python3 -c "import yaml; yaml.safe_load(open('$f'))" && echo "OK: $f" || echo "FAIL: $f"
done
```

Expected: All `OK`.

**Step 3: Check that removed forbidden behaviors are no longer referenced in feature files**

```bash
grep -r "skip_opportunity_gate\|invoke_opportunity_sniper_without_approval\|run_opportunity_attack_as_slot" strategist/tests/
```

Expected: No matches (these behaviors were removed in Task 13).

**Step 4: Check that new forbidden behaviors appear in the feature file**

```bash
grep -l "skip_opportunity_attack_routine\|suppress_opportunity_attack_feedback\|hunter_decides_side_quest" strategist/tests/specs/
```

Expected: `forbidden-behaviors.feature` (updated in Task 14).
