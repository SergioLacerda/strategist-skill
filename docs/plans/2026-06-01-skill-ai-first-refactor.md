# SKILL.md AI-First Refactor + Output Verbosity — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce SKILL.md from 715 to ~200 lines by migrating phase detail into contracts/roles, and add per-invocation output verbosity via OTEL-aligned severity levels.

**Architecture:** SKILL.md becomes a lean pipeline manifest (routing + invariants only). Phase procedures move into their canonical contract/role YAML files. Output verbosity is controlled by a new `--output` flag and an `emit-taxonomy.yaml` that maps each emit key to a severity level (TRACE/DEBUG/INFO), aligned with `sdd_telemetry/constants.py`.

**Tech Stack:** YAML (contracts, roles, test files), Markdown (SKILL.md). No executable code — contract tests are declarative behavioral specs in YAML.

---

## Test Convention

Contract tests live in `.strategist/contracts/tests/<name>.test.yaml`. Structure:

```yaml
subject: contracts/<name>.yaml   # or roles/<name>.yaml
version: "1.0"

invariants:
  - id: <snake_case_id>
    description: <what must always be true>
    rule: <machine-readable assertion>

cases:
  - name: <scenario_name>
    given:
      <input_field>: <value>
    expect:
      <output_field>: <value>
      emit: <event string>
```

Tests are not executed by a test runner — they are behavioral contracts read by the LLM agent. "Passing" means the contract YAML satisfies every invariant declared in its test file. Self-verification step: after writing a contract, read both files and confirm each invariant holds.

---

## Task 1: Test Infrastructure Setup

**Files:**
- Create: `.strategist/contracts/tests/.gitkeep`

**Step 1: Create the tests directory**

```bash
mkdir -p .strategist/contracts/tests
touch .strategist/contracts/tests/.gitkeep
```

**Step 2: Verify directory exists**

```bash
ls .strategist/contracts/tests/
```
Expected: `.gitkeep`

---

## Task 2: Output Profiles — Emit Taxonomy + Profiles

**Files:**
- Create: `.strategist/output-profiles/emit-taxonomy.yaml`
- Create: `.strategist/output-profiles/profiles/default.yaml`
- Create: `.strategist/output-profiles/profiles/verbose.yaml`
- Create: `.strategist/output-profiles/profiles/full.yaml`
- Create: `.strategist/contracts/tests/emit-taxonomy.test.yaml`

**Step 1: Write the emit-taxonomy test first**

Create `.strategist/contracts/tests/emit-taxonomy.test.yaml`:

```yaml
subject: output-profiles/emit-taxonomy.yaml
version: "1.0"

invariants:
  - id: all_keys_have_level
    description: Every emit key must declare a severity level
    rule: for all keys in emits: level is present and non-empty

  - id: valid_levels_only
    description: Only TRACE, DEBUG, or INFO are valid levels
    rule: emits[*].level in [TRACE, DEBUG, INFO]

  - id: compliance_summary_is_info
    description: compliance_summary is always visible regardless of output profile
    rule: emits.compliance_summary == INFO

  - id: phase_status_is_trace
    description: Pipeline telemetry lines are TRACE — suppressed in default mode
    rule: emits.phase_status == TRACE

  - id: pipeline_starting_is_info
    description: Mission start is always visible
    rule: emits.pipeline_starting == INFO

cases:
  - name: default_profile_hides_telemetry
    given:
      output_threshold: INFO
    expect:
      visible: [pipeline_starting, ranger_start, archivist_done, compliance_summary]
      hidden: [phase_status, opportunity_attack_done, bootstrap_path]

  - name: verbose_profile_shows_narrative
    given:
      output_threshold: DEBUG
    expect:
      visible: [opportunity_attack_header, side_quest_none_found, treasure_chest_found]
      hidden: [phase_status, opportunity_attack_done]

  - name: full_profile_shows_everything
    given:
      output_threshold: TRACE
    expect:
      visible: [phase_status, opportunity_attack_done, bootstrap_path, pipeline_starting]
```

**Step 2: Create emit-taxonomy.yaml**

Create `.strategist/output-profiles/emit-taxonomy.yaml`:

```yaml
# Severity levels align with sdd_telemetry/constants.py (TRACE=1, DEBUG=5, INFO=9)
# and internal/telemetry/schema.go (strategist.* attribute namespace).
# OTEL export is never filtered — this taxonomy controls stdout only.

version: "1.0"

emits:
  # TRACE — pipeline telemetry ([Strategist] phase=X status=Y lines)
  # Suppressed in default and verbose output profiles.
  phase_status:                TRACE
  phase_skipped:               TRACE
  opportunity_attack_done:     TRACE
  learning_buffer_flushed:     TRACE
  bootstrap_path:              TRACE
  preflight_path:              TRACE
  context_enrichment_path:     TRACE
  policy_eval:                 TRACE
  phase_blocked:               TRACE

  # DEBUG — internal narrative events (opportunity attack, side quests)
  # Visible in verbose and full profiles. Hidden in default.
  opportunity_attack_header:   DEBUG
  side_quest_none_found:       DEBUG
  side_quest_detected:         DEBUG
  treasure_chest_found:        DEBUG
  adr_opportunity:             DEBUG

  # INFO — always visible to user regardless of output profile
  pipeline_starting:           INFO
  ranger_start:                INFO
  ranger_done:                 INFO
  archivist_start:             INFO
  archivist_done:              INFO
  sniper_start:                INFO
  approval_prompt:             INFO
  execution_tasks_header:      INFO
  execution_task_line:         INFO
  mission_checkpoint:          INFO
  compliance_summary:          INFO
```

