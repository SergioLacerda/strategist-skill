# Side Quest Housekeeping — Design Spec
**Date:** 2026-05-28
**Status:** pending implementation
**Topic:** Adicionar housekeeping_scan e mini approval gate ao pipeline do Strategist

---

## Problem Statement

Quando o Scout descobre que itens em `.analysis/todo/` já estão implementados, o Strategist
não tem caminho estruturado para agir. O comportamento atual é bloquear e pedir autorização
ad-hoc, sem pipeline formal — o Engineer depois analisa um workspace ainda bagunçado.

Raiz do problema: o pipeline assume que todo item descoberto é trabalho futuro.
Não existe fase para reconhecer e resolver "trabalho já feito antes de analisar o resto".

---

## Goals

1. Strategist organiza o workspace antes de analisar — Engineer trabalha com estado real
2. Side quests têm approval gate próprio (mesmo contrato que o Hunter principal)
3. O resultado do side quest alimenta o contexto do Engineer (remove ruído)
4. Falha de housekeeping não cancela a missão principal

---

## Pipeline Revisado

**Antes:**
```
Scout → Engineer → approval gate → Hunter
```

**Depois:**
```
Scout
  └─► housekeeping_scan   (nova fase, determinística, sem slot)
        └─► mini approval gate   ← usuário confirma o que mover
              └─► Hunter (side quests)
                    └─► Engineer (Scout artifact + side quest report)
                          └─► approval gate
                                └─► Hunter (missão principal)
```

Princípio: **organiza primeiro, analisa depois.**

---

## Seção 1 — housekeeping_scan

Fase determinística executada pelo Strategist diretamente (não invoca slot).

**Varredura por diretório:**

```
<base_path>/todo/     → verifica se spec tem commit de implementação correspondente
<base_path>/pending/  → verifica se spec tem plano em refined/
<base_path>/refined/  → verifica se plano tem report em done/
<base_path>/done/     → referência apenas (não move para fora)
```

**Tipos de side quest detectados:**

| Tipo | Condição | Ação sugerida |
|------|----------|---------------|
| `move_to_done` | Spec em `todo/` com impl. confirmada em git | Mover para `done/` |
| `update_status` | Arquivo com campo `Status:` desatualizado | Atualizar campo |
| `promote` | Spec em `pending/` com plano correspondente em `refined/` | Consolidar ou promover |

**Heurística de impl. confirmada** (para `move_to_done`):
- Git log contém commit com mensagem que referencia o spec (por slug de data ou título)
- OU arquivo de spec lista features que existem como código no repo (verificação por nome de função/arquivo)
- OU usuário confirmou no mini approval gate (override manual)

**Output:** side quest manifest — lista de itens com tipo, caminho, motivo sugerido.

Se manifest vazio: pular mini approval gate, ir direto ao Engineer.

---

## Seção 2 — Mini Approval Gate

Apresentado ao usuário após housekeeping_scan, antes de qualquer movimentação.

**Formato:**

```
[Strategist] Workspace scan encontrou N side quests antes da análise principal:

  [1] todo/<arquivo> → done/
       Motivo: <razão detectada>

  [2] pending/<arquivo> → status atualizado
       Motivo: <razão detectada>

Aprovar todos? [yes / no / select]
```

**Respostas:**
- `yes` — Hunter executa todos os side quests
- `no` — side quests descartados, análise principal prossegue com workspace como está
- `select` — usuário especifica quais aprovar (por número ou nome)

**Invariante:** Hunter de side quest só age após `yes` explícito.
Invocar Hunter sem mini approval gate é **forbidden behavior**.

---

## Seção 3 — Hunter: Side Quest Execution

Hunter executa as movimentações/atualizações aprovadas.

**Operações suportadas:**
- `mv <base_path>/todo/<file> <base_path>/done/<file>`
- Atualização do campo `Status:` em markdown (`sed` ou equivalente)
- Nenhuma operação de escrita fora de `<base_path>/`

**Output obrigatório:** side quest report (bloco markdown), usado como contexto pelo Engineer.

**Formato do report:**

