# Docs Nomenclature Update — Discovery Artifact
**Date:** 2026-05-28
**Status:** pending refinement
**Topic:** Atualizar readme.md e readme_detailed.md com Ranger/Archivist/Sniper

---

## Inventário de ocorrências

### readme.md (5 ocorrências)

| Linha | Conteúdo atual | Tipo |
|-------|---------------|------|
| 8 | `Scout → Engineer → Hunter` (prose + destaque) | rename |
| 14 | `Scout (discovery), Engineer (refinement) e Hunter (execution)` | rename |
| 15 | `nunca invoca o Hunter sem aprovação` | rename |
| 55 | `Slot providers: Scout, Engineer, Hunter` (tabela) | rename |

### readme_detailed.md (~40 ocorrências)

**Seção pipeline/intro:**
- L14: `Scout (discovery) → Engineer (refinement) → Hunter (execution)` — rename
- L91: `labels: scout/engineer/hunter` — rename labels + valores corretos
- L95: `hunter = _injected_by_sdd` — rename
- L123: `└── engineer/` (tree do filesystem) — rename para `archivist/`
- L163-169: bloco de slots (Scout / Engineer / Hunter) — rename

**Seção preflight (L210, L220-221):**
- L210: `Para cada slot (scout, engineer, hunter)` — rename para `(discovery, refinement, execution)`
- L220: `Scout e Engineer: risk_score DEVE ser read_only` — rename + **corrigir valores**: `write_pending` / `write_analysis`
- L221: `Hunter: risk_score DEVE ser controlled_write` — rename + **corrigir valor**: `controlled`

**Seção fases 5a/5b/7:**
- L267, L270, L281, L284: Scout → Ranger, labels scout_label → ranger_label
- L286, L289, L300: Engineer → Archivist, labels engineer_label → archivist_label
- L309, L314, L316, L320, L324: Hunter → Sniper
- L328, L331, L341: Hunter → Sniper, labels hunter_label → sniper_label

**Mission result (L370-372):**
- Comentários `# presente se Scout/Engineer/Hunter executou` → Ranger/Archivist/Sniper

**Tabela personas (L385-388):**
- `scout` → `Ranger`, `engineer` → `Archivist`, `hunter` → `Sniper`
- "Authorize Hunter deployment?" → "Authorize Sniper deployment?"

**Exemplos de roles YAML (L464, L472):**
- `refinement: engineer` → `refinement: archivist`

**Template epic-sdd (L488):**
- `# sobrescreve hunter slot` → `# sobrescreve Sniper slot`

**Error codes (L513-514):**
- `Scout não produziu artefato. Não avança para Engineer` → Ranger/Archivist

**Drift patterns (L546, L548):**
- `approval_bypass: invocar Hunter` → Sniper
- `hunter_provider_override` → `sniper_provider_override`

**Texto de contexto (L564):**
- `após o Scout já ter rodado` → Ranger

**Phase labels em event examples (L586-592):**
- `scout_label`, `engineer_label`, `hunter_label` → `ranger_label`, `archivist_label`, `sniper_label`

---

## Correções factuais (além do rename)

| Arquivo | Linha | Atual | Correto |
|---------|-------|-------|---------|
| readme_detailed.md | 220 | `Scout e Engineer: risk_score DEVE ser read_only` | Ranger: `write_pending` / Archivist: `write_analysis` |
| readme_detailed.md | 221 | `Hunter: risk_score DEVE ser controlled_write` | Sniper: `controlled` |