**Step 3: Create profile files**

Create `.strategist/output-profiles/profiles/default.yaml`:
```yaml
profile: default
level_threshold: INFO
description: Key milestones visible to user. Pipeline telemetry suppressed.
```

Create `.strategist/output-profiles/profiles/verbose.yaml`:
```yaml
profile: verbose
level_threshold: DEBUG
description: Includes opportunity attack narrative and internal events.
```

Create `.strategist/output-profiles/profiles/full.yaml`:
```yaml
profile: full
level_threshold: TRACE
description: All output including [Strategist] pipeline telemetry lines.
```

**Step 4: Self-verify against test invariants**

Read `emit-taxonomy.test.yaml` and `emit-taxonomy.yaml` together. Confirm:
- `compliance_summary` is `INFO` ✓
- `phase_status` is `TRACE` ✓
- All keys have a level ✓
- No level other than TRACE/DEBUG/INFO ✓

---

## Task 3: New Contract — intake.yaml

**Files:**
- Create: `.strategist/contracts/intake.yaml`
- Create: `.strategist/contracts/tests/intake.test.yaml`

**Step 1: Write the intake test first**

Create `.strategist/contracts/tests/intake.test.yaml`:

```yaml
subject: contracts/intake.yaml
version: "1.0"

invariants:
  - id: mission_id_required
    description: Intake must produce a mission_id
    rule: output.mission_id is present and non-empty

  - id: quick_draw_route_explicit
    description: Quick Draw route only triggers on explicit keyword match
    rule: quick_draw_triggered only when prompt matches quick_draw_triggers pattern

  - id: conflict_stops_pipeline
    description: Constraint conflict must stop intake and ask user to resolve
    rule: prompt-intake conflict → stop, do not proceed to phases

  - id: mission_checkpoint_after_intake
    description: Mission checkpoint emitted immediately after intake completes
    rule: after intake_complete → emit mission_checkpoint with step_1_icon=⏳

cases:
  - name: normal_mission
    given:
      prompt: "refactor the authentication module"
    expect:
      output.task_type: refactor
      output.mission_id: present
      route: standard
      emit: mission_checkpoint

  - name: quick_draw_triggered
    given:
      prompt: "quick draw: ideia sobre cache layer"
    expect:
      route: quick_draw
      emit: side_quest_detected description="Quick Draw — rapid idea capture."

  - name: constraint_conflict
    given:
      prompt_intake_result: conflict=true
    expect:
      action: stop_and_ask_user
      next_phase: none
```

**Step 2: Create intake.yaml** (migrating §3 + §3.1 + §3.2 from SKILL.md)

Create `.strategist/contracts/intake.yaml`:

```yaml
module: intake
type: agent_phase
description: >
  Classifies the user prompt, generates mission_id, emits mission checkpoint,
  and routes quick-draw captures to the side-quest pipeline.

contract:
  input:
    - name: user_prompt
      type: string
      required: true
      description: Raw user invocation text
    - name: output_threshold
      type: string
      required: false
      default: INFO
      description: Active output profile threshold (loaded during bootstrap)

  output:
    - name: task_type
      type: string
      description: Classification from prompt-intake (architecture_analysis, refactor, general, etc.)
    - name: risk_level
      type: string
      description: Estimated mission risk
    - name: constraints
      type: object
      description: delivery_strategy, legacy_compatibility, execution_intent
    - name: mission_id
      type: string
      description: Generated unique mission identifier
    - name: route
      type: enum[standard, quick_draw]
      description: Determines which pipeline to run

  error_conditions:
    - code: constraint_conflict
      trigger: prompt-intake returns conflicting constraints
      behavior: Stop. Ask user to resolve conflict. Do not proceed to phases.

  quick_draw_triggers:
    - "quick draw"
    - "saque rapido"
    - "saque rápido"
    - prompt starts with "TODO"

  quick_draw_behavior: >
    If any trigger matches: emit side_quest_detected with description="Quick Draw — rapid idea
    capture." before routing. Do NOT run regular intake classification. Skip §4–§7.
    Execute only quick_draw pipeline (§5.0).

  mission_checkpoint:
    emit_key: mission_checkpoint
    after: intake_complete
    initial_state:
      step_1_icon: "⏳"
      step_2_icon: "⬜"
      step_3_icon: "⬜"
      step_4_icon: "⬜"
    transitions:
      after_ranger_done:    { step_1: "✅", step_2: "⏳" }
      after_archivist_done: { step_1: "✅", step_2: "✅", step_3: "⏳" }
      after_gate_approved:  { step_1: "✅", step_2: "✅", step_3: "✅", step_4: "⏳" }
      after_sniper_done:    { step_1: "✅", step_2: "✅", step_3: "✅", step_4: "✅" }
    skip_reemit_on: plan_only

  write_scope: read-only
  owner: internal (orchestrator)
```

**Step 3: Self-verify against test invariants**

Read both files. Confirm each invariant holds in the contract definition.

---

## Task 4: New Contract — opportunity-attack.yaml

**Files:**
- Create: `.strategist/contracts/opportunity-attack.yaml`
- Create: `.strategist/contracts/tests/opportunity-attack.test.yaml`

**Step 1: Write the test first**

Create `.strategist/contracts/tests/opportunity-attack.test.yaml`:

