# Slot Write Contracts — Design Spec
**Date:** 2026-05-28
**Status:** pending implementation
**Topic:** Expandir contratos de Scout e Engineer para escrita silenciosa de .md no escopo correto

---

## Problem Statement

Scout e Engineer são `read_only`. Qualquer escrita de artefato — mesmo criar um `.md`
em `pending/` — passa pelo Hunter com approval gate obrigatório. Isso torna o fluxo
de análise excessivamente interativo para operações de baixo risco.

**Exemplo concreto:** criar `.analysis/pending/wizard-validation-plan.md` exigiu
Approval Gate + "yes" explícito, mesmo sendo uma escrita local de markdown sem
impacto em código ou sistema.

---

## Goals

1. Scout escreve seu artefato de descoberta em `pending/` **silenciosamente**
2. Engineer escreve plano refinado em `refined/` e summaries em `<base_path>/` **silenciosamente**
3. Approval gate permanece apenas para Hunter, que opera fora de `.analysis/`
4. Notificações somente em erros, violações de escopo, ou dúvidas
5. Contratos são verificados em preflight — não em runtime

---

## Vocabulário de Contratos (atualizado)

| Contrato | Escopo de escrita | Tipo permitido | Gate? |
|----------|-------------------|----------------|-------|
| `read_only` | nenhum | — | n/a |
| `write_pending` | `<base_path>/pending/` | `.md` apenas | não |
| `write_analysis` | `<base_path>/` + `<base_path>/refined/` | `.md` apenas | não |
| `controlled` | qualquer lugar | qualquer tipo | **sim** |

**Invariante:** escrita fora do escopo declarado é BLOCK, mesmo que o arquivo seja `.md`.

---

## Seção 1 — Scout: contrato `write_pending`

**Slot:** discovery
**Contrato novo:** `write_pending` (era `read_only`)

Scout pode criar ou sobrescrever arquivos `.md` dentro de `<base_path>/pending/`.
Nenhuma outra escrita é permitida.

**Comportamento no pipeline:**
- Strategist passa o artifact path ao Scout: `<base_path>/pending/<mission_id>-discovery.md`
- Scout escreve diretamente, sem gate
- Strategist emite: `[Strategist] phase=scout status=done artifact=<path>`

**Violações bloqueadas:**
- Scout tenta escrever `.yaml` → BLOCK: `slot_write_type_violation`
- Scout tenta escrever em `refined/` → BLOCK: `slot_write_scope_violation`

---

## Seção 2 — Engineer: contrato `write_analysis`

**Slot:** refinement
**Contrato novo:** `write_analysis` (era `read_only`)

Engineer pode criar ou sobrescrever arquivos `.md` em:
- `<base_path>/refined/` — plano refinado principal
- `<base_path>/` — summaries, índices, consolidações (raiz da análise)

**O que Engineer NÃO pode:**
- Escrever `.yaml`, `.json`, `.sh` ou qualquer não-`.md`
- Escrever fora de `<base_path>/`
- Escrever em `<base_path>/pending/` (pertence ao Scout)
- Escrever em `<base_path>/done/` (pertence ao Hunter)
- Escrever em `<base_path>/todo/` (input apenas)

**Comportamento no pipeline:**
- Engineer recebe artifact path: `<base_path>/refined/<mission_id>-plan.md`
- Engineer escreve diretamente, sem gate
- Engineer pode opcionalmente criar `<base_path>/summary.md` ou similar
- Strategist emite: `[Strategist] phase=engineer status=done artifact=<path>`

---

## Seção 3 — Hunter: contrato `controlled` (inalterado, gate redefinido)

Hunter permanece `controlled`. O que muda é a **condição de disparo** do Approval Gate.

**Antes:** gate sempre disparava antes de Hunter.

**Depois:** Strategist avalia o plano do Engineer antes de apresentar o gate.

```
Se plano do Engineer contém operações APENAS em <base_path>/done/:
  → gate apresentado (1x), mas apenas com o plano final — não interrompe análise
  
Se plano do Engineer contém operações FORA de <base_path>/:
  → gate apresentado com aviso explícito de escopo externo
  
Se plano do Engineer não requer Hunter (missão puramente analítica):
  → gate não é apresentado; missão encerra com status: plan_only
```

**Resultado:** para missões de análise pura (análise → plano → done/report.md), o fluxo
é silencioso exceto por um gate no final com o plano completo à vista.

---

