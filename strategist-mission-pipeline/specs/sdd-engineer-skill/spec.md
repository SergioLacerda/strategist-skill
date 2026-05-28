## ADDED Requirements

### Requirement: sdd-engineer is a read-only skill in the SDD skill registry
`sdd-engineer` SHALL be registered in `.sdd/skills/registry.json` with `risk_score: read_only`, `status: active`, and `category: refinement`. It MUST NOT modify any file — it only reads input artifacts and produces output artifacts.

#### Scenario: sdd-engineer registered
- **WHEN** skill registry is loaded
- **THEN** `sdd-engineer` entry exists with `risk_score: read_only` and `status: active`

---

### Requirement: sdd-engineer requires a discovery artifact as input
`sdd-engineer` SHALL accept a discovery artifact path as its primary input. It MUST NOT run without a valid discovery artifact present at the declared input path.

#### Scenario: Discovery artifact present
- **WHEN** a discovery artifact exists at the input path
- **THEN** sdd-engineer reads it and begins refinement

#### Scenario: Discovery artifact missing
- **WHEN** no artifact exists at the declared input path
- **THEN** sdd-engineer stops with `reason=missing_discovery_artifact` and does not produce output

---

### Requirement: sdd-engineer produces a reviewed plan with required sections
The output artifact from `sdd-engineer` MUST contain all required sections: executive summary, tasks with subitems, technical details, modules/documents index, design (context, goals, non-goals, Do, Do Not), execution checklist, and Hunter instructions.

#### Scenario: Complete reviewed plan produced
- **WHEN** sdd-engineer successfully processes a discovery artifact
- **THEN** output markdown contains all required sections with non-empty content

#### Scenario: Missing section detected
- **WHEN** sdd-engineer cannot produce a required section due to insufficient evidence
- **THEN** the section is marked `[INSUFFICIENT EVIDENCE]` and a blocker is added to the output

---

### Requirement: sdd-engineer must not invent evidence not present in discovery artifact
`sdd-engineer` SHALL only make claims in the reviewed plan that are grounded in the discovery artifact content. It MUST NOT add assumptions, invented constraints, or speculative architecture.

#### Scenario: Evidence-grounded plan
- **WHEN** discovery artifact contains module list and risk analysis
- **THEN** reviewed plan references only modules and risks present in the discovery artifact

#### Scenario: Speculation detected
- **WHEN** sdd-engineer would need to speculate to fill a required section
- **THEN** it marks the section with `[NEEDS CLARIFICATION]` and lists the specific missing information

---

### Requirement: sdd-engineer output persists to the refinement state path
The reviewed plan output MUST be written to `<base>/refined/<artifact-name>` where `<base>` is provided by the Strategist from `sdd_injection.base_path`.

#### Scenario: Artifact persisted correctly
- **WHEN** sdd-engineer completes refinement
- **THEN** output file appears at `.sdd/analysis/refined/<mission-id>-plan.md`