```yaml
subject: contracts/opportunity-attack.yaml
version: "1.0"

invariants:
  - id: runs_inside_every_role
    description: Opportunity attack is mandatory inside Ranger, Archivist, and Sniper
    rule: roles [ranger, archivist, sniper] must each invoke opportunity_attack

  - id: emit_always_produced
    description: Emit is produced even when items=0
    rule: after opportunity_attack completes → emit phase=<role> opportunity_attack=done|failed

  - id: non_blocking_on_error
    description: Technical failure does not stop the pipeline
    rule: opportunity_attack_failed → emit failed reason=<why>, continue to next step

  - id: narrow_scope_no_exception
    description: Single-file missions do not skip opportunity attack
    rule: "foco em alvo único" is not a valid skip reason

cases:
  - name: ranger_no_items
    given:
      role: ranger
      items_found: 0
    expect:
      emit: "[Strategist] phase=ranger opportunity_attack=done items=0"

  - name: archivist_side_quests
    given:
      role: archivist
      side_quests_found: 2
    expect:
      emit: "[Strategist] phase=archivist opportunity_attack=done side_quests=2"

  - name: technical_error
    given:
      role: sniper
      error: "file not found"
    expect:
      emit: "[Strategist] phase=sniper opportunity_attack=failed reason=file not found"
      pipeline: continues
```

**Step 2: Create opportunity-attack.yaml** (migrating §5.-1 from SKILL.md)

Create `.strategist/contracts/opportunity-attack.yaml`:

```yaml
module: opportunity-attack
type: invariant
description: >
  Mandatory routine that runs INSIDE each role (Ranger, Archivist, Sniper).
  Surfaces items outside the declared mission scope. Not a standalone phase.

contract:
  roles_that_must_run: [ranger, archivist, sniper]

  per_role_emit:
    ranger:   "[Strategist] phase=ranger opportunity_attack=done items=<N>"
    archivist: "[Strategist] phase=archivist opportunity_attack=done side_quests=<N>"
    sniper:
      done:      "[Strategist] phase=sniper opportunity_attack=done items=0"
      triggered: "[Strategist] phase=sniper opportunity_attack=triggered items=<N>"

  error_emit: "[Strategist] phase=<role> opportunity_attack=failed reason=<why>"

  invariants:
    - emit always produced (items=0 is valid, silence is not)
    - technical failure is non-blocking — log and continue
    - '"foco em alvo único" is NOT a valid reason to skip'

  write_scope: read-only (surfaces only; does not act on findings without a gate)
  owner: internal (orchestrator)
```

**Step 3: Self-verify** — read both files, confirm all invariants hold.

---

## Task 5: New Contract — quick-draw.yaml

**Files:**
- Create: `.strategist/contracts/quick-draw.yaml`
- Create: `.strategist/contracts/tests/quick-draw.test.yaml`

**Step 1: Write the test first**

Create `.strategist/contracts/tests/quick-draw.test.yaml`:

```yaml
subject: contracts/quick-draw.yaml
version: "1.0"

invariants:
  - id: gate_mandatory
    description: Quick draw gate cannot be bypassed — must stop and wait for sim/nao
    rule: after archivist completes → STOP, show gate, wait for response before any write

  - id: no_write_before_gate
    description: Sniper must not append before gate is approved
    rule: sim/nao gate must precede any file write

  - id: path_resolves_per_language
    description: Destination path uses the correct bucket for active.language.chat
    rule: >
      pt-BR → .analysis/todo/<arquitetura|seguranca|analise|geral>.md
      en    → .analysis/todo/<architecture|security|analysis|general>.md

cases:
  - name: user_approves
    given:
      gate_response: "sim"
      normalized_idea: "ideia sobre cache layer"
      theme: arquitetura
    expect:
      action: append to .analysis/todo/arquitetura.md
      output: "sucesso: ideia adicionada em .analysis/todo/arquitetura.md"

  - name: user_declines
    given:
      gate_response: "nao"
    expect:
      action: return without writing
      output: none

  - name: policy_blocked
    given:
      policy_eval: blocked
    expect:
      emit: "[Strategist] phase=policy_eval status=blocked"
      action: stop with reason=policy_blocked
```

**Step 2: Create quick-draw.yaml** (migrating §5.0 from SKILL.md)

Create `.strategist/contracts/quick-draw.yaml`:

```yaml
module: quick-draw
type: agent_phase
description: >
  Side-quest pipeline for rapid idea capture. Triggered when intake detects a
  quick-draw keyword. Runs a minimal Ranger → Archivist → gate → Sniper pipeline.
  Does NOT run opportunity attack (scope is intentionally narrow).

contract:
  pipeline: ranger_organize → archivist_theme → gate → sniper_append

  ranger_behavior:
    input: original quick note prompt
    output: one normalized line — ideia: <formalization without expanding scope>
    must_not: add requirements, milestones, or implementation details

  archivist_behavior:
    determine_theme:
      pt-BR: [arquitetura, seguranca, analise, geral]
      en:    [architecture, security, analysis, general]
    resolve_path: "<base_path>/todo/<bucket>.md"
    compute:
      - total_ideas: count of existing entries in destination file
      - similar_ideas: entries with textual similarity to normalized idea

  gate:
    display: |
      ideia: <texto_normalizado>
      adicionar ideia? (sim/nao)
    stop: true
    wait_for_response: true
    responses:
      sim: proceed to policy_eval then sniper_append
      nao: return without writing

  policy_eval:
    transition_group: finalize_analysis
    emit: "[Strategist] phase=policy_eval status=<allowed|blocked> mission=<id> mode=<mode> can_execute=<bool> transition_group=finalize_analysis"
    on_blocked: stop with reason=policy_blocked

  sniper_behavior:
    action: append new entry to <base_path>/todo/<tema>.md
    entry_format: timestamp + normalized idea
    return:
      - "sucesso: ideia adicionada em <path>"
      - "total de ideias: X"
      - "ideias similares (mesmo tema): Y"

  write_scope: <base_path>/todo/ (append-only, single file per invocation)
  owner: internal (side-quest)
```

