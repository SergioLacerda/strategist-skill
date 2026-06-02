# Discovery Brief — {{mission_id}}

> **Format note:** This is a reference template for implementing skills.
> Required handoff contract fields: see `schemas/handoff-ranger-to-archivist.schema.yaml`.
> The implementing skill decides the actual file format and layout.

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
| SQ-1 | {{description}} | {{related/unrelated/duplicate}} | {{strategy}} |

## Recommended Refinement Focus

- {{focus_1}}
- {{focus_2}}
