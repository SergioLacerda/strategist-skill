# Strategist Skill — Documentação Técnica Detalhada

## Runtime Cognitivo, Autoaprendizado e Convergência de Agentes IA

---

## Visão Geral

O **Strategist** é uma skill autônoma de orquestração de implementação de funcionalidades(missões) para agentes IA.

Ela coordena trabalho multi-fase através de três papeis(slots) plugáveis:

```
Ranger (discovery) -> Papel responsavel por explorar o escopo do problema apartir do prompt inicial
Archivist (refinement) -> Papel responsavel por refinar o escopo do problema e criar um plano de execução
Sniper (execution) -> Papel responsavel por executar o plano de execução
```
Cada papel tem sua funcao definida, porem o interessante eh que voce pode dizer qual skill cumpre aquele papel.
o Strategist orquestra o fluxo, valida contratos, emite eventos de progresso e impõe o approval gate.

É **standalone por padrão** e pode opcionalmente integrar como plugin a modelos de governaça(harness engineering) como o **SDD Harness**

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

### Wizard (interativo)

```bash
sh install.sh
```

TUI que permite selecionar template, base_path, providers dos três slots(papeis), e fonte de conhecimento(opcional). Gera `active.yaml` e escreve `roles/default.yaml` com os providers escolhidos.

### Zero config no repositório alvo

Toda configuração vive dentro da skill root (`active.yaml`, `personas/`, `roles/`, `memory/`, `knowledge.index.yaml`). O repositório alvo recebe apenas artefatos da skill.

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
│   └── epic.yaml                    ← tom estratégico; labels: ranger/archivist/sniper
│
├── roles/
│   ├── default.yaml                 ← bindings padrão dos slots
│   ├── mission.yaml                 ← bindings para SDD (Sniper = _injected_by_sdd)
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
    └── archivist/                    ← skill de refinamento (Archivist slot padrão)
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

Pipeline completo: Ranger → housekeeping_scan → [mini approval gate] → Sniper(side quests) → Archivist → approval gate → Sniper(main)

### Fluxo de Negócio: Iteração entre Papéis

```
                    ┌─────────────────────────────────────────┐
                    │              STRATEGIST                 │
                    │      Orquestrador — não executa         │
                    └──────────────────┬──────────────────────┘
                                       │
                         ┌─────────────▼──────────────┐
                         │           RANGER           │
                         │         (discovery)        │
                         │  "O que precisa ser feito? │
                         │   Qual o estado atual?"    │
                         │  → pending/<id>-discovery  │
                         └─────────────┬──────────────┘
                                       │
                         ┌─────────────▼──────────────┐
                         │    Housekeeping Scan       │
                         │  (Ataque de oportunidade)  │
                         │  Detecta workspace stale:  │
                         │  todo/ pending/ refined/   │
                         └──────┬──────────────┬──────┘
                     sem quests │              │ com side quests
                                │     ┌────────▼────────────┐
                                │     │   Mini Approval     │
                                │     │      Gate           │
                                │     └────────┬────────────┘
                                │    aprovado  │
                                │     ┌────────▼────────────┐
                                │     │       SNIPER        │
                                │     │   (side quests)     │
                                │     │  Move / promove     │
                                │     │  artefatos stale    │
                                │     └────────┬────────────┘
                                │              │ side quest report
                                └──────────────┤
                         ┌─────────────▼──────────────┐
                         │         ARCHIVIST          │
                         │        (refinement)        │
                         │  "Como executar? Que       │
                         │   decisões tomar?"         │
                         │  → refined/<id>/           │
                         │    proposal.md             │
                         │    design.md               │
                         │    tasks.md                │
                         └─────────────┬──────────────┘
                                       │
                    ┌──────────────────▼──────────────────┐
                    │           Approval Gate             │
                    │   PARADA OBRIGATÓRIA (se tasks.md   │
                    │   não estiver vazio)                │
                    └──────────────────┬──────────────────┘
                              aprovado │
                         ┌─────────────▼──────────────┐
                         │           SNIPER           │
                         │         (execution)        │
                         │  "Executar o plano         │
                         │   aprovado."               │
                         │  → done/<id>-report.md     │
                         └─────────────┬──────────────┘
                                       │
                         ┌─────────────▼──────────────┐
                         │       Learning Phase       │
                         │      (não-bloqueante)      │
                         │  Registra outcomes e       │
                         │  source-hints com          │
                         │  aprovação humana          │
                         └────────────────────────────┘
```