**Step 3: Self-verify** — read both files, confirm all invariants hold.

---

## Task 6: New Contract — approval-gate.yaml

**Files:**
- Create: `.strategist/contracts/approval-gate.yaml`
- Create: `.strategist/contracts/tests/approval-gate.test.yaml`

**Step 1: Write the test first**

Create `.strategist/contracts/tests/approval-gate.test.yaml`:

```yaml
subject: contracts/approval-gate.yaml
version: "1.0"

invariants:
  - id: gate_cannot_be_skipped
    description: Sniper must never be invoked without approval gate completing first
    rule: phase=approval_gate must emit status=approved or status=plan_only before sniper starts

  - id: plan_only_not_blocked
    description: Declining the gate is plan_only, not an error
    rule: response=no|decline → status=plan_only (not status=blocked)

  - id: checkpoint_update_on_approve
    description: Mission checkpoint updates when gate is approved
    rule: gate approved → re-emit checkpoint with step_3=✅ step_4=⏳

cases:
  - name: user_approves
    given:
      gate_response: "yes"
    expect:
      emit: "[Strategist] phase=approval_gate status=approved"
      checkpoint_update: { step_3: "✅", step_4: "⏳" }
      next_phase: sniper

  - name: user_declines
    given:
      gate_response: "no"
    expect:
      emit: "[Strategist] phase=approval_gate status=plan_only"
      next_phase: adr_opportunity
      mission_result_status: plan_only

  - name: user_declines_alternate_phrasing
    given:
      gate_response: "decline"
    expect:
      emit: "[Strategist] phase=approval_gate status=plan_only"
```

**Step 2: Create approval-gate.yaml** (migrating §6 from SKILL.md)

Create `.strategist/contracts/approval-gate.yaml`:

```yaml
module: approval-gate
type: gate
description: >
  Mandatory gate between Archivist and Sniper. User must explicitly approve
  before any execution occurs. Declining results in plan_only mission status.

contract:
  input:
    - name: artifact_path
      type: path
      required: true
      description: Path to the refined plan artifact to present

  gate:
    emit_prompt_key: approval_prompt
    stop: true
    wait_for_response: true

  responses:
    approve_aliases: [yes, approve, authorize, sim, autorizar]
    decline_aliases:  [no, decline, stop, nao, cancelar]

  on_approve:
    emit: "[Strategist] phase=approval_gate status=approved"
    checkpoint_update: { step_3_icon: "✅", step_4_icon: "⏳" }
    next_phase: sniper

  on_decline:
    emit: "[Strategist] phase=approval_gate status=plan_only"
    mission_result_status: plan_only
    next_phase: adr_opportunity

  write_scope: read-only
  owner: internal (orchestrator)
```

**Step 3: Self-verify** — read both files, confirm all invariants hold.

---

## Task 7: New Contract — adr.yaml

**Files:**
- Create: `.strategist/contracts/adr.yaml`
- Create: `.strategist/contracts/tests/adr.test.yaml`

**Step 1: Write the test first**

Create `.strategist/contracts/tests/adr.test.yaml`:

```yaml
subject: contracts/adr.yaml
version: "1.0"

invariants:
  - id: skip_when_disabled
    description: Entire ADR stage skipped when adr_enabled=false
    rule: active.adr_enabled=false → skip §8, proceed directly to §9

  - id: two_gate_flow
    description: ADR requires two explicit gates — generate and approve
    rule: gate_1 (generate?) must precede gate_2 (approve content?)

  - id: rollback_on_gate2_decline
    description: Declining gate 2 removes the file
    rule: gate_2=no → delete artifact, mission_result.adr absent

  - id: language_from_docs
    description: ADR content language follows active.language.docs
    rule: docs=pt-BR → Portuguese sections; docs=en → English sections

cases:
  - name: adr_disabled
    given:
      adr_enabled: false
    expect:
      action: skip entirely
      next_phase: learning_phase

  - name: no_criteria_met
    given:
      criteria_matched: []
    expect:
      action: skip to learning_phase

  - name: gate1_declined
    given:
      criteria_matched: [new_pattern]
      gate_1_response: "no"
    expect:
      action: log "ADR declined (gate 1)"
      next_phase: learning_phase

  - name: gate2_approved
    given:
      gate_1_response: "yes"
      gate_2_response: "yes"
    expect:
      action: sniper commits ADR
      mission_result.adr: present

  - name: gate2_declined
    given:
      gate_1_response: "yes"
      gate_2_response: "no"
    expect:
      action: delete artifact
      mission_result.adr: absent
      mission_result.status: completed
```

**Step 2: Create adr.yaml** (migrating §8 from SKILL.md)

Create `.strategist/contracts/adr.yaml`:

