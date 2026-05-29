# Role Standardization — Discovery Artifact
**Date:** 2026-05-28
**Status:** pending refinement
**Topic:** Padronizar papéis internos/externos da Strategist skill

---

## 1. Terminologia atual — inventário por arquivo

### Camada 1 — Slot keys em `roles/*.yaml` e `skill.yaml` (chamados de "internal names" no código)

| Arquivo | Linha | Ocorrência | Tipo |
|---------|-------|-----------|------|
| `roles/default.yaml` | 2 | `# Slot keys: discovery, refinement, execution (internal names).` | comentário |
| `roles/default.yaml` | 5–7 | `discovery: sdd-diagnose` / `refinement: engineer` / `execution: caveman` | slot key → provider binding |
| `roles/mission.yaml` | 5–7 | `discovery: diagnose` / `refinement: engineer` / `execution: _injected_by_sdd` | slot key → provider binding |
| `roles/spec-driven.yaml` | 4–6 | `discovery: brainstorm` / `refinement: openspec` / `execution: _injected_by_sdd` | slot key → provider binding |
| `skill.yaml` `slots:` | 28–42 | keys `discovery`, `refinement`, `execution` | slot contract definitions |
| `skill.yaml` `pipeline:` | 57–85 | `slot: discovery`, `slot: refinement`, `slot: execution` | pipeline stage → slot binding |

### Camada 2 — Phase labels em `personas/*.yaml` (exibidos em eventos de progresso)

| Arquivo | Linha | Ocorrência | Tipo |
|---------|-------|-----------|------|
| `personas/epic.yaml` | 5–7 | `discovery: scout` / `refinement: engineer` / `execution: hunter` | phase label (display) |
| `personas/pragmatic.yaml` | 5–7 | `discovery: analysis` / `refinement: refinement` / `execution: execution` | phase label (display) |

**Observação crítica:** as duas personas usam sistemas diferentes de label.
`epic` usa os nomes antigos (scout/engineer/hunter); `pragmatic` usa os nomes de slot key como label.
Não há convenção estabelecida.

### Camada 3 — Prosa em `SKILL.md` (Scout / Engineer / Hunter como nomes canônicos)

| Linha | Ocorrência | Contexto |
|-------|-----------|---------|
| 4 | "Scout (discovery) → Engineer (refinement) → Hunter (execution)" | declaração da pipeline |
| 51 | "For each slot (scout, engineer, hunter):" | preflight resolution |
| 61 | "Scout (discovery):" | contrato de risco |
| 67 | "Hunter (execution):" | contrato de risco |
| 108 | "Pipeline: Scout → housekeeping_scan → Hunter(side quests) → Engineer → Hunter(main)" | pipeline overview |
| 110 | "### 5a. Scout (discovery slot)" | cabeçalho de seção |
| 120 | "Scout writes the artifact directly" | prose |
| 126 | "scout_failed" | blocker code |
| 148 | "proceed to 5e (Engineer)" | referência de fluxo |
| 176 | "Invoking Hunter side quests without..." | forbidden behavior |
| 215 | "### 5e. Engineer (refinement slot)" | cabeçalho de seção |
| 268 | "## 7. Hunter (execution slot)" | cabeçalho de seção |
| 335 | "approval_bypass: ...invoke Hunter..." | drift pattern |
| 338 | "hunter_provider_override" | drift pattern |
| 339 | "housekeeping_scan_as_slot: ...Scout..." | drift pattern |

### Camada 4 — `install.sh` (wizard de configuração)

| Linha | Ocorrência | Tipo |
|-------|-----------|------|
| 181 | `# Scout provider` | comentário |
| 183 | `echo "  Scout: descobre o espaço do problema..."` | prompt de wizard |
| 186 | `printf "Scout provider (write_pending)..."` | prompt de wizard |
| 187–189 | `read -r scout` / `validate_provider "$scout" ...` | variável / validação |
| 193 | `# Engineer provider` | comentário |
| 195 | `echo "  Engineer: refina a descoberta..."` | prompt de wizard |
| 199–201 | `read -r engineer` / `validate_provider "$engineer" ...` | variável / validação |
| 205 | `# Hunter provider` | comentário |
| 207 | `echo "  Hunter: executa o plano refinado..."` | prompt de wizard |
| 210–214 | `read -r hunter` / `validate_provider "$hunter" ...` | variável / validação |
| 231–233 | `scout: ${scout}` / `engineer: ${engineer}` / `hunter: ${hunter}` | escrita em roles YAML |

