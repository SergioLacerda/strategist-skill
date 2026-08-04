---
mission_id: 20260101-example-fixture
mission_status: ranger_pending
date: 2026-01-01
---

# Analysis — Example Fixture (D1 Golden Fixture)

This is a hand-authored golden fixture used by
`TestD1_RangerArtifactShapeValid` (see
`.analysis/archived/20260804-eval-fake-provider-adr.md` DEC-2). It is not a
live Ranger output — it exists to validate the shape a real Ranger artifact
must have: correct frontmatter and all seven required sections declared in
`internal_skills/ranger/SKILL.md`.

## Mission Objective

Demonstrate the required frontmatter and section shape of a Ranger discovery
artifact, for fixture-based Phase 2 content assertions.

## Known Facts

- Ranger's required section list is defined in
  `.strategist/internal_skills/ranger/SKILL.md`.
- The canonical handoff field contract is
  `.strategist/schemas/handoff-ranger-to-archivist.schema.yaml`.

## Uncertainties

None — this is a fixture, not a real discovery pass.

## Affected Scope

`internal/eval/**`, `tests/evals/fixtures/**`.

## Side Quests

None.

## Scope Observations

None.

## Recommended Refinement Focus

Not applicable — fixture only.
