## ADDED Requirements

### Requirement: Strategist init configures base_path, slot providers, and knowledge base
On first use (no `roles/default.yaml` present), the Strategist MUST prompt the user for: base directory (default `.analysis`), Scout provider (default `sdd-diagnose`), Engineer provider (default `sdd-engineer`), Hunter provider (default: ask user), and knowledge base path (existing path or create new at `<base_path>/.strategist/knowledge/`). After confirmation, Strategist MUST create `<base_path>/todo/`, `pending/`, `refined/`, `done/`, `.strategist/` and write `roles/default.yaml`.

#### Scenario: First-use init
- **WHEN** `roles/default.yaml` does not exist and Strategist is invoked
- **THEN** init prompts are shown, base structure is created, and `roles/default.yaml` is written

#### Scenario: Init with existing knowledge path
- **WHEN** user provides a valid path at knowledge base prompt
- **THEN** `roles/default.yaml` records `knowledge_paths: [<user_path>]` and Strategist loads from it

#### Scenario: Init creates new knowledge base
- **WHEN** user presses Enter at knowledge base prompt (no path)
- **THEN** `<base_path>/.strategist/knowledge/` is initialized with empty template files and recorded as knowledge path

---

### Requirement: Strategist is optionally registered as an SDD analysis plugin
When SDD's `sdd-analysis-plugin-protocol` is deployed, the Strategist MAY be registered in `.sdd/plugins/registry.yaml` as type `analysis_orchestrator`. This is optional and additive — Strategist functions standalone without this registration.

#### Scenario: SDD integration active
- **WHEN** `.sdd/plugins/registry.yaml` has `id: strategist` with `status: active`
- **THEN** `sdd_injection` overrides Hunter slot and appends knowledge_paths from SDD

#### Scenario: Standalone mode (no SDD registry)
- **WHEN** no plugin registry is present
- **THEN** Strategist uses `roles/default.yaml` config entirely; no SDD injection occurs

---

### Requirement: Strategist runs preflight before pipeline starts
Before invoking any slot, the Strategist MUST load `<base_path>/.strategist/index.yaml`, load `load_always` files, resolve all slot providers from `roles/<config>.yaml`, and validate each provider's `skill.yaml` against the slot risk contract.

#### Scenario: Preflight passes
- **WHEN** all slot providers are found and their risk_score matches requirements
- **THEN** `[Strategist] phase=preflight status=done slots=ok` is emitted and pipeline continues

#### Scenario: Preflight fails — provider not found
- **WHEN** a declared slot provider's skill.yaml cannot be located
- **THEN** `[Strategist] phase=preflight status=blocked reason=slot_provider_not_found` is emitted and pipeline stops

#### Scenario: Preflight fails — risk mismatch
- **WHEN** Scout or Engineer slot provider has `risk_score` other than `read_only`
- **THEN** `[Strategist] phase=preflight status=blocked reason=slot_risk_mismatch` is emitted and pipeline stops

---

### Requirement: Strategist emits progress events on every phase transition
The Strategist MUST emit a structured `[Strategist]` progress event at the start and end of every phase, and on any blocker. Events MUST use public role names (Scout/Engineer/Hunter) and include phase, status, skill, and checklist fields.

#### Scenario: Phase start event
- **WHEN** Strategist begins a phase
- **THEN** event format is `[Strategist] phase=<scout|engineer|hunter> status=running skill=<provider> checklist=<n>/<total>`

#### Scenario: Phase completion event
- **WHEN** a phase completes successfully
- **THEN** event includes `status=done` and `artifact=<path>`

#### Scenario: Blocker event
- **WHEN** a phase cannot continue
- **THEN** event includes `status=blocked`, `reason=<code>`, and `action=<resolution>`

---

### Requirement: Approval gate is mandatory before Hunter slot
The Strategist MUST stop after the Engineer slot completes, present the refined plan path to the user, and explicitly ask for execution approval before invoking the Hunter slot.

#### Scenario: User approves execution
- **WHEN** user responds affirmatively to approval prompt
- **THEN** Strategist invokes Hunter slot with the reviewed plan

#### Scenario: User denies execution
- **WHEN** user declines execution
- **THEN** Strategist stops with `status=plan_only` and returns refined plan path

#### Scenario: Approval gate skipped
- **WHEN** Strategist attempts to invoke Hunter slot without user approval
- **THEN** this is a forbidden behavior (constitutes M017 violation when SDD-integrated; internal contract violation in standalone mode)

---

### Requirement: Strategist returns a valid mission result to SDD
After pipeline completion or stop, the Strategist MUST return a mission result conforming to `mission-result.schema.yaml` with mission_id, status, artifact paths, and any blockers.

#### Scenario: Completed mission result
- **WHEN** all phases complete including execution
- **THEN** mission result has `status: completed` and artifact paths for discovery, refined_plan, and execution_report

#### Scenario: Plan-only mission result
- **WHEN** execution is declined
- **THEN** mission result has `status: plan_only` and artifact paths for discovery and refined_plan
