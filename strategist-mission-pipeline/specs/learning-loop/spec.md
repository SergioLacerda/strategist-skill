## ADDED Requirements

### Requirement: prompt-intake classifies task_type and risk before pipeline
The `prompt-intake` skill SHALL analyze the user prompt and produce `task_type`, `risk_level`, and detected intake constraints as a structured output before the pipeline starts.

#### Scenario: Intent classified
- **WHEN** user prompt is processed by prompt-intake
- **THEN** output contains `task_type`, `risk_level`, and `constraints` fields

#### Scenario: Classification failure
- **WHEN** prompt-intake cannot determine task_type with sufficient confidence
- **THEN** `task_type=general` is returned and mission proceeds without task-specific enrichment

---

### Requirement: context-enrichment retrieves relevant knowledge from configured paths
The `context-enrichment` skill SHALL search all configured `knowledge_paths` for rubrics, examples (good and bad), directives, and lessons matching the current `task_type`. Results are merged and deduplicated by `id`.

#### Scenario: Relevant examples found
- **WHEN** task_type matches examples in knowledge_paths
- **THEN** at most 2 good examples and 1 bad example are returned (most relevant by label match)

#### Scenario: No knowledge paths configured
- **WHEN** knowledge_paths is empty or all paths are missing
- **THEN** enrichment returns empty result — pipeline continues without enrichment, no error

---

### Requirement: dossier-builder produces a minimal dossier for slot providers
The `dossier-builder` skill SHALL construct a minimal dossier containing task_type, applicable directives, matched examples (good and bad), output template, and rubric. The dossier MUST NOT include the full knowledge base.

#### Scenario: Minimal dossier built
- **WHEN** dossier-builder receives enrichment output
- **THEN** dossier contains only the sections relevant to current task_type, within token budget

#### Scenario: No enrichment available
- **WHEN** context-enrichment returns empty
- **THEN** dossier-builder produces a minimal dossier with only task_type and output template

---

### Requirement: response-critic evaluates slot output against rubric
The `response-critic` skill SHALL evaluate the output of a slot provider against the rubric for the current task_type. Evaluation checks `must_have` items (present) and `must_not` items (absent).

#### Scenario: Output passes rubric
- **WHEN** slot output contains all `must_have` items and no `must_not` items
- **THEN** response-critic returns `pass` with score ≥ threshold

#### Scenario: Output fails rubric
- **WHEN** slot output is missing a `must_have` item
- **THEN** response-critic returns `fail` with list of missing items

---

### Requirement: learning-curator requires explicit human approval before writing memory
The `learning-curator` skill SHALL present a learning checkpoint to the user after each mission. It MUST NOT write to `memory/outcomes.jsonl` or update `memory/lessons.yaml` without the user's explicit approval at the checkpoint.

#### Scenario: User approves recording
- **WHEN** user selects "accepted" and "good example" at learning checkpoint
- **THEN** learning-curator appends outcome to `outcomes.jsonl` and optionally adds a lesson to `lessons.yaml`

#### Scenario: User declines recording
- **WHEN** user selects "do not record" at learning checkpoint
- **THEN** learning-curator writes nothing to memory and mission result is unaffected

#### Scenario: Learning loop failure does not block mission
- **WHEN** any Learning Loop skill fails or times out
- **THEN** mission result is returned to user without modification; learning failure is logged but not surfaced as a mission error