**Problema:** linha 231–233 escreve as variáveis `scout`/`engineer`/`hunter` como KEYS em roles YAML,
mas as keys esperadas são `discovery`/`refinement`/`execution`. **Bug latente no wizard.**

### Camada 5 — Sub-skill interna `skills/engineer/`

| Arquivo | Linha | Ocorrência | Tipo |
|---------|-------|-----------|------|
| `skills/engineer/skill.yaml` | 1 | `id: engineer` | skill ID (referenciado por `roles/default.yaml: refinement: engineer`) |
| `skills/engineer/skill.yaml` | 14 | "path to Scout's output" | prose |
| `skills/engineer/skill.yaml` | 10 | "handoff to Hunter" | prose |
| `skills/engineer/SKILL.md` | 1 | `# engineer — Agent Instructions` | título |
| `skills/engineer/SKILL.md` | 32, 53–65 | "Hunter" (3 ocorrências) | prose |

**Observação:** `roles/default.yaml` tem `refinement: engineer` — o valor `engineer` é o `id` da sub-skill.
Renomear o papel interno para `Archivist` não quebra essa binding automaticamente;
o `id: engineer` da sub-skill pode permanecer como está ou ser renomeado separadamente.

### Camada 6 — `skill.yaml` forbidden_behaviors

```
scout_writes_outside_pending
engineer_writes_non_md
invoke_side_quest_hunter_without_approval
invoke_execution_slot_without_approval
```

### Camada 7 — `schemas/progress-contract.yaml`

| Linha | Ocorrência | Tipo |
|-------|-----------|------|
| 15–16 | "epic: scout / engineer / hunter" + nota sobre nomes internos | documentação |
| 35 | `phase=scout`, `phase=engineer`, `phase=hunter` nos exemplos | event examples |
| 38 | `phase=hunter` no exemplo | event examples |

### Camada 8 — `templates/domain/identity/what-i-am.yaml`

```
- A planner. I do not produce plans; Engineer does.
- An executor. I do not apply changes; Hunter does.
- I never invoke Hunter without explicit user approval at the approval gate.
```

### Camada 9 — `protocol.md`

Usa `discovery`/`refinement`/`execution` como nomes de slot (não Scout/Engineer/Hunter).
Única exceção: "execution provider" referenciado no contexto de sdd_injection.

---

## 2. Análise: interno vs externo — estado atual

```
HOJE (inconsistente)
═══════════════════════════════════════════════════════

  CÓDIGO (roles YAML, skill.yaml slots)
  ─────────────────────────────────────
  Slot keys:  discovery / refinement / execution
  Chamados de "internal names" em comentário

  PROSA (SKILL.md, install.sh, personas/epic)
  ────────────────────────────────────────────
  Nomes:  Scout / Engineer / Hunter
  Tratados como "canonical names"

  PERSONAS (phase_labels — display nos eventos)
  ──────────────────────────────────────────────
  epic:      scout / engineer / hunter    ← usa nomes antigos como labels
  pragmatic: analysis / refinement / execution  ← usa slot keys como labels

  RESULTADO
  ─────────
  Nenhum documento define a distinção entre as duas camadas.
  Os nomes colidem e a convenção varia por arquivo.
```

