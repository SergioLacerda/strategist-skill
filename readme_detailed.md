# Strategist Skill — Documentação Técnica Detalhada

## Runtime Cognitivo, Governança e Convergência de Agentes IA

---

## Visão Geral

O **Strategist** é uma skill autônoma de orquestração de missões para agentes IA.

Ela coordena trabalho multi-fase através de três slots plugáveis:

```
Scout (discovery) → Engineer (refinement) → Hunter (execution)
```

O Strategist não executa discovery, refinement ou execution diretamente — ele delega para os providers configurados em cada slot. Toda implementação concreta é feita pelos providers; o Strategist orquestra o fluxo, valida contratos, emite eventos de progresso e impõe o approval gate.

É **standalone por padrão** e pode opcionalmente integrar ao **SDD** como plugin registrado.

---

## Problema

Agentes IA tendem a falhar quando operam sem governança:

- perdem contexto entre iterações
- confundem análise com execução
- executam sem diagnóstico ou plano aprovado
- ignoram arquitetura e decisões existentes
- entram em loops de retry sem rastreabilidade
- geram prompts grandes e pouco densos
- alteram código antes de haver aprovação humana

O Strategist resolve isso com:

```
governança estrutural
+ roteamento via slots plugáveis
+ contexto seletivo via knowledge index
+ approval gate obrigatório
+ learning loop não-bloqueante
+ drift self-correction
```

---

## Instalação

### Silent (padrão)

```bash
sh install.sh
```

Gera `active.yaml` a partir do template `pragmatic-standalone` e scaffolda o workspace no repositório alvo. Sem prompts. Ideal para automação e CI.

### Wizard (interativo)

```bash
sh install.sh --wizard
```

TUI que permite selecionar template, base_path, providers dos três slots, e fonte de conhecimento. Gera `active.yaml` e escreve `roles/default.yaml` com os providers escolhidos.

### Repositório alvo customizado

```bash
sh install.sh --target /path/to/repo
```

### Zero config no repositório alvo

Toda configuração vive dentro da skill root (`active.yaml`, `personas/`, `roles/`, `memory/`, `knowledge.index.yaml`). O repositório alvo recebe apenas artefatos de missão.

---

## Estrutura de Arquivos

```
strategist/
├── install.sh                       ← instalação silent + wizard TUI
├── skill.yaml                       ← contrato da skill (slots, pipeline, forbidden_behaviors)
├── SKILL.md                         ← instruções completas do agente
├── protocol.md                      ← regras de roteamento obrigatórias
├── active.yaml                      ← gerado no install (gitignore'd)
├── knowledge.index.yaml             ← índice de fontes de conhecimento
│
├── personas/
│   ├── pragmatic.yaml               ← tom analítico; labels: analysis/refinement/execution
│   └── epic.yaml                    ← tom estratégico; labels: scout/engineer/hunter
│
├── roles/
│   ├── default.yaml                 ← bindings padrão dos slots
│   ├── mission.yaml                 ← bindings para SDD (hunter = _injected_by_sdd)
│   └── spec-driven.yaml             ← bindings para fluxo spec-driven
│
├── schemas/
│   ├── intake.schema.yaml           ← campos do mission_contract
│   └── progress-contract.yaml       ← formato dos eventos de progresso
│
├── templates/
│   ├── pragmatic-standalone.yaml    ← active.yaml template: pragmatic, sem SDD
│   ├── epic-standalone.yaml         ← active.yaml template: epic, sem SDD
│   ├── epic-sdd.yaml                ← active.yaml template: epic, com SDD injection
│   └── domain/                      ← templates do workspace (.strategist/)
│       ├── index.yaml
│       ├── identity/
│       ├── directives/
│       ├── rubrics/
│       └── patterns/
│
├── memory/
│   ├── outcomes.jsonl               ← histórico de missões (gitignore'd)
│   └── source-hints.yaml            ← ajustes de prioridade aprendidos (gitignore'd)
│
└── skills/
    ├── prompt-intake/               ← classifica task_type, risk_level, constraints
    ├── context-enrichment/          ← consulta knowledge index, aplica source-hints
    ├── dossier-builder/             ← monta dossiê mínimo dentro do token budget
    ├── response-critic/             ← avalia output dos slots contra rubrica
    ├── learning-curator/            ← propõe entradas para outcomes + source-hints
    └── engineer/                    ← skill de refinamento (Engineer slot padrão)
        ├── skill.yaml
        └── SKILL.md
```