```markdown
## Side Quest Report
**Executado:** <data> | **Itens processados:** N

### Movimentações
- `<origem>` → `<destino>` (<motivo>)

### Estado atual do workspace (pós-limpeza)
- `todo/`: N itens restantes
- `pending/`: N itens
- `refined/`: N itens
- `done/`: N itens (incluindo movidos agora)

### Itens excluídos da análise principal
<lista dos itens movidos — Engineer não deve tratá-los como pendentes>
```

---

## Seção 4 — Engineer: Contexto Enriquecido

Engineer recebe:
1. Scout artifact (discovery do problema principal)
2. Side quest report (estado pós-limpeza + itens já resolvidos)

A instrução no prompt do Engineer inclui:
> "Os itens listados em Side Quest Report → Itens excluídos estão resolvidos.
> Não os trate como trabalho pendente. Base sua análise no estado pós-limpeza."

Isso elimina falsos positivos no plano do Engineer.

---

## Seção 5 — Mudanças em SKILL.md

### 5.1 Pipeline de fases (Seção 5 do SKILL.md)

Substituir:

```
5a. Scout
5b. Engineer
```

Por:

```
5a. Scout
5b. Housekeeping Scan  ← nova fase
5c. Mini Approval Gate ← novo gate (só se manifest não-vazio)
5d. Hunter (side quests) ← só se aprovado
5e. Engineer
```

### 5.2 Fase housekeeping_scan (novo bloco)

```
Emit: [Strategist] phase=housekeeping_scan status=running

Execute varredura determinística em <base_path>/. Produza side quest manifest.

Se manifest vazio:
  Emit: [Strategist] phase=housekeeping_scan status=done side_quests=0
  Prosseguir direto ao Engineer.

Se manifest não-vazio:
  Emit: [Strategist] phase=housekeeping_scan status=done side_quests=N
  Apresentar mini approval gate.
```

### 5.3 Injeção do side quest report no Engineer

```
Invocar Engineer com:
  - input: discovery_artifact
  - context: side_quest_report (se presente)
  - planning_rules: mission_contract.planning_rules
```

---

## Seção 6 — Mudanças em skill.yaml

### 6.1 Novo stop condition (non-blocking)

```yaml
stop_conditions:
  - side_quest_hunter_failed   # non-blocking: registra erro, continua para Engineer
```

### 6.2 Novos forbidden behaviors

```yaml
forbidden_behaviors:
  - run_housekeeping_scan_as_slot        # scan é determinístico, não delega para Scout
  - skip_mini_approval_gate              # Hunter de side quest requer approval explícito
  - invoke_side_quest_hunter_without_approval  # igual ao Hunter principal
```

### 6.3 Pipeline stage (novo)

```yaml
pipeline:
  - stage: discovery
    slot: discovery
    ...
  - stage: housekeeping_scan          # NOVO
    type: internal                    # não invoca slot
    produces: side_quest_manifest
    artifact_path: null               # manifest é in-memory, não persiste
  - stage: side_quest_approval        # NOVO
    type: conditional_pause
    condition: side_quest_manifest.count > 0
    description: Present side quest manifest. Require explicit approval before Hunter.
  - stage: side_quest_execution       # NOVO
    slot: execution
    condition: side_quest_approval_granted
    produces: side_quest_report
  - stage: refinement
    slot: refinement
    input: [discovery_artifact, side_quest_report]  # enriquecido
    ...
```

---

## O que NÃO muda

- Contrato do Scout (discovery slot) — não recebe instrução de housekeeping
- Approval gate principal (seção 6 do SKILL.md) — inalterado
- Hunter principal — inalterado
- Learning phase — inalterado
- Providers configurados (brainstorming, openspec-explore, sdd-ask)
- Bootstrap, install.sh, known-providers.yaml

---

## Scope

Esta spec cobre apenas mudanças no Strategist:
- `strategist/SKILL.md` — novas fases 5b–5d, atualização da fase 5e
- `strategist/skill.yaml` — pipeline stages, stop_conditions, forbidden_behaviors

Não cobre:
- Implementação da heurística de git log (pode ser manual no primeiro ciclo)
- UI do mini approval gate além do formato definido aqui
