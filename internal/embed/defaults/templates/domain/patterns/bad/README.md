# Bad Patterns

Place examples of low-quality or problematic slot outputs here — anti-patterns to avoid.
Each example is a markdown file.

## File naming convention

`<task_type>-<short-description>.md`

Examples:
- `architecture-analysis-speculative-claims.md`
- `refactor-scope-explosion.md`

## Required frontmatter

Each example file must begin with:

```yaml
---
task_type: <string>
label: <short description>
source_mission: <mission_id or "manual">
why_bad: <one sentence>
correction: <what the correct approach would have been>
---
```

## Loading

Bad patterns are loaded `on_demand` via `index.yaml`. At most 1 bad example is included
per mission dossier (the most relevant by label match), to avoid anchoring on failures.