### Workspace no repositório alvo (gerado pelo install)

```
<base_path>/
├── todo/                            ← missões aguardando execução
├── pending/                         ← artefatos de discovery em andamento
├── refined/                         ← planos revisados prontos para aprovação
├── done/                            ← relatórios de execução concluídos
└── .strategist/                     ← domínio interno (copiado de templates/domain/)
    ├── index.yaml                   ← controla carregamento seletivo de arquivos
    ├── identity/
    │   ├── what-i-am.yaml
    │   └── drift-patterns.yaml
    ├── directives/
    ├── rubrics/
    └── patterns/
```

---

## Pipeline de Missão

### Fluxo Principal

```
Prompt do usuário
  ↓
Bootstrap (active.yaml + persona + SDD injection)
  ↓
Preflight (valida slots, carrega domínio interno)
  ↓
Intake (extrai mission_contract)
  ↓
Context Enrichment (consulta knowledge index → monta dossiê)
  ↓
Scout / análise (discovery slot)
  ↓
Engineer / refinement (refinement slot)
  ↓
Approval Gate ← PARADA OBRIGATÓRIA
  ↓ (somente com aprovação explícita)
Hunter / execution (execution slot)
  ↓
Learning Phase (não-bloqueante)
  ↓
Mission Result
```

---

### 1. Bootstrap

Ao ser invocado, o agente:

1. Carrega `active.yaml` (fonte única de configuração).
2. Resolve a persona (`personas/<mode>.yaml`) e aplica `tone_directive` e `phase_labels`.
3. Se `--mode` foi passado, sobrescreve o modo apenas para esta missão.
4. Se `--roles` foi passado, sobrescreve `roles_config` apenas para esta missão.
5. Se `sdd_injection` está presente e o plugin está ativo em `.sdd/plugins/registry.yaml`:
   - Sobrescreve o Hunter slot com `sdd_injection.execution_provider`
   - Sobrescreve `base_path`
   - Adiciona `sdd_injection.knowledge_paths` às fontes do knowledge index (sem substituir)
   - Carrega `sdd_injection.governance_context` como contexto read-only adicional

---

### 2. Preflight

Executado **antes de qualquer slot ou intake**. Para na primeira falha.

**2a. Domínio interno**

Carrega `<base_path>/.strategist/index.yaml`. Se não existir, continua sem domínio.
Se existir, carrega apenas os arquivos listados em `load_always`. Nenhum arquivo fora do index é carregado.

**2b. Arquivos de identidade**

- `identity/what-i-am.yaml` → carrega `core_invariants` (ativos durante toda a missão)
- `identity/drift-patterns.yaml` → carrega todos os padrões (usados para autocorreção)

**2c. Resolução de slot providers**

Para cada slot (scout, engineer, hunter), tenta resolver o `skill.yaml` do provider na ordem:
1. `<skill_root>/<provider>/skill.yaml`
2. `.claude/skills/<provider>/skill.yaml`
3. Entrada no skill registry (se presente)

Se o provider é `_injected_by_sdd`, resolve de `sdd_injection.execution_provider`.
Se nenhum caminho resolve: emite evento bloqueado e para.

**2d. Validação de contratos de risco**

- Scout e Engineer: `risk_score` DEVE ser `read_only`
- Hunter: `risk_score` DEVE ser `controlled_write`
- Mismatch → evento bloqueado com `reason=slot_risk_mismatch`

**Evento de conclusão:** `[Strategist] phase=preflight status=done slots=ok`

---

### 3. Intake

Invoca a skill `prompt-intake` com o prompt completo do usuário.

Resultado (`mission_contract`):

```yaml
task_type: architecture_analysis | refactor | general | ...
risk_level: low | medium | high
constraints:
  delivery_strategy: incremental | total
  legacy_compatibility: required | not_required
  execution_intent: plan_only | plan_then_execute
```

Conflitos nos constraints → para e pede esclarecimento ao usuário.
Campos ausentes → defaults aplicados via `intake.schema.yaml`.

O `mission_contract` é passado para todos os providers de slot.

---

### 4. Context Enrichment

Invoca `context-enrichment` com `task_type` e o token budget da missão.

