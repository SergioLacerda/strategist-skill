# Strategist — ADR Contract

## 8. ADR Opportunity (post-mission, conditional)

**Skip this entire section if `active.adr_enabled` is `false`.** Proceed directly to §9.

After Sniper completes (`status=documentation_applied`) OR after approval gate (`status=analysis_delivered`, `revision_requested`, or `rejected`):

**Activation criteria — evaluate if the mission contains architectural decisions:**

| Criterion | Signal |
|-----------|--------|
| New pattern introduced | New interface, contract, schema, or abstraction |
| Breaking change (even controlled) | Field removed, signature changed, behavior changed |
| Documented trade-off | `tasks.md` / `design.md` describe a choice with discarded alternatives |
| New external dependency | Library, service, or protocol added |

If no criterion is met: skip directly to §9 (Learning Phase).

If any criterion is met:

Emit via `persona.content_by_lang[active.language.chat].side_quest_detected` with
`{description}` = `"ADR — opportunity to document architectural decision."` before presenting the gate.

Emit via `persona.content_by_lang[active.language.chat].adr_opportunity` with `{mission_id}`.

**Gate 1 — Generate draft?** STOP. Wait for response:
- **no**: Log in learning phase as "ADR declined (gate 1)". Continue to §9.
- **yes**: Archivist writes draft AND **presents the full content in chat**:
  ```markdown
  ---
  📚 **Archivist — ADR draft:**

  {full ADR content per template below}
  ---
  ```
  Artifact also written to `<base_path>/archived/<mission_id>-adr.md`.

  Emit via `persona.content_by_lang[active.language.chat].adr_gate` with `{draft_content}`.

  **Gate 2 — Approve content?** STOP. Wait for response:
  - **yes**: Sniper commits the ADR. `mission_result.adr = <path>`. Continue to §9.
  - **no**: ADR discarded (file removed). `mission_result.status = documentation_applied` (no ADR). Continue to §9.
  - **edit**: User wants to adjust the content. Accept inline edits and re-present the draft. Re-open gate 2.

No gate after Sniper — content approval happens BEFORE the commit, not after.

**Language instruction for Archivist:** generate the ADR in the language defined by `active.language.docs`.
- `docs: pt-BR` → content in Portuguese
- `docs: en` → content in English
- Canonical contract: `contracts/07-adr.md`

**Minimum ADR structure (template for Archivist):**

```markdown
# ADR: {title}
**Date:** {date} | **Status:** accepted
**Mission:** {mission_id}

## Context
{problem statement derived from proposal.md or tasks.md}

## Decision
{what was chosen and why}

## Consequences
{accepted trade-offs; what becomes harder; what becomes easier}
```

The template above uses English section names. If `docs: pt-BR`, Archivist uses `Contexto`, `Decisão`, `Consequências`.