#### Side Quests — Detalhe

O Ataque de oportunidade(Housekeeping Scan) detecta artefatos em estado inconsistente **antes** de a análise principal começar. Isso evita que o Archivist trate como "pendente" algo que já foi resolvido.

```
              todo/spec.md ──────────► já implementado no git?
                                              │ sim
                                              ▼
                                       move → done/

           pending/discovery.md ──────► tem plano em refined/?
                                              │ sim
                                              ▼
                                       promote → move pending → done

            refined/plan/ ────────────► tem report em done/?
                                              │ sim
                                              ▼
                                       promote → move refined → done
```

Cada operação de workspace requer aprovação no **mini gate** antes de o Sniper executar.

---

### Fluxo Técnico Interno

```
INVOCAÇÃO
─────────────────────────────────────────────────────────────────────
  prompt do usuário
       │
       ▼
  ┌──────────────────────────────────────────────────────────────┐
  │ 1. Bootstrap                                                 │
  │    • Carrega active.yaml (fonte única de config)             │
  │    • Resolve persona → tone_directive + phase_labels         │
  │    • SDD injection (se plugin ativo): override Sniper slot,  │
  │      base_path, knowledge_paths, governance_context          │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 2. Preflight                              para na 1ª falha   │
  │    2a. Carrega .strategist/index.yaml → arquivos load_always │
  │    2b. Carrega identity/what-i-am.yaml + drift-patterns.yaml │
  │    2c. Resolve slot providers (roles/<config>.yaml)          │
  │        skill_root → .claude/skills → registry                │
  │    2d. Valida contratos de risco:                            │
  │        Ranger    → write_pending                             │
  │        Archivist → write_analysis                            │
  │        Sniper    → controlled                                │
  │    emit: phase=preflight status=done                         │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 3. Intake                                                    │
  │    invoca prompt-intake skill                                │
  │    → mission_contract: task_type, risk_level, constraints    │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 4. Context Enrichment                                        │
  │    invoca context-enrichment (knowledge.index.yaml + hints)  │
  │    invoca dossier-builder → dossiê mínimo por token budget   │
  └──────────────────────────┬───────────────────────────────────┘
                             │
FASES DA MISSÃO
─────────────────────────────────────────────────────────────────────
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 5a. RANGER (discovery slot)         contrato: write_pending  │
  │     → pending/<mission_id>-discovery.md                      │
  │     emit: phase=<ranger_label> status=done                   │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 5b. Housekeeping Scan  (Strategist interno — não é slot)     │
  │     varre: todo/ pending/ refined/                           │
  │     produz: side quest manifest                              │
  │     emit: phase=housekeeping_scan status=done side_quests=N  │
  └────────────┬────────────────────────┬────────────────────────┘
  N=0          │                        │ N>0
  pula 5c/5d   │                        │
               │         ┌─────────────▼──────────────────────┐
               │         │ 5c. Mini Approval Gate (condicional)│
               │         │     STOP — aguarda resposta         │
               │         └─────────────┬──────────────────────┘
               │                       │ yes/select
               │         ┌─────────────▼──────────────────────┐
               │         │ 5d. SNIPER side quests              │
               │         │     contrato: controlled            │
               │         │     mv todo→done / promove pending  │
               │         │     produz: side quest report       │
               │         │     falha: non-blocking             │
               │         └─────────────┬──────────────────────┘
               │                       │ side quest report
               └───────────────────────┤
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 5e. ARCHIVIST (refinement slot)   contrato: write_analysis   │
  │     input: discovery artifact + side quest report            │
  │     → refined/<mission_id>/                                  │
  │         proposal.md  design.md  tasks.md                     │
  │     emit: phase=<archivist_label> status=done                │
  └──────────────────────────┬───────────────────────────────────┘
                             │
GATE E EXECUÇÃO
─────────────────────────────────────────────────────────────────────
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 6. Approval Gate                                             │
  │    lê tasks.md:                                             │
  │    • vazio → plan_only automático (sem gate)                 │
  │    • interno → gate normal                                   │
  │    • externo → gate + aviso de escopo                        │
  │    STOP — aguarda resposta explícita                         │
  └────────────┬──────────────────────────┬──────────────────────┘
  no/decline   │                          │ yes/approve
  plan_only    │                          │
               │         ┌───────────────▼───────────────────┐
               │         │ 7. SNIPER (execution slot)        │
               │         │    contrato: controlled            │
               │         │    input: refined plan (tasks.md)  │
               │         │    → done/<mission_id>-report.md   │
               │         │    emit: phase=<sniper_label> done  │
               │         └───────────────┬───────────────────┘
               │                         │
               └─────────────────────────┤
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 8. Learning Phase (non-blocking)                             │
  │    invoca response-critic → invoca learning-curator          │
  │    checkpoint ao usuário antes de qualquer escrita           │
  │    falha: não bloqueia o resultado                           │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 9. Mission Result                                            │
  │    status: completed | plan_only | blocked                   │
  │    artifacts: discovery, side_quest_report?,                 │
  │               refined_plan, execution_report?                │
  └──────────────────────────────────────────────────────────────┘
```