```
PROPOSTO (padronizado)
═══════════════════════════════════════════════════════

  INTERNO (conceito dentro do Strategist)
  ────────────────────────────────────────
  Ranger    → slot: discovery
  Archivist → slot: refinement
  Sniper    → slot: execution

  EXTERNO (label nos eventos / binding em roles YAML)
  ────────────────────────────────────────────────────
  discovery   → Ranger executa este slot
  refinement  → Archivist executa este slot
  execution   → Sniper executa este slot

  SEPARAÇÃO CLARA
  ───────────────
  - SKILL.md usa Ranger/Archivist/Sniper na prosa
  - phase_labels nas personas usa discovery/refinement/execution (externo)
  - roles YAML não muda: discovery/refinement/execution como keys
  - install.sh mostra "Ranger (discovery)" / "Archivist (refinement)" / "Sniper (execution)"
```

---

## 3. Gaps nos contratos atuais

### Gap 1 — Sem documento canônico de separação
Não existe nenhum arquivo que diga explicitamente:
> "Os nomes internos são X; os labels externos são Y. Toda a prosa usa X. Todos os eventos usam Y."

O comentário em `roles/default.yaml` e a nota em `progress-contract.yaml` são parciais e inconsistentes entre si.

### Gap 2 — `epic.yaml` usa nomes internos como labels de display
`phase_labels.discovery = scout` significa que em modo epic, o evento de progresso diz
`phase=scout` — misturando o nome interno (scout) como se fosse label externo.
Após a padronização, o epic mode deve emitir `phase=discovery` (externo) com contexto interno "Ranger".

### Gap 3 — Bug no wizard (install.sh linhas 231–233)
O wizard lê variáveis `scout`/`engineer`/`hunter` mas escreve as keys `scout:`/`engineer:`/`hunter:` no roles YAML.
As keys esperadas pelo Strategist são `discovery:`/`refinement:`/`execution:`.
Roles gerados pelo wizard atual são inválidos.

### Gap 4 — Sub-skill `skills/engineer/` tem identity acoplada ao nome
O `id: engineer` da sub-skill é referenciado em `roles/default.yaml: refinement: engineer`.
Renomear a sub-skill para `archivist` exige atualizar todos os roles que a referenciam.
Se a sub-skill NÃO for renomeada, há discrepância: papel interno "Archivist" mas skill-id "engineer".

### Gap 5 — forbidden_behaviors usam nomes antigos como prefixo
`scout_writes_outside_pending` e `engineer_writes_non_md` codificam o nome do papel no identificador do comportamento proibido. Após renomeação, esses identificadores ficam desatualizados.

---

## 4. Estimativa de impacto da mudança

| Arquivo | Tipo de mudança | Volume | Risco |
|---------|----------------|--------|-------|
| `SKILL.md` | rename Scout→Ranger, Engineer→Archivist, Hunter→Sniper | ~25 ocorrências | baixo |
| `skill.yaml` | forbidden_behaviors + slot descriptions | ~5 linhas | baixo |
| `personas/epic.yaml` | phase_labels + prose | ~6 linhas | baixo |
| `personas/pragmatic.yaml` | approval_prompt (menciona "Refinement") | ~1 linha | baixo |
| `install.sh` | variáveis + prompts + **fix do bug de escrita** | ~15 linhas | médio (correção de bug) |
| `skills/engineer/` | rename diretório + id + prose interna | ~10 linhas | médio (path resolution) |
| `schemas/progress-contract.yaml` | exemplos + nota | ~5 linhas | baixo |
| `templates/domain/identity/what-i-am.yaml` | prose | ~3 linhas | baixo |
| `protocol.md` | prose | ~3 linhas | baixo |
| `roles/*.yaml` | NÃO muda (keys já são externos) | 0 | nenhum |

**Decisão pendente para refinement:**
- Renomear `skills/engineer/` para `skills/archivist/` e atualizar `id: engineer` → `id: archivist`?
  - Pro: consistência total
  - Contra: quebra `roles/default.yaml` e todos os roles que apontam `refinement: engineer`
- Manter `skills/engineer/` como está, atualizando só a prosa interna?
  - Pro: sem quebra de bindings
  - Contra: sub-skill visível com nome desatualizado

**Decisão (aprovada pelo usuário):** Opção A — rename completo.
- `skills/engineer/` → `skills/archivist/`
- `id: engineer` → `id: archivist`
- Todos os `roles/*.yaml` com `refinement: engineer` → `refinement: archivist`