```yaml
module: adr
type: agent_phase
description: >
  Post-mission stage (conditional). Detects architectural decisions in the
  mission and offers to document them as ADRs. Runs after Sniper completes
  or at gate decline. Skipped entirely when adr_enabled=false.

contract:
  skip_condition: active.adr_enabled == false

  activation_criteria:
    - new_pattern: new interface, contract, schema, or abstraction introduced
    - breaking_change: field removed, signature changed, behavior changed
    - documented_tradeoff: tasks.md/design.md describe a choice with discarded alternatives
    - new_external_dependency: library, service, or protocol added

  flow:
    gate_1:
      emit_key: adr_opportunity
      stop: true
      on_no: log "ADR declined (gate 1)", proceed to §9
      on_yes: archivist writes draft AND presents full content in chat

    draft_format: |
      ---
      📚 **Archivist — ADR draft:**
      {full ADR content per template}
      ---
    artifact_path: "<base_path>/done/<mission_id>-adr.md"

    gate_2:
      emit_key: adr_gate
      stop: true
      on_yes: sniper commits ADR; mission_result.adr = <path>; proceed to §9
      on_no: delete artifact; mission_result.status = completed (no ADR); proceed to §9
      on_edit: accept inline edits, re-present draft, re-open gate_2

  template:
    language_source: active.language.docs
    sections:
      en:    [Context, Decision, Consequences]
      pt-BR: [Contexto, Decisão, Consequências]
    required_fields: [title, date, status, mission_id]

  write_scope: <base_path>/done/ (conditional on gate_2 approval)
  owner: internal (post-mission)
```

**Step 3: Self-verify** — read both files, confirm all invariants hold.

---

## Task 8: New Contract — compliance-summary.yaml

**Files:**
- Create: `.strategist/contracts/compliance-summary.yaml`
- Create: `.strategist/contracts/tests/compliance-summary.test.yaml`

**Step 1: Write the test first**

Create `.strategist/contracts/tests/compliance-summary.test.yaml`:

```yaml
subject: contracts/compliance-summary.yaml
version: "1.0"

invariants:
  - id: emitted_every_response
    description: Compliance summary must be the final element of every response
    rule: every pipeline termination (completed|plan_only|blocked) emits compliance_summary

  - id: always_info_level
    description: Compliance summary is always visible regardless of output profile
    rule: compliance_summary emit_level == INFO

  - id: non_compliant_includes_reason
    description: pipeline_compliant=no must include a reason
    rule: pipeline_compliant=no → reason field present and non-empty

cases:
  - name: full_pipeline_compliant
    given:
      phases_run: [learning_buffer, bootstrap, preflight, intake, context_enrichment, ranger, archivist, sniper]
      phases_skipped: []
    expect:
      pipeline_compliant: yes
      reason: absent

  - name: phase_skipped_with_reason
    given:
      phases_skipped: [adr_opportunity]
      skip_reason: adr_enabled=false
    expect:
      pipeline_compliant: yes
      phases_skipped: "adr_opportunity"

  - name: phase_silently_omitted
    given:
      phases_run: [bootstrap, intake, sniper]
      phases_skipped: []
    expect:
      pipeline_compliant: no
      reason: "phases ranger, archivist ran without emit"
```

**Step 2: Create compliance-summary.yaml** (migrating §10 from SKILL.md)

Create `.strategist/contracts/compliance-summary.yaml`:

```yaml
module: compliance-summary
type: mandatory_emit
description: >
  Mandatory block appended as the final element of every response, regardless
  of pipeline outcome. Always emitted at INFO level — never suppressed by output profile.

contract:
  emit_level: INFO
  emit_timing: after_all_phases (or on early termination)
  emit_format: |
    ---
    [Strategist] response_complete
      pipeline_compliant: yes | no
      phases_run: <comma-separated list>
      phases_skipped: <list or none>
      opportunity_attack: ranger=<N> archivist=<N> sniper=<N|triggered|n/a>
      treasure_chests_consulted: yes | no | none_configured
      gate_presented: yes | no | n/a

  on_non_compliant:
    include_field: reason
    format: "reason: <which phases were skipped and why>"

  invariants:
    - emitted on every response without exception
    - emit_level is always INFO (cannot be overridden by output profile)
    - pipeline_compliant=no must include reason

  write_scope: read-only
  owner: internal (orchestrator)
```

**Step 3: Self-verify** — read both files, confirm all invariants hold.

---

## Task 9: Expand bootstrap.yaml

**Files:**
- Modify: `.strategist/contracts/bootstrap.yaml`
- Create: `.strategist/contracts/tests/bootstrap.test.yaml`

**Step 1: Write bootstrap test**

Create `.strategist/contracts/tests/bootstrap.test.yaml`:

```yaml
subject: contracts/bootstrap.yaml
version: "1.0"

invariants:
  - id: output_profile_loaded_at_bootstrap
    description: Output threshold resolved from --output flag during bootstrap
    rule: bootstrap reads --output flag → loads output-profiles/profiles/<name>.yaml → stores output_threshold

  - id: fast_path_on_fresh_artifact
    description: Fast path taken when .compiled/.config.gz is fresh
    rule: check-stale exit=0 → emit bootstrap=fast_path, skip YAML reads

  - id: blocked_on_missing_slot
    description: Missing slot configuration stops the mission
    rule: active.slots absent → emit blocked reason=slots_not_configured

  - id: governance_precedence_respected
    description: user instruction > active.yaml > governance_injection > slot contracts > kernel
    rule: explicit user response always wins over any configuration

cases:
  - name: fast_path
    given:
      compiled_config_fresh: true
      output_flag: verbose
    expect:
      emit: "[Strategist] bootstrap=fast_path"
      output_threshold: DEBUG

  - name: standard_path_no_output_flag
    given:
      compiled_config_fresh: false
    expect:
      emit: "[Strategist] bootstrap=standard_path"
      output_threshold: INFO

  - name: missing_slots
    given:
      active_yaml_has_slots: false
    expect:
      emit: "[Strategist] phase=bootstrap status=blocked reason=slots_not_configured"
```

**Step 2: Expand bootstrap.yaml** — add the output profile loading procedure

Open `.strategist/contracts/bootstrap.yaml` and append to the `contract:` section after existing fields:

