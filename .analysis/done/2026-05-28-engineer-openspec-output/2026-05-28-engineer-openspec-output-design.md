# Engineer OpenSpec Output — Design Spec
**Date:** 2026-05-28
**Status:** pending implementation
**Topic:** Engineer produz subdirectório OpenSpec em refined/; gate condicional; wizard sugere providers

---

## Problem Statement

Na missão "ainda temos um item pendente?", o Strategist:

1. Delegou "criar documento de análise" ao **Hunter** com approval gate — errado: criar
   planos/specs é trabalho do **Engineer** (contrato `write_analysis`)
2. O output foi para `pending/` como arquivo avulso — errado: deve ir para `refined/`
   no formato OpenSpec (proposal + design + tasks)
3. O approval gate disparou para uma operação puramente analítica — errado: gate só
   deve aparecer quando Hunter vai executar fora de `.analysis/`
4. O wizard não explica por que `brainstorming` e `openspec-explore` são os defaults

---

## Goals

1. Engineer produz `refined/<mission_id>/proposal.md` + `design.md` + `tasks.md` silenciosamente
2. Approval Gate só dispara quando `tasks.md` exige operações fora de `<base_path>/`
3. Criar documentos de análise nunca é roteado ao Hunter
4. Wizard explica o papel de cada slot e o porquê dos defaults

---

## Seção 1 — Engineer: output OpenSpec em `refined/`

**Arquivo:** `strategist/SKILL.md` — Seção 5e

Engineer produz um **subdiretório OpenSpec** em `refined/`:

```
<base_path>/refined/<mission_id>/
  proposal.md   ← o quê e porquê (resumo da descoberta + decisão tomada)
  design.md     ← como (arquitetura, mudanças, componentes afetados)
  tasks.md      ← passos de implementação numerados (input direto do Hunter)
```

**Regras:**
- Engineer NUNCA produz um `.md` avulso em `refined/` — sempre o subdiretório com 3 arquivos
- `proposal.md` alimentado pelo Scout's discovery artifact
- `tasks.md` é o contrato de entrada do Hunter; se vazio ou ausente, Hunter não é invocado
- Engineer escreve os 3 arquivos diretamente (contrato `write_analysis`), sem gate

**Atualização em `skill.yaml` pipeline stage:**

```yaml
- stage: refinement
  slot: refinement
  input: [discovery_artifact, side_quest_report]
  artifact_path: "<base_path>/refined/<mission_id>/"   # diretório, não arquivo
  produces:
    - "<base_path>/refined/<mission_id>/proposal.md"
    - "<base_path>/refined/<mission_id>/design.md"
    - "<base_path>/refined/<mission_id>/tasks.md"
```

---

## Seção 2 — Approval Gate: condicional sobre `tasks.md`

**Arquivo:** `strategist/SKILL.md` — Seção 6

Após Engineer completar, Strategist lê `tasks.md` antes de decidir sobre o gate:

```
Se tasks.md está vazio ou ausente:
  → Missão puramente analítica
  → Emitir: [Strategist] phase=approval_gate status=plan_only
  → Retornar mission result status: plan_only
  → NÃO apresentar gate

Se tasks.md contém tarefas apenas dentro de <base_path>/:
  → Gate padrão apresentado uma vez com plano completo à vista
  → Aguardar: yes / no / review

Se tasks.md contém tarefas fora de <base_path>/ (código, git, config, sistema):
  → Gate com aviso explícito de escopo externo apresentado
  → Aguardar: yes / no / review
```

---

## Seção 3 — Drift Pattern e Forbidden Behavior

**Arquivo:** `strategist/SKILL.md` — Seção Drift Self-Correction

Novo drift pattern:
```
route_plan_creation_to_hunter: "Você está prestes a pedir ao Hunter para criar
um documento de análise, spec, ou plano de implementação." → Stop.
Criar documentos de análise é trabalho do Engineer (write_analysis).
Retorne à fase 5e e invoque o slot de refinement.
```

**Arquivo:** `strategist/skill.yaml` — `forbidden_behaviors`

```yaml
- delegate_analysis_creation_to_hunter
```

---

## Seção 4 — Wizard: sugerir providers com contexto

**Arquivo:** `strategist/install.sh` — `run_wizard()`

Adicionar linha de contexto antes de cada prompt de provider:

```bash
# Scout
echo ""
echo "  Scout: descobre o espaço do problema → escreve discovery em pending/"
echo "  Provider recomendado: brainstorming (explora antes de decidir)"
printf "Scout provider (write_pending) [brainstorming]: "
read -r scout_provider
scout_provider="${scout_provider:-brainstorming}"

# Engineer
echo ""
echo "  Engineer: refina a descoberta → escreve proposal/design/tasks em refined/"
echo "  Provider recomendado: openspec-explore (gera estrutura OpenSpec)"
printf "Engineer provider (write_analysis) [openspec-explore]: "
read -r engineer_provider
engineer_provider="${engineer_provider:-openspec-explore}"

# Hunter
echo ""
echo "  Hunter: executa o plano refinado → requer approval gate"
echo "  Provider recomendado: sdd-ask (execução governada)"
printf "Hunter provider (controlled) [sdd-ask]: "
read -r hunter_provider
hunter_provider="${hunter_provider:-sdd-ask}"
```

---

## Seção 5 — Mudanças em `skill.yaml` (consolidado)

```yaml
pipeline:
  # ... stages anteriores ...
  - stage: refinement
    slot: refinement
    input: [discovery_artifact, side_quest_report]
    artifact_path: "<base_path>/refined/<mission_id>/"
    produces:
      - proposal.md
      - design.md
      - tasks.md

forbidden_behaviors:
  # ... existentes ...
  - delegate_analysis_creation_to_hunter   # NOVO
```

---

## Ordem de execução

1. Editar `strategist/SKILL.md` seção 5e — output OpenSpec subdirectory
2. Editar `strategist/SKILL.md` seção 6 — gate condicional sobre `tasks.md`
3. Editar `strategist/SKILL.md` seção Drift — novo pattern `route_plan_creation_to_hunter`
4. Editar `strategist/skill.yaml` — `artifact_path` para diretório + `forbidden_behaviors`
5. Editar `strategist/install.sh` — contexto nos prompts do wizard

---

## O que NÃO muda

- Contratos dos slots (`write_pending`, `write_analysis`, `controlled`)
- housekeeping_scan e mini approval gate
- side quest execution
- Bootstrap scripts e release workflow
- learning phase

---

## Critério de done

- `/strategist` com missão de análise → Engineer cria `refined/<id>/proposal.md` + `design.md` + `tasks.md` silenciosamente
- `tasks.md` vazio → gate não aparece, missão encerra com `plan_only`
- `tasks.md` com tarefas de código → gate aparece com aviso de escopo externo
- Nenhum path de missão roteia "criar spec/plano" para Hunter
- `bash install.sh --wizard` exibe contexto de papel para cada slot
