# SCOPE LOCKING
id: scope-locking
severity: medium

Every change declares scope before starting. Expansions during documentation materialization
require a pause and a new approval — never executed inline.

## Rules

- Sniper materializes only what is listed in the `tasks.md` approved at the approval gate
- Any file outside the declared scope requires a pause + mini approval
- Opportunity improvements discovered during materialization go to a new item
  in `<base_path>/todo/`, not executed in the same Sniper run
- "While I'm here I'll also..." is scope expansion — requires gate
- Adjacent refactors to approved scope are scope expansion — requires gate

## When to Pause

Sniper must pause and signal Strategist when:
- A file not listed in `tasks.md` would need to be modified to complete the task
- A task reveals a dependency not mapped in the design
- Materialization would require changing a public contract

## Enforcement

Strategist validates `tasks.md` before invoking Sniper.
The approval gate includes an explicit warning when `tasks.md` contains writes outside `<base_path>/`.
response-critic signals scope drift detected after materialization.