O enrichment:
1. Consulta `knowledge.index.yaml` filtrando por tags que correspondem ao `task_type`
2. Aplica ajustes de prioridade de `memory/source-hints.yaml`
3. Carrega excerpts dentro do token budget

Carrega arquivos de `load_by_task_type[task_type]` do `index.yaml` (se domínio interno presente).

Invoca `dossier-builder` para montar o dossiê mínimo para os slot providers. Se nenhuma fonte corresponder, o dossiê contém apenas `task_type` e `output_template`.

---

### 5. Fases da Missão

#### 5a. Scout (discovery slot)

```
[Strategist] phase=<scout_label> status=running skill=<provider> checklist=0/3
```

Invoca o provider do slot discovery com:
- Prompt do usuário
- `mission_contract.planning_rules`
- Dossiê do context enrichment

Artefato produzido: `<base_path>/pending/<mission_id>-discovery.md`

```
[Strategist] phase=<scout_label> status=done artifact=<path>
```

Falha → evento bloqueado com `reason=scout_failed`. Não avança para Engineer.

#### 5b. Engineer (refinement slot)

```
[Strategist] phase=<engineer_label> status=running skill=<provider> checklist=1/3
```

Invoca o provider do slot refinement com:
- Path do artefato de discovery
- `mission_contract.planning_rules`
- Dossiê

Artefato produzido: `<base_path>/refined/<mission_id>-plan.md`

```
[Strategist] phase=<engineer_label> status=done artifact=<path>
```

Falha → evento bloqueado. Não apresenta approval gate.

---

### 6. Approval Gate (OBRIGATÓRIO)

Após o Engineer concluir, **PARA**. Hunter não é invocado sem aprovação explícita.

Apresenta ao usuário (template da persona ativa):

```
Engineer briefing complete. Mission plan at: <artifact_path>

Authorize Hunter deployment? (yes / no / review)
```

Respostas:
- **yes / approve / authorize** → avança para Hunter
- **no / decline / stop** → emite `[Strategist] phase=approval_gate status=plan_only`, retorna resultado `status: plan_only` com paths dos artefatos de discovery e plano refinado
- **review** → exibe conteúdo do plano, re-pergunta

Invocar Hunter sem aprovação explícita é um **forbidden behavior**.

---

### 7. Hunter (execution slot)

```
[Strategist] phase=<hunter_label> status=running skill=<provider> checklist=2/3
```

Invoca o provider do slot execution com:
- Path do plano refinado aprovado
- `mission_contract.planning_rules`

Artefato produzido: `<base_path>/done/<mission_id>-report.md`

```
[Strategist] phase=<hunter_label> status=done artifact=<path>
```

---

### 8. Learning Phase (não-bloqueante)

Após a missão concluir (status `completed` ou `plan_only`):

1. Invoca `response-critic` com os outputs dos slots e a rubrica do `task_type`
2. Invoca `learning-curator` com a avaliação do critic, resultado da missão e `task_type`

O `learning-curator` **DEVE apresentar um checkpoint ao usuário** antes de escrever qualquer arquivo.
Propõe atualizações a:
- `memory/outcomes.jsonl` — registro da missão (append-only)
- `memory/source-hints.yaml` — ajustes de prioridade para fontes de conhecimento

Ambos requerem aprovação explícita (podem ser aprovados/rejeitados individualmente).

**Falha na learning phase não bloqueia nem modifica o resultado da missão.**

---

### 9. Mission Result

```yaml
mission_id: <id>
status: completed | plan_only | blocked
artifacts:
  discovery: <path>         # presente se Scout executou
  refined_plan: <path>      # presente se Engineer executou
  execution_report: <path>  # presente se Hunter executou
blockers: []                # códigos de bloqueio se status=blocked
```

---

## Modos de Operação (Personas)

O Strategist tem dois modos com o **mesmo pipeline** e **voz diferente**.

| Aspecto | Pragmatic | Epic |
|---------|-----------|------|
| **Tom** | Analítico, direto | Estratégico, decisivo |
| **Label discovery** | `analysis` | `scout` |
| **Label refinement** | `refinement` | `engineer` |
| **Label execution** | `execution` | `hunter` |
| **Approval prompt** | "Refinement complete. Proceed?" | "Authorize Hunter deployment?" |
| **Template padrão** | `pragmatic-standalone.yaml` | `epic-standalone.yaml` / `epic-sdd.yaml` |