---

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
Ranger / discovery (discovery slot)
  ↓
Ataque de oportunidade / Housekeeping Scan (interno — sem slot)
  ↓
[Mini Approval Gate — somente se side quests > 0]
  ↓ (se aprovado)
Sniper / side quests (execution slot — operações de workspace)
  ↓
Archivist / refinement (refinement slot)
  ↓
Approval Gate ← PARADA OBRIGATÓRIA (se tasks.md não estiver vazio)
  ↓ (somente com aprovação explícita)
Sniper / execution (execution slot)
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
   - Sobrescreve o Sniper slot com `sdd_injection.execution_provider`
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

Para cada slot (discovery, refinement, execution), tenta resolver o `skill.yaml` do provider na ordem:
1. `<skill_root>/<provider>/skill.yaml`
2. `.claude/skills/<provider>/skill.yaml`
3. Entrada no skill registry (se presente)

Se o provider é `_injected_by_sdd`, resolve de `sdd_injection.execution_provider`.
Se nenhum caminho resolve: emite evento bloqueado e para.

**2d. Validação de contratos de risco**

- Ranger: `risk_score` DEVE ser `write_pending`
- Archivist: `risk_score` DEVE ser `write_analysis`
- Sniper: `risk_score` DEVE ser `controlled`
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

#### 5a. Ranger (discovery slot)

```
[Strategist] phase=<ranger_label> status=running skill=<provider> checklist=0/3
```

Invoca o provider do slot discovery com:
- Prompt do usuário
- `mission_contract.planning_rules`
- Dossiê do context enrichment

Artefato produzido: `<base_path>/pending/<mission_id>-discovery.md`

```
[Strategist] phase=<ranger_label> status=done artifact=<path>
```

Falha → evento bloqueado com `reason=ranger_failed`. Não avança para Archivist.

#### 5b. Ataque de oportunidade / Housekeeping Scan (interno — sem slot)

Após o Ranger concluir, o Strategist executa um scan determinístico de `<base_path>/`.
**Não delega a um slot provider** — executa internamente.

| Diretório | Verificação | Tipo de side quest |
|-----------|-------------|-------------------|
| `todo/` | Spec tem commit correspondente no git? | `move_to_done` |
| `pending/` | Spec tem plano correspondente em `refined/`? | `promote` |
| `refined/` | Plano tem relatório em `done/`? | `promote` |

Produz um **side quest manifest** com os itens detectados.

- Se manifest vazio: pula 5c e 5d, avança direto para 5e.
- Se manifest não vazio: apresenta mini approval gate (5c).

#### 5c. Mini Approval Gate (condicional — somente se side_quests > 0)

STOP. Nenhum arquivo é movido sem aprovação explícita.

Apresenta ao usuário:

```
[Strategist] Workspace scan encontrou N side quest(s) antes da análise principal:
  [1] <origin_path> → <destination> (<type>)  Motivo: <reason>
Aprovar todos? [yes / no / select]
```

Respostas:
- **yes**: avança para 5d (Sniper executa todos os side quests).
- **no**: descarta manifest, avança para 5e com workspace como está.
- **select**: usuário especifica itens por número; Sniper executa apenas os selecionados.

Invocar Sniper para side quests sem resposta ao mini gate é um **forbidden behavior**.

#### 5d. Sniper: Side Quest Execution (condicional — somente se mini approval concedido)

Invoca o execution slot provider com os itens aprovados do manifest. Operações permitidas:
- `mv <base_path>/todo/<file> <base_path>/done/<file>`
- Atualizar campo `Status:` em arquivos markdown
- Nenhuma escrita fora de `<base_path>/`

Produz um **side quest report** (bloco markdown inline — não é arquivo gravado em disco).

Falha no side quest Sniper é **non-blocking**: registra a falha, avança para 5e com report parcial ou vazio.