```yaml
  output_profile:
    description: Output verbosity resolved from --output flag at bootstrap time
    flag: --output
    valid_values: [default, verbose, full]
    default: default
    load_from: output-profiles/profiles/<flag_value>.yaml
    stores: output_threshold (level_threshold value from profile)
    fallback: if profile file missing, use INFO threshold and continue

  governance_precedence:
    order:
      1: explicit user instruction (approval gates, user responses)
      2: active.yaml (local configuration)
      3: governance_injection.* (external context, when declared in active.yaml)
      4: slot provider contracts (risk_score, write scope)
      5: embedded kernel (forbidden_behaviors, stop_conditions in skill.yaml)
    note: governance_injection does not override active.yaml — it extends it
```

**Step 3: Self-verify** — read both files, confirm all invariants hold.

---

## Task 10: Expand preflight.yaml

**Files:**
- Modify: `.strategist/contracts/preflight.yaml`
- Create: `.strategist/contracts/tests/preflight.test.yaml`

**Step 1: Write preflight test**

Create `.strategist/contracts/tests/preflight.test.yaml`:

```yaml
subject: contracts/preflight.yaml
version: "1.0"

invariants:
  - id: slot_resolution_order
    description: Provider resolution uses declared order — skill_root first
    rule: resolve provider in order: skill_root/ → .claude/skills/ → registry

  - id: risk_mismatch_blocks
    description: Slot with wrong risk_score stops the mission
    rule: slot.risk_score != expected → emit blocked reason=slot_risk_mismatch

  - id: governance_mode_declared
    description: Governance mode always declared in preflight done event
    rule: emit phase=preflight status=done governance=GOVERNED|COMPATIBLE

cases:
  - name: governed_mode
    given:
      governance_injection_present: true
      slots_valid: true
    expect:
      emit: "[Strategist] phase=preflight status=done slots=ok governance=GOVERNED"

  - name: compatible_mode
    given:
      governance_injection_present: false
      slots_valid: true
    expect:
      emit: "[Strategist] phase=preflight status=done slots=ok governance=COMPATIBLE"

  - name: risk_mismatch
    given:
      ranger_risk_score: controlled
    expect:
      emit: "[Strategist] phase=preflight status=blocked reason=slot_risk_mismatch slot=discovery"
```

**Step 2: Expand preflight.yaml** — append slot risk contract details and the full resolution order to existing content.

Read the current file first, then add:

```yaml
  slot_risk_contracts:
    ranger:
      expected_risk: write_pending
      authorized_write: ".md files in <base_path>/pending/"
      violation: any write outside that scope or non-.md type
    archivist:
      expected_risk: write_analysis
      authorized_write: ".md files in <base_path>/ and <base_path>/refined/"
      violation: any write outside <base_path>/ or non-.md type
    sniper:
      expected_risk: controlled
      requirement: approval gate required before any execution

  provider_resolution_order:
    1: "<skill_root>/<provider>/skill.yaml"
    2: ".claude/skills/<provider>/skill.yaml"
    3: "skill registry entry skill_yaml path (if registry present)"
    special: _runtime_provider → resolve from governance_injection.execution_provider
```

**Step 3: Self-verify** — read both files, confirm all invariants hold.

---

## Task 11: Expand context-enrichment.yaml, learning-buffer.yaml, learning-curator.yaml

**Files:**
- Modify: `.strategist/contracts/context-enrichment.yaml`
- Modify: `.strategist/contracts/learning-buffer.yaml`
- Modify: `.strategist/contracts/learning-curator.yaml`
- Create: `.strategist/contracts/tests/context-enrichment.test.yaml`
- Create: `.strategist/contracts/tests/learning-buffer.test.yaml`
- Create: `.strategist/contracts/tests/learning-curator.test.yaml`

**Step 1: Write context-enrichment test**

Create `.strategist/contracts/tests/context-enrichment.test.yaml`:

```yaml
subject: contracts/context-enrichment.yaml
version: "1.0"

invariants:
  - id: fast_path_on_fresh_index
    description: Compiled index used when fresh, skipping linear scan
    rule: check-stale .compiled/.index.gz exit=0 → emit context_enrichment=fast_path

  - id: empty_enrichment_non_blocking
    description: No matching sources is not an error
    rule: enrichment returns empty → dossier contains only task_type and output_template, pipeline continues

  - id: treasure_chest_signal_emitted
    description: Chest consultation triggers treasure_chest_found emit
    rule: chest consulted → emit treasure_chest_found chest_id=<id>

cases:
  - name: fast_path_with_matches
    given:
      index_fresh: true
      task_type: architecture_analysis
    expect:
      emit: "[Strategist] context_enrichment=fast_path"

  - name: no_matching_sources
    given:
      task_type: unknown_type
    expect:
      action: return empty dossier
      pipeline: continues
```

**Step 2: Write learning-buffer test**

Create `.strategist/contracts/tests/learning-buffer.test.yaml`:

```yaml
subject: contracts/learning-buffer.yaml
version: "1.0"

invariants:
  - id: absent_file_is_skip_not_error
    description: Missing outcomes.tmp is normal for first run
    rule: outcomes.tmp absent → status=skipped reason=buffer_empty, continue

  - id: flush_threshold_is_20
    description: Flush occurs only when buffer reaches 20 lines
    rule: line_count >= 20 → flush; line_count < 20 → skip

  - id: flush_does_not_block
    description: Flush failure is non-blocking
    rule: flush_failure → log to stderr, continue mission

cases:
  - name: absent_file
    given:
      outcomes_tmp_exists: false
    expect:
      emit: "[Strategist] learning_buffer=skipped reason=buffer_empty"

  - name: below_threshold
    given:
      line_count: 5
    expect:
      action: skip flush

  - name: at_threshold
    given:
      line_count: 20
    expect:
      action: flush to outcomes.jsonl
      emit: "[Strategist] learning_buffer=flushed count=20"
```