Seleção:
- Via `active.yaml`: `mode: pragmatic` ou `mode: epic`
- Override por missão: `--mode pragmatic` ou `--mode epic`

---

## Sistema de Conhecimento

### knowledge.index.yaml

Índice multi-source para context enrichment. Cada fonte possui:

```yaml
sources:
  - id: project-architecture
    type: docs
    path: /abs/path/to/docs/architecture
    tags: [architecture, system-design, architecture_analysis]
    priority: high

  - id: past-good-examples
    type: examples
    path: .analysis/.strategist/patterns/good
    tags: [examples, patterns, refactor, architecture_analysis]
    priority: medium

  - id: team-directives
    type: directives
    path: /abs/path/to/team-directives.md
    tags: [all]
    priority: high
```

O `context-enrichment` filtra por tags que correspondem ao `task_type` da missão e carrega apenas as fontes relevantes dentro do token budget.

### source-hints.yaml

Camada de ajuste de prioridade aprendida. Sobreposta sobre o index antes do ranking. Atualizada pelo `learning-curator` com aprovação humana.

### Domínio Interno (.strategist/)

O `index.yaml` controla carregamento seletivo — o agente **nunca escaneia o diretório completo**:

```yaml
load_always:
  - identity/what-i-am.yaml
  - identity/drift-patterns.yaml
  - directives/core.yaml

load_by_task_type:
  architecture_analysis:
    - directives/by-task/architecture-analysis.yaml
    - rubrics/architecture-analysis.yaml
  refactor:
    - directives/by-task/architecture-analysis.yaml
    - rubrics/architecture-analysis.yaml

load_on_demand:
  - patterns/good/
  - patterns/bad/
  - memory/lessons.yaml
```

---

## Configuração de Slots (roles/)

Cada arquivo `roles/<config>.yaml` declara os providers dos três slots:

### roles/default.yaml (standalone)

```yaml
discovery: sdd-diagnose
refinement: engineer
execution: caveman
```

### roles/mission.yaml (SDD integration)

```yaml
discovery: diagnose
refinement: engineer
execution: _injected_by_sdd   # resolvido de sdd_injection.execution_provider em runtime
```

Override por missão: `--roles mission`

---

## Integração SDD (Opcional)

O Strategist pode ser registrado como plugin no SDD Harness via `.sdd/plugins/registry.yaml`.

Quando ativo, o SDD injeta em `active.yaml`:

```yaml
sdd_injection:
  execution_provider: sdd-ask       # sobrescreve hunter slot
  base_path: .sdd/analysis          # sobrescreve base_path
  knowledge_paths:
    - .sdd/docs                     # adicionado ao knowledge index
  governance_context: .sdd/agent-instructions.md
```

**Regras:**
- Execution slot é **sempre** sobrescrito por `sdd_injection.execution_provider`
- `knowledge_paths` são **adicionados** às fontes, não substituem
- `governance_context` é read-only e não sobrescreve o `protocol.md`

Template para uso com SDD: `templates/epic-sdd.yaml`

---

## Stop Conditions

| Código | Condição | Resolução |
|--------|----------|-----------|
| `slot_provider_not_found` | skill.yaml do provider não encontrado | Verificar id em roles config e caminho do skill root |
| `slot_risk_mismatch` | Discovery/refinement com `risk_score` ≠ `read_only`, ou execution ≠ `controlled_write` | Substituir provider |
| `intake_conflict_unresolved` | Dois aliases de constraint mutuamente exclusivos no prompt | Usuário deve esclarecer |
| `preflight_failed` | Qualquer checagem de preflight falhou | Ver reason code emitido |
| `user_denies_execution` | Usuário recusou no approval gate | Retorna `plan_only` (não é erro) |
| `discovery_failed` | Scout não produziu artefato | Não avança para Engineer |
| `refinement_failed` | Engineer não produziu artefato | Não apresenta approval gate |

---

## Forbidden Behaviors

Os seguintes comportamentos são **nunca permitidos**:

1. **Executar discovery, refinement ou execution diretamente** — sempre delegar ao slot provider configurado. Se não houver provider, para com `slot_provider_not_found`.