#### 5e. Archivist (refinement slot)

```
[Strategist] phase=<archivist_label> status=running skill=<provider> checklist=1/3
```

Invoca o provider do slot refinement com:
- Path do artefato de discovery
- Side quest report (se presente) como contexto adicional — itens movidos não devem ser tratados como pendentes
- `mission_contract.planning_rules`
- Dossiê

Artefato produzido: `<base_path>/refined/<mission_id>/` (subdiretório com três arquivos)
- `proposal.md` — o quê e por quê (alimentado pelo artefato de discovery)
- `design.md` — como (arquitetura, componentes afetados, decisões)
- `tasks.md` — passos de implementação numerados (input do Sniper)

Regras:
- Archivist nunca produz um `.md` avulso em `refined/` — sempre o subdiretório com os três arquivos
- Se `tasks.md` estiver vazio ou ausente após Archivist concluir, Sniper não é invocado

```
[Strategist] phase=<archivist_label> status=done artifact=<path>
```

Falha → evento bloqueado. Não apresenta approval gate.

---

### 6. Approval Gate (OBRIGATÓRIO)

Após o Archivist concluir, o Strategist lê `tasks.md` antes de apresentar o gate:

**Se `tasks.md` estiver vazio ou ausente:**
  emite `[Strategist] phase=approval_gate status=plan_only`, retorna resultado `status: plan_only`.
  O gate **não é apresentado** — a missão está completa.

**Se `tasks.md` contiver tarefas apenas dentro de `<base_path>/`:**
  apresenta o gate uma vez com o plano visível.

**Se `tasks.md` contiver tarefas que escrevem fora de `<base_path>/` (código, git, config, sistema):**
  apresenta o gate com aviso explícito de escopo externo.

Apresenta ao usuário (template da persona ativa):

```
Archivist briefing complete. Mission plan at: <artifact_path>

Authorize Sniper deployment? (yes / no / review)
```

Respostas:
- **yes / approve / authorize** → avança para Sniper
- **no / decline / stop** → emite `[Strategist] phase=approval_gate status=plan_only`, retorna resultado `status: plan_only` com paths dos artefatos de discovery e plano refinado
- **review** → exibe conteúdo do plano, re-pergunta

Invocar Sniper sem aprovação explícita é um **forbidden behavior**.

---

### 7. Sniper (execution slot)

```
[Strategist] phase=<sniper_label> status=running skill=<provider> checklist=2/3
```

Invoca o provider do slot execution com:
- Path do plano refinado aprovado
- `mission_contract.planning_rules`

Artefato produzido: `<base_path>/done/<mission_id>-report.md`

```
[Strategist] phase=<sniper_label> status=done artifact=<path>
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
  discovery: <path>           # presente se Ranger executou
  side_quest_report: inline   # presente se side quests executaram (bloco inline, não arquivo)
  refined_plan: <path>        # presente se Archivist executou
  execution_report: <path>    # presente se Sniper executou
blockers: []                  # códigos de bloqueio se status=blocked
```

---

## Modos de Operação (Personas)

O Strategist tem dois modos com o **mesmo pipeline** e **voz diferente**.

| Aspecto | Pragmatic | Epic |
|---------|-----------|------|
| **Tom** | Analítico, direto | Estratégico, decisivo |
| **Label discovery** | `analysis` | `ranger` |
| **Label refinement** | `refinement` | `archivist` |
| **Label execution** | `execution` | `sniper` |
| **Approval prompt** | "Refinement complete. Proceed?" | "Authorize Sniper deployment?" |
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
refinement: archivist
execution: caveman
```

### roles/mission.yaml (SDD integration)

```yaml
discovery: diagnose
refinement: archivist
execution: _injected_by_sdd   # resolvido de sdd_injection.execution_provider em runtime
```

Override por missão: `--roles mission`

---

## Integração SDD (Opcional)

O Strategist pode ser registrado como plugin no SDD Harness via `.sdd/plugins/registry.yaml`.

Quando ativo, o SDD injeta em `active.yaml`:

```yaml
sdd_injection:
  execution_provider: sdd-ask       # sobrescreve Sniper slot
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
| `slot_risk_mismatch` | Ranger ≠ `write_pending`, Archivist ≠ `write_analysis`, ou Sniper ≠ `controlled` | Substituir provider |
| `intake_conflict_unresolved` | Dois aliases de constraint mutuamente exclusivos no prompt | Usuário deve esclarecer |
| `preflight_failed` | Qualquer checagem de preflight falhou | Ver reason code emitido |
| `user_denies_execution` | Usuário recusou no approval gate | Retorna `plan_only` (não é erro) |
| `ranger_failed` | Ranger não produziu artefato | Não avança para Archivist |
| `refinement_failed` | Archivist não produziu artefato | Não apresenta approval gate |
| `side_quest_sniper_failed` | Sniper falhou nos side quests | Non-blocking — avança para Archivist |

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