**Step 3: Write learning-curator test**

Create `.strategist/contracts/tests/learning-curator.test.yaml`:

```yaml
subject: contracts/learning-curator.yaml
version: "1.0"

invariants:
  - id: checkpoint_before_write
    description: Learning curator must present checkpoint before writing anything
    rule: show checkpoint → wait for user → then write

  - id: failure_non_blocking
    description: Learning phase failure never affects mission result
    rule: learning_phase_fails → log failure, mission_result unchanged

  - id: outcome_always_appended
    description: Even on failure, append outcome to buffer
    rule: after learning_curator completes or fails → append to outcomes.tmp

cases:
  - name: curator_success
    given:
      critic_evaluation: present
      mission_result: completed
    expect:
      action: present checkpoint, wait, write
      outcomes_tmp: appended

  - name: curator_timeout
    given:
      learning_phase: timeout
    expect:
      action: log failure
      mission_result: unchanged
      outcomes_tmp: appended (with failure note)
```

**Step 4: Expand the three contract files** — append the new invariant details to each, matching what the tests declare.

**Step 5: Self-verify each** — read test + contract pairs, confirm invariants hold.

---

## Task 12: Expand Roles — ranger.yaml, archivist.yaml, sniper.yaml

**Files:**
- Modify: `.strategist/roles/ranger.yaml`
- Modify: `.strategist/roles/archivist.yaml`
- Modify: `.strategist/roles/sniper.yaml`
- Create: `.strategist/contracts/tests/ranger.test.yaml`
- Create: `.strategist/contracts/tests/archivist.test.yaml`
- Create: `.strategist/contracts/tests/sniper.test.yaml`

**Step 1: Write ranger test**

Create `.strategist/contracts/tests/ranger.test.yaml`:

```yaml
subject: roles/ranger.yaml
version: "1.0"

invariants:
  - id: opportunity_attack_mandatory
    description: Ranger always runs opportunity attack
    rule: ranger_done → opportunity_attack=done items=<N> emitted

  - id: no_strategy_decisions
    description: Ranger surfaces items only, does not decide strategy for side_quests
    rule: ranger outputs discovery artifact and opportunity manifest only

  - id: emit_on_completion
    description: Ranger emits done event with artifact path
    rule: ranger_done → emit via ranger_done persona key with artifact_path

cases:
  - name: normal_run_no_items
    given:
      opportunity_items: 0
    expect:
      emit: "[Strategist] phase=ranger opportunity_attack=done items=0"
      artifact: "<base_path>/pending/<mission_id>-discovery.md"

  - name: items_found
    given:
      opportunity_items: 3
    expect:
      emit: "[Strategist] phase=ranger opportunity_attack=done items=3"
      action: present items to user via opportunity_detected persona key
```

**Step 2: Write archivist test**

Create `.strategist/contracts/tests/archivist.test.yaml`:

```yaml
subject: roles/archivist.yaml
version: "1.0"

invariants:
  - id: opportunity_attack_mandatory
    description: Archivist always runs opportunity attack (side_quest detection)
    rule: archivist_done → opportunity_attack=done side_quests=<N> emitted

  - id: side_quest_gate_when_n_gt_0
    description: Non-zero side quests trigger a gate before proceeding
    rule: side_quests > 0 → present gate via side_quest_detected persona key

  - id: emit_on_completion
    description: Archivist emits done event with artifact path
    rule: archivist_done → emit via archivist_done persona key with artifact_path

cases:
  - name: no_side_quests
    given:
      side_quests: 0
    expect:
      emit: "[Strategist] phase=archivist opportunity_attack=done side_quests=0"

  - name: side_quest_detected
    given:
      side_quests: 1
    expect:
      emit: "[Strategist] phase=archivist opportunity_attack=done side_quests=1"
      action: present side_quest gate
```

**Step 3: Write sniper test**

Create `.strategist/contracts/tests/sniper.test.yaml`:

```yaml
subject: roles/sniper.yaml
version: "1.0"

invariants:
  - id: opportunity_attack_mandatory
    description: Sniper always runs opportunity attack during execution
    rule: if side_quest surfaces mid-execution → emit opportunity_attack=triggered items=<N>

  - id: task_progress_events
    description: Sniper emits per-task progress events
    rule: task=<N> status=running → re-emit full task list with N marked ⏳

  - id: no_document_authoring
    description: Sniper does not create specs, plans, or analysis documents
    rule: document authoring → stop, return to Archivist (contract: write_analysis)

cases:
  - name: clean_execution_no_side_quests
    given:
      side_quests_during_execution: 0
    expect:
      emit: "[Strategist] phase=sniper opportunity_attack=done items=0"

  - name: side_quest_mid_execution
    given:
      side_quest_detected: true
      items: 2
    expect:
      emit: "[Strategist] phase=sniper opportunity_attack=triggered items=2"
      action: present gate before continuing
```

**Step 4: Expand each role YAML** — add `phase_procedure` block with the full behavior from §5a (Ranger), §5e (Archivist), §7 (Sniper) in SKILL.md.

**Step 5: Self-verify each** — read test + role pairs, confirm invariants hold.

---

## Task 13: Create mission-result.schema.yaml

**Files:**
- Create: `.strategist/schemas/mission-result.schema.yaml`