## Seção 4 — `skill.yaml` — Mudanças

```yaml
slots:
  discovery:
    contract: write_pending          # era: read_only
    write_scope: "<base_path>/pending/"
    write_types: [".md"]
    description: Explores the problem space and writes discovery artifact to pending/.

  refinement:
    contract: write_analysis         # era: read_only
    write_scope:
      - "<base_path>/"
      - "<base_path>/refined/"
    write_types: [".md"]
    description: Reads discovery artifact, writes refined plan and any .md summaries.

  execution:
    contract: controlled             # inalterado
    description: Executes the refined plan. Approval gate required.
```

Novos stop conditions:

```yaml
stop_conditions:
  - slot_write_type_violation    # tentou escrever tipo não permitido pelo contrato
  - slot_write_scope_violation   # tentou escrever fora do escopo do contrato
```

Novos forbidden behaviors:

```yaml
forbidden_behaviors:
  - scout_writes_outside_pending     # Scout escreveu fora de pending/
  - engineer_writes_outside_analysis # Engineer escreveu fora de base_path/
  - engineer_writes_non_md           # Engineer escreveu arquivo não-.md
```

---

## Seção 5 — `SKILL.md` — Mudanças

### 5.1 Seção 2d — Validação de contratos

Substituir:

```
- Scout (discovery) and Engineer (refinement) slots: risk_score MUST be read_only.
- Hunter (execution) slot: risk_score MUST be controlled.
```

Por:

```
Scout (discovery): risk_score MUST be write_pending
  → autorizado a criar/sobrescrever .md em <base_path>/pending/ sem gate
  → qualquer escrita fora desse escopo ou tipo diferente de .md: BLOCK slot_write_scope_violation

Engineer (refinement): risk_score MUST be write_analysis
  → autorizado a criar/sobrescrever .md em <base_path>/ e <base_path>/refined/ sem gate
  → qualquer escrita fora de <base_path>/ ou tipo diferente de .md: BLOCK slot_write_scope_violation

Hunter (execution): risk_score MUST be controlled
  → approval gate obrigatório antes de qualquer execução
```

### 5.2 Seção 5a — Scout passa a escrever diretamente

Remover: "Discovery artifact path: … (Strategist grava)"
Adicionar:

```
Scout recebe o artifact path e escreve diretamente.
Strategist não intermediá a escrita — apenas aguarda conclusão e emite done event.
```

### 5.3 Seção 5e — Engineer passa a escrever diretamente

Igual ao Scout: Engineer recebe o path e escreve diretamente.
Strategist aguarda e emite done event.

### 5.4 Seção 6 — Approval Gate com condição de escopo

Adicionar antes da instrução de gate:

```
Antes de apresentar o gate, avaliar o plano do Engineer:
- Se o plano requer operações FORA de <base_path>/: gate com aviso de escopo externo.
- Se o plano opera apenas dentro de <base_path>/: gate padrão com plano à vista.
- Se o plano não requer Hunter: emitir status: plan_only, não apresentar gate.
```

### 5.5 Seção 2c — Resolver novos valores no known-providers.yaml

Adicionar ao lookup de risk_score: os valores `write_pending` e `write_analysis`
são válidos. Atualizar `known-providers.yaml` template com os valores corretos para
os providers configurados.

---

## known-providers.yaml — Valores atualizados

```yaml
providers:
  brainstorming: write_pending      # Scout — era read_only
  openspec-explore: write_analysis  # Engineer — era read_only
  openspec-propose: write_analysis
  openspec-apply-change: controlled
  sdd-ask: controlled
  sdd-diagnose: write_analysis
  # ...
```

---

## O que NÃO muda

- Pipeline de fases (Scout → housekeeping_scan → ... → Engineer → gate → Hunter)
- Approval gate existe — só a condição de disparo é refinada
- Hunter contract `controlled` — inalterado
- Side quest execution — Hunter ainda requer mini approval gate (5c)
- Bootstrap, install.sh, personas, knowledge index

---

## Critério de done

- `/strategist` cria `.analysis/pending/*.md` via Scout sem nenhum gate ou pergunta
- `/strategist` cria `.analysis/refined/*.md` via Engineer sem nenhum gate ou pergunta
- Gate aparece apenas uma vez, antes do Hunter, com o plano completo visível
- Scout tentando escrever `.yaml` → BLOCK com mensagem clara
- Engineer tentando escrever fora de `<base_path>/` → BLOCK com mensagem clara
