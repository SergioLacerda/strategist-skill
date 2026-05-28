## ADDED Requirements

### Requirement: Strategist extracts intake constraints from user prompt
Before invoking any slot, the Strategist MUST extract three constraint fields from the user prompt: `delivery_strategy`, `legacy_compatibility`, and `execution_intent`. These become `mission_contract.planning_rules` passed to all slot providers.

#### Scenario: Constraints extracted from prompt
- **WHEN** user prompt contains "sem prazo, sem retrocompatibilidade"
- **THEN** intake extracts `delivery_strategy=total` and `legacy_compatibility=not_required`

#### Scenario: Constraint absent — default applied
- **WHEN** user prompt contains no delivery strategy indicator
- **THEN** `delivery_strategy` defaults to `sprint` without asking the user

---

### Requirement: Constraint extraction uses confidence threshold
A constraint SHALL be applied without asking the user only when LLM-assessed confidence is ≥ 0.65. Below threshold, the default is used unless the missing field blocks the mission.

#### Scenario: High confidence extraction
- **WHEN** prompt contains "big bang" and confidence is 0.87
- **THEN** `delivery_strategy=total` is applied without clarification

#### Scenario: Conflicting constraints detected
- **WHEN** prompt contains both "por sprint" and "entrega total"
- **THEN** Strategist stops and asks user to resolve the conflict before proceeding

---

### Requirement: task_type is classified separately from intake constraints
`task_type` (e.g., `architecture_analysis`, `refactor`) SHALL be derived by `prompt-intake` from the user prompt independently of the intake constraint fields. It drives selective loading in `.strategist/index.yaml`.

#### Scenario: task_type classified
- **WHEN** user prompt is "analisar boundary entre módulos de autenticação"
- **THEN** `task_type=architecture_analysis` is derived and used to load `load_by_task_type` entries from index.yaml

#### Scenario: task_type unrecognized
- **WHEN** user prompt does not match any known task_type pattern
- **THEN** `task_type=general` is applied and only `load_always` files are loaded

---

### Requirement: intake.schema.yaml defines recognized aliases
`.sdd/skills/strategist/intake.schema.yaml` SHALL declare all recognized aliases for each constraint value, enabling deterministic alias matching.

#### Scenario: Alias match for delivery_strategy
- **WHEN** prompt contains "faseado" (alias for sprint)
- **THEN** `delivery_strategy=sprint` is extracted via alias match

#### Scenario: New alias not in schema
- **WHEN** prompt contains an expression not listed in intake.schema.yaml
- **THEN** constraint is treated as absent and default is applied
