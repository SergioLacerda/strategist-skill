# Tasks: Sincronizar .strategist/SKILL.md com canônico
**Escopo:** `.strategist/SKILL.md` — 4 edições cirúrgicas

## T1 — Inserir tabela de contratos

Após a linha:
```
You do not perform discovery, refinement, or execution yourself — you delegate.
```

Inserir:
```

| Internal name | Slot key   | Contract       | Progress label |
|---------------|------------|----------------|----------------|
| Ranger        | discovery  | write_pending  | discovery      |
| Archivist     | refinement | write_analysis | refinement     |
| Sniper        | execution  | controlled     | execution      |

```

## T2 — Atualizar formato de artefato do Archivist

Find:
```
- Primary artifact path: `<base_path>/refined/<mission_id>-plan.md`
- Secondary artifact scope: `<base_path>/` (Archivist may create additional `.md` summaries here)
```

Replace:
```
- Artifact path: `<base_path>/refined/<mission_id>/` (subdirectory)
  - `proposal.md` — what and why (fed by Ranger's discovery artifact)
  - `design.md` — how (architecture, affected components, decisions)
  - `tasks.md` — numbered implementation steps (Sniper's input contract)

**Rules:**
- Archivist NEVER produces a standalone `.md` in `refined/` — always the three-file subdirectory
- If `tasks.md` is empty or absent after Archivist completes, Sniper is not invoked
- Archivist writes all three files directly (contract: `write_analysis`), no gate
```

## T3 — Atualizar lógica do Approval Gate

Find:
```
**If the plan requires no Sniper execution** (purely analytical mission, no writes outside
`<base_path>/`): emit `[Strategist] phase=approval_gate status=plan_only`, return mission
result with `status: plan_only`. Do NOT present the gate — the mission is complete.

**If the plan requires Sniper to write only inside `<base_path>/`** (e.g., moving files
to `done/`, creating a report): present the gate once with the full plan visible.

**If the plan requires Sniper to write outside `<base_path>/`** (code, git, config, system):
present the gate with an explicit external-scope warning.
```

Replace:
```
Read `<base_path>/refined/<mission_id>/tasks.md` before deciding:

**If `tasks.md` is empty or absent:**
  emit `[Strategist] phase=approval_gate status=plan_only`, return mission result
  with `status: plan_only`. Do NOT present the gate — the mission is complete.

**If `tasks.md` contains tasks scoped only to `<base_path>/`:**
  present the gate once with the full plan visible.

**If `tasks.md` contains tasks that write outside `<base_path>/` (code, git, config, system):**
  present the gate with an explicit external-scope warning.
```

## T4 — Adicionar drift pattern ausente

No final da seção "## Drift Self-Correction", após a última linha:
```
- `housekeeping_scan_as_slot`: You are about to delegate the housekeeping scan to Ranger or another slot. → Stop. Execute the scan directly as Strategist (deterministic, internal phase).
```

Adicionar:
```
- `route_plan_creation_to_sniper`: You are about to ask Sniper to create a document, spec, analysis, or implementation plan. → Stop. Document authoring is Archivist's work (contract: `write_analysis`). Return to phase 5e and invoke the refinement slot.
```

## Verificação

```bash
diff strategist/SKILL.md .strategist/SKILL.md
```
Expected: zero diff.
