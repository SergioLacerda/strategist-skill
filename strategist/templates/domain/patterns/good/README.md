# Good Patterns

Place examples of high-quality slot outputs here. Each example is a markdown file.

## File naming convention

`<task_type>-<short-description>.md`

Examples:
- `architecture-analysis-clean-modules-index.md`
- `refactor-well-scoped-tasks.md`

## Required frontmatter

Each example file must begin with:

```yaml
---
task_type: <string>
label: <short description>
source_mission: <mission_id or "manual">
why_good: <one sentence>
---
```

## Loading

Good patterns are loaded `on_demand` via `index.yaml`. They are included in the dossier
when `dossier-builder` determines they are relevant to the current task_type and token budget.
