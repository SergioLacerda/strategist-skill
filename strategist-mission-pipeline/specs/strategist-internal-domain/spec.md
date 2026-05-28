## ADDED Requirements

### Requirement: Internal domain lives at <base_path>/.strategist/
The Strategist internal domain SHALL exist at `<base_path>/.strategist/` where `<base_path>` is: the value injected by SDD (`sdd_injection.base_path`) when SDD-integrated, or the value configured at init (`roles/default.yaml` base field) in standalone mode. Default standalone base is `.analysis`, so default domain path is `.analysis/.strategist/`. It MUST contain `index.yaml` and the `identity/` directory with `what-i-am.yaml` and `drift-patterns.yaml`.

```
<base_path>/
├── todo/
├── pending/
├── refined/
├── done/
└── .strategist/          ← internal domain always inside base_path
    ├── index.yaml
    ├── identity/
    │   ├── what-i-am.yaml
    │   └── drift-patterns.yaml
    ├── directives/
    ├── rubrics/
    ├── patterns/
    ├── memory/
    └── knowledge/        ← user-provided or auto-created knowledge base
```

#### Scenario: Domain initialized (standalone)
- **WHEN** `.analysis/.strategist/index.yaml` exists (after `strategist init`)
- **THEN** Strategist preflight succeeds in loading the domain

#### Scenario: Domain initialized (SDD integration)
- **WHEN** `.sdd/analysis/.strategist/index.yaml` exists (after SDD injects base_path)
- **THEN** Strategist preflight succeeds in loading the domain

#### Scenario: Domain missing
- **WHEN** `<base_path>/.strategist/index.yaml` does not exist
- **THEN** Strategist operates without internal domain (no enrichment, no self-correction patterns)

---

### Requirement: index.yaml controls selective file loading
`index.yaml` SHALL declare `load_always` (loaded every mission), `load_by_task_type` (loaded per task_type), and `load_on_demand` (loaded only when needed) sections. Strategist MUST NOT load files not referenced in the index.

#### Scenario: Selective loading by task_type
- **WHEN** `task_type=refactor` and index has `refactor` entry in `load_by_task_type`
- **THEN** only the directive and rubric files listed under `refactor` are loaded, nothing else

#### Scenario: Unknown task_type
- **WHEN** task_type is not present in `load_by_task_type`
- **THEN** only `load_always` files are loaded

---

### Requirement: what-i-am.yaml declares identity invariants
`identity/what-i-am.yaml` SHALL declare `i_am`, `i_am_not`, and `core_invariants` lists. These are loaded at every preflight and used to anchor the agent's behavioral boundaries.

#### Scenario: Identity loaded at preflight
- **WHEN** Strategist starts a mission
- **THEN** `what-i-am.yaml` is loaded and its `core_invariants` are active for the duration

---

### Requirement: drift-patterns.yaml enables self-correction
`identity/drift-patterns.yaml` SHALL list known drift behaviors with `symptom` and `correction` fields. The agent SHALL check for matching symptoms and apply corrections without consulting `.sdd/` governance files.

#### Scenario: Drift detected and corrected
- **WHEN** agent is about to perform discovery work directly instead of delegating to scout slot
- **THEN** drift pattern `direct_execution` matches and correction "Stop. Identify active slot. Invoke provider. Resume." is applied

#### Scenario: Approval bypass drift detected
- **WHEN** agent is about to invoke execution slot without user approval
- **THEN** drift pattern `approval_bypass` matches and execution is stopped

---

### Requirement: memory/lessons.yaml stores human-curated lessons
`memory/lessons.yaml` SHALL store lessons derived from past missions. Each lesson MUST have `id`, `task_type`, `lesson`, and `source` fields. This file is human-curated — the agent only reads it, never writes to it directly.

#### Scenario: Lesson loaded for matching task_type
- **WHEN** `task_type=architecture_analysis` and a lesson exists for this type
- **THEN** lesson is included in the dossier for the current mission

#### Scenario: No lessons for task_type
- **WHEN** no lessons exist for current task_type
- **THEN** mission continues without lessons — no error