**Step 1: Create the schema** (migrating §11 from SKILL.md)

Create `.strategist/schemas/mission-result.schema.yaml`:

```yaml
schema: mission-result
version: "1.0"
description: Shape of the object returned at the end of every Strategist mission.

fields:
  mission_id:
    type: string
    required: true

  status:
    type: enum
    values: [completed, plan_only, blocked]
    required: true

  artifacts:
    type: object
    fields:
      discovery:
        type: path
        present_when: Ranger ran
      opportunity_report:
        type: string
        value: inline
        present_when: opportunity execution ran
      refined_plan:
        type: path
        present_when: Archivist ran
      execution_report:
        type: path
        present_when: Sniper ran
      adr:
        type: path
        present_when: ADR generated and committed

  blockers:
    type: array
    items: string (blocker codes)
    default: []
```

**Step 2: Verify schema file is well-formed YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('.strategist/schemas/mission-result.schema.yaml'))" && echo "valid"
```
Expected: `valid`

---

## Task 14: Rewrite SKILL.md as Pipeline Manifest

**Files:**
- Modify: `.strategist/SKILL.md`

This is the final task. All contracts and roles are now expanded and tested. SKILL.md is reduced to a routing document.

**Step 1: Read the current SKILL.md in full**

Read `.strategist/SKILL.md` (715 lines) to confirm all sections have been migrated to contracts/roles before touching this file.

**Step 2: Verify all contracts exist**

```bash
ls .strategist/contracts/*.yaml .strategist/roles/*.yaml .strategist/output-profiles/emit-taxonomy.yaml .strategist/schemas/mission-result.schema.yaml
```

Expected: all 20 content files present (no errors).

**Step 3: Rewrite SKILL.md**

Replace the full content with the lean manifest (~200 lines). Structure:

```markdown
## ⚠️ MANDATORY — BEFORE ANY RESPONSE
[keep as-is — 15 lines, critical compliance header]

# Strategist — Agent Instructions
[keep identity table — 10 lines]

## Output Verbosity
> Contract: output-profiles/emit-taxonomy.yaml
Flag --output=default|verbose|full. Threshold loaded at bootstrap. Before each emit: check level >= threshold.
OTEL export is never filtered.

## §0 Pre-Bootstrap
> Contract: contracts/learning-buffer.yaml
Check and flush if needed. On absent file: continue.

## §1 Bootstrap
> Contract: contracts/bootstrap.yaml
Load config, persona, slots, output profile. On failure: emit blocked.

## §2 Preflight
> Contract: contracts/preflight.yaml
Resolve slots, validate risk contracts, declare governance mode.

## §3 Intake
> Contract: contracts/intake.yaml
Classify prompt, generate mission_id, emit checkpoint. Route quick_draw if triggered.

## §4 Context Enrichment
> Contract: contracts/context-enrichment.yaml
Query knowledge index, assemble dossier.

## §5 Mission Phases
Pipeline: Ranger → Archivist → §6 Approval Gate → §7 Sniper

Opportunity Attack is mandatory inside each role.
> Contract: contracts/opportunity-attack.yaml

### §5a Ranger
> Role: roles/ranger.yaml
> Treasure chests: filter scope=discovery|all

### §5e Archivist
> Role: roles/archivist.yaml
> Treasure chests: filter scope=refinement|all

## §6 Approval Gate (MANDATORY)
> Contract: contracts/approval-gate.yaml
STOP. Present plan. Wait for yes/no. No Sniper without approval.

## §7 Sniper
> Role: roles/sniper.yaml
> Treasure chests: filter scope=execution|all

## §8 ADR Opportunity (conditional)
> Contract: contracts/adr.yaml
Skip if adr_enabled=false. Two-gate flow. Language from active.language.docs.

## §9 Learning Phase (non-blocking)
> Contracts: contracts/learning-curator.yaml, contracts/learning-buffer.yaml
Non-blocking. Failure never affects mission result.

## §10 Compliance Summary (mandatory — every response)
> Contract: contracts/compliance-summary.yaml
Always emitted. Always INFO level.

## §11 Mission Result
> Schema: schemas/mission-result.schema.yaml

## Footprint Rule
[keep as-is — 12 lines]

## Drift Self-Correction
[keep as-is — 12 lines]
```

**Step 4: Verify line count**

```bash
wc -l .strategist/SKILL.md
```

Expected: < 250 lines.

**Step 5: Spot-check behavioral completeness**

Read the new SKILL.md and verify:
- Every phase has a contract/role reference
- Opportunity Attack invariant is present
- All gates (quick_draw, approval, ADR) are referenced
- Footprint Rule and Drift Self-Correction are intact

---

## Summary

| Task | Files | Type |
|------|-------|------|
| 1 | tests/ directory | setup |
| 2 | emit-taxonomy + profiles + test | new |
| 3 | intake.yaml + test | new |
| 4 | opportunity-attack.yaml + test | new |
| 5 | quick-draw.yaml + test | new |
| 6 | approval-gate.yaml + test | new |
| 7 | adr.yaml + test | new |
| 8 | compliance-summary.yaml + test | new |
| 9 | bootstrap.yaml expanded + test | expand |
| 10 | preflight.yaml expanded + test | expand |
| 11 | context-enrichment + learning-buffer + learning-curator + tests | expand |
| 12 | ranger + archivist + sniper + tests | expand |
| 13 | mission-result.schema.yaml | new |
| 14 | SKILL.md rewrite | rewrite |

**Total:** 34 files. Zero behavioral change. SKILL.md: 715 → ~200 lines.