8. **Invocar side quest Sniper sem apresentar mini approval gate** — o mini gate é obrigatório quando o housekeeping scan retorna itens. Avançar direto para execução de side quests é um bypass proibido.

9. **Delegar housekeeping scan a um slot provider** — o scan é uma fase interna determinística do Strategist. Ranger, Archivist e Sniper não executam o scan.

10. **Pedir ao Sniper para criar documentos, specs ou planos** — criação de artefatos de análise é responsabilidade do Archivist (contrato: `write_analysis`). Sniper executa; nunca escreve análises.

---

## Drift Self-Correction

Quando `drift-patterns.yaml` está carregado, o agente verifica padrões antes de cada fase:

| Padrão | Sintoma | Correção |
|--------|---------|----------|
| `direct_execution` | Prestes a executar trabalho de slot diretamente | Parar. Identificar slot ativo. Invocar provider. Retomar. |
| `silent_phase_advance` | Prestes a iniciar próxima fase sem emitir evento `done` | Emitir evento `done` primeiro. |
| `approval_bypass` | Prestes a invocar Sniper sem perguntar ao usuário | Parar. Apresentar approval gate prompt. |
| `scope_expansion` | Endereçando algo fora da missão do usuário | Parar. Retornar ao escopo da missão. |
| `sniper_provider_override` | Resolveu Sniper de fonte diferente de roles config ou sdd_injection | Parar. Re-resolver da fonte declarada. |
| `side_quest_approval_bypass` | Prestes a mover arquivos do housekeeping_scan sem apresentar mini approval gate | Parar. Apresentar mini approval gate com o manifest completo. |
| `route_plan_creation_to_sniper` | Prestes a pedir ao Sniper para criar documento, spec ou plano | Parar. Criação de artefatos é trabalho do Archivist. Retornar à fase 5e. |
| `housekeeping_scan_as_slot` | Prestes a delegar o housekeeping scan ao Ranger ou outro slot | Parar. Executar o scan diretamente como Strategist (fase interna). |

---

## Decisões Arquiteturais

### Standalone-first

Strategist não requer SDD nem nenhum framework de governança. A integração SDD é opcional e aditiva — não modifica a lógica do pipeline.

### Pipeline idêntico para ambos os modos

Pragmatic e Epic compartilham o mesmo pipeline. A separação é apenas de vocabulário e tom. Adicionar novos modos no futuro requer apenas um novo arquivo `personas/<mode>.yaml`.

### Preflight valida todos os slots antes de começar

Falha rápida no preflight evita execução parcial. Descobrir um mismatch de risco após o Ranger já ter rodado criaria estado de artefato órfão e situação de difícil recuperação.

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
[Strategist] phase=<ranger_label> status=running skill=<provider> checklist=0/3
[Strategist] phase=<ranger_label> status=done artifact=<path>
[Strategist] phase=housekeeping_scan status=running
[Strategist] phase=housekeeping_scan status=done side_quests=N
[Strategist] phase=side_quest_execution status=running          # somente se side_quests > 0 e aprovado
[Strategist] phase=side_quest_execution status=done             # somente se side_quests > 0 e aprovado
[Strategist] phase=<archivist_label> status=running skill=<provider> checklist=1/3
[Strategist] phase=<archivist_label> status=done artifact=<path>
[Strategist] phase=approval_gate status=waiting                 # somente se tasks.md não vazio
[Strategist] phase=<sniper_label> status=running skill=<provider> checklist=2/3
[Strategist] phase=<sniper_label> status=done artifact=<path>
```

Emitir evento `running` e avançar para a próxima fase sem emitir `done` é uma violação do padrão `silent_phase_advance`.

---

Para instruções completas do agente, ver [`strategist/SKILL.md`](strategist/SKILL.md).
Para regras de roteamento obrigatórias, ver [`strategist/protocol.md`](strategist/protocol.md).
Para contrato completo da skill, ver [`strategist/skill.yaml`](strategist/skill.yaml).