2. **Invocar execution slot sem aprovação explícita do usuário** — o approval gate é obrigatório. Qualquer caminho que chegue ao execution slot sem resposta afirmativa ao prompt de aprovação é um bypass proibido.

3. **Escrever config no repositório alvo** — `active.yaml`, `personas/`, `roles/`, `memory/`, `knowledge.index.yaml` e qualquer outra config da skill root nunca devem ser escritas no repositório alvo.

4. **Carregar arquivos não referenciados no `index.yaml`** — quando o domínio interno está presente, apenas arquivos listados em `load_always`, `load_by_task_type` ou `load_on_demand` podem ser carregados.

5. **Escrever em `memory/` sem aprovação** — o `learning-curator` deve apresentar as entradas propostas para revisão antes de qualquer escrita.

6. **Resolver execution slot de fonte não declarada** — o provider do execution slot deve vir de `roles/<config>.yaml` ou `sdd_injection.execution_provider`.

7. **Pular preflight** — preflight executa antes do intake, em toda invocação, inclusive re-invocações com a mesma config.

---

## Drift Self-Correction

Quando `drift-patterns.yaml` está carregado, o agente verifica padrões antes de cada fase:

| Padrão | Sintoma | Correção |
|--------|---------|----------|
| `direct_execution` | Prestes a executar trabalho de slot diretamente | Parar. Identificar slot ativo. Invocar provider. Retomar. |
| `silent_phase_advance` | Prestes a iniciar próxima fase sem emitir evento `done` | Emitir evento `done` primeiro. |
| `approval_bypass` | Prestes a invocar Hunter sem perguntar ao usuário | Parar. Apresentar approval gate prompt. |
| `scope_expansion` | Endereçando algo fora da missão do usuário | Parar. Retornar ao escopo da missão. |
| `hunter_provider_override` | Resolveu Hunter de fonte diferente de roles config ou sdd_injection | Parar. Re-resolver da fonte declarada. |

---

## Decisões Arquiteturais

### Standalone-first

Strategist não requer SDD nem nenhum framework de governança. A integração SDD é opcional e aditiva — não modifica a lógica do pipeline.

### Pipeline idêntico para ambos os modos

Pragmatic e Epic compartilham o mesmo pipeline. A separação é apenas de vocabulário e tom. Adicionar novos modos no futuro requer apenas um novo arquivo `personas/<mode>.yaml`.

### Preflight valida todos os slots antes de começar

Falha rápida no preflight evita execução parcial. Descobrir um mismatch de risco após o Scout já ter rodado criaria estado de artefato órfão e situação de difícil recuperação.

### Learning Loop não-bloqueante

O Learning Loop é uma camada de otimização. Falha em qualquer skill do loop (prompt-intake, context-enrichment, dossier-builder, response-critic, learning-curator) não bloqueia o resultado da missão.

### Carregamento seletivo via index.yaml

O domínio interno cresce ao longo do tempo com exemplos, lições e rubricas. Carregá-lo integralmente em toda missão criaria problema de token budget. O `index.yaml` limita o hot-path a 2–4 arquivos relevantes ao `task_type`.

### Two-file learning cache com aprovação independente

`outcomes.jsonl` e `source-hints.yaml` são aprovados separadamente — o usuário pode querer registrar o resultado da missão sem concordar com um ajuste de prioridade de fonte sugerido.

---

## Fluxo de Progresso

Toda transição de fase emite exatamente um evento:

```
[Strategist] phase=preflight status=done slots=ok
[Strategist] phase=<scout_label> status=running skill=<provider> checklist=0/3
[Strategist] phase=<scout_label> status=done artifact=<path>
[Strategist] phase=<engineer_label> status=running skill=<provider> checklist=1/3
[Strategist] phase=<engineer_label> status=done artifact=<path>
[Strategist] phase=approval_gate status=waiting
[Strategist] phase=<hunter_label> status=running skill=<provider> checklist=2/3
[Strategist] phase=<hunter_label> status=done artifact=<path>
```

Emitir evento `running` e avançar para a próxima fase sem emitir `done` é uma violação do padrão `silent_phase_advance`.

---

Para instruções completas do agente, ver [`strategist/SKILL.md`](strategist/SKILL.md).
Para regras de roteamento obrigatórias, ver [`strategist/protocol.md`](strategist/protocol.md).
Para contrato completo da skill, ver [`strategist/skill.yaml`](strategist/skill.yaml).
