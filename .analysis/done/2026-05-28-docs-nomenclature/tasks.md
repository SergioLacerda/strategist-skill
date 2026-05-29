# Tasks: Docs Nomenclature Update + Flow Diagrams
**Scope:** external — readme.md and readme_detailed.md in project root
**Idioma:** pt-BR em toda a prosa. Exceções: nomes de papéis (Ranger, Archivist, Sniper), slot keys (discovery, refinement, execution), nomes de variáveis, IDs de eventos, nomes de arquivos, valores YAML, código e termos técnicos sem tradução consagrada (e.g. "approval gate", "side quest", "checklist", "token budget").

---

## T1 — readme.md: bulk rename

replace_all in file:
- `Scout → Engineer → Hunter` → `Ranger → Archivist → Sniper`
- `Scout (discovery), Engineer (refinement) e Hunter (execution)` → `Ranger (discovery), Archivist (refinement) e Sniper (execution)`
- `o Hunter` → `o Sniper`
- `Scout, Engineer, Hunter` (tabela) → `Ranger, Archivist, Sniper`

---

## T2 — readme_detailed.md: bulk rename + factual corrections

### T2a — bulk rename (replace_all)
- `scout_label` → `ranger_label`
- `engineer_label` → `archivist_label`
- `hunter_label` → `sniper_label`
- `scout_failed` → `ranger_failed`
- `hunter_provider_override` → `sniper_provider_override`
- `Scout (discovery)` → `Ranger (discovery)` (where paired)
- `Engineer (refinement)` → `Archivist (refinement)` (where paired)
- `Hunter (execution)` → `Sniper (execution)` (where paired)
- `└── engineer/` → `└── archivist/`
- `scout/engineer/hunter` → `Ranger/Archivist/Sniper`
- `hunter = _injected_by_sdd` → `Sniper = _injected_by_sdd`
- `sobrescreve hunter slot` → `sobrescreve Sniper slot`
- `refinement: engineer` → `refinement: archivist`
- `Scout` → `Ranger` (replace_all remaining)
- `Engineer` → `Archivist` (replace_all remaining)
- `Hunter` → `Sniper` (replace_all remaining)

### T2b — factual corrections (risk_score values)

Find:
```
- Scout e Engineer: `risk_score` DEVE ser `read_only`
- Hunter: `risk_score` DEVE ser `controlled_write`
```
Replace:
```
- Ranger: `risk_score` DEVE ser `write_pending`
- Archivist: `risk_score` DEVE ser `write_analysis`
- Sniper: `risk_score` DEVE ser `controlled`
```

### T2c — slot list (lines ~163-169)

Find (approximate):
```
Scout / análise (discovery slot)
...
Engineer / refinement (refinement slot)
...
Hunter / execution (execution slot)
```
Replace names: Scout→Ranger, Engineer→Archivist, Hunter→Sniper

### T2d — preflight slot list (line ~210)
Find: `Para cada slot (scout, engineer, hunter)`
Replace: `Para cada slot (discovery, refinement, execution)`

---

## T3 — readme_detailed.md: coherence fixes (pipeline gaps)

### T3a — Visão Geral (header section, lines ~13-17)

Replace simplified pipeline:
```
Scout (discovery) → Engineer (refinement) → Hunter (execution)
```
With full pipeline:
```
Ranger (discovery) → Archivist (refinement) → Sniper (execution)
```
(rename only here; full pipeline diagram added in T4)

### T3b — Fluxo Principal (lines ~152-174) — replace entire block

Find the block:
```
### Fluxo Principal

```
[current ASCII flow]
```
```

Replace with updated flow that includes housekeeping_scan and side quests:

```
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
Housekeeping Scan (interno — sem slot)
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
```

### T3c — §5 "Fases da Missão" — replace lead line

Find:
```
### Fluxo Principal

```
(after the heading, before the ASCII flow)
```

At the top of section 5 ("## 5. Fases da Missão"), add pipeline summary line:

Find:
```
## Pipeline de Missão
```

Replace with:
```
## Pipeline de Missão

Pipeline completo: Ranger → housekeeping_scan → [mini approval gate] → Sniper(side quests) → Archivist → approval gate → Sniper(main)
```

### T3d — §5a Ranger section (was Scout)

After the rename in T2a, update content to match SKILL.md §5a:

Find (after rename):
```
#### 5a. Ranger (discovery slot)
```

Ensure the artifact path is correct:
- Artifact: `<base_path>/pending/<mission_id>-discovery.md` ✓ (no change needed)
- Error code: `ranger_failed` ✓ (fixed in T2a)

### T3e — §5b — replace entire Engineer section with housekeeping_scan + Archivist

Find:
```
#### 5b. Engineer (refinement slot)
...
Falha → evento bloqueado. Não apresenta approval gate.
```

Replace with three sections:

```
#### 5b. Housekeeping Scan (interno — sem slot)

Após o Ranger concluir, o Strategist executa um scan determinístico de `<base_path>/`.
**Não delega a um slot provider** — executa internamente.

| Diretório | Verificação | Tipo de side quest |
|-----------|-------------|-------------------|
| `todo/` | Spec tem commit correspondente no git? | `move_to_done` |
| `pending/` | Spec tem plano correspondente em `refined/`? | `promote` |
| `refined/` | Plano tem relatório em `done/`? | `promote` |

Produz um **side quest manifest** com os itens detectados.

- Se manifest vazio: pula 5c e 5d, avança para 5e.
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
- **yes**: avança para 5d.
- **no**: descarta manifest, avança para 5e com workspace como está.
- **select**: usuário especifica itens; Sniper executa apenas os selecionados.

Invocar Sniper para side quests sem aprovação é um **forbidden behavior**.

#### 5d. Sniper: Side Quest Execution (condicional — somente se mini approval concedido)

Invoca o execution slot provider apenas com as operações de workspace aprovadas:
- `mv <base_path>/todo/<file> <base_path>/done/<file>`
- Atualizar campo `Status:` em arquivos markdown

Produz um **side quest report** (bloco markdown inline — não é arquivo).

Falha no side quest Sniper é **non-blocking**: registra falha, avança para 5e com report parcial.

#### 5e. Archivist (refinement slot)

Invoca o provider do slot refinement com:
- Path do artefato de discovery
- Side quest report (se presente) como contexto adicional
- `mission_contract.planning_rules`
- Dossiê

Artefato produzido: `<base_path>/refined/<mission_id>/` (subdiretório)
- `proposal.md` — o quê e por quê (alimentado pelo discovery)
- `design.md` — como (arquitetura, componentes afetados, decisões)
- `tasks.md` — passos de implementação numerados (input do Sniper)

Regras:
- Archivist nunca produz um `.md` avulso em `refined/` — sempre o subdiretório com três arquivos
- Se `tasks.md` estiver vazio ou ausente após Archivist concluir, Sniper não é invocado

Falha → evento bloqueado. Não apresenta approval gate.
```

### T3f — §6 Approval Gate — replace content

Find:
```
### 6. Approval Gate (OBRIGATÓRIO)

Após o Engineer concluir, **PARA**. Hunter não é invocado sem aprovação explícita.
...
Invocar Hunter sem aprovação explícita é um **forbidden behavior**.
```

Replace with:
```
### 6. Approval Gate (OBRIGATÓRIO)

Após o Archivist concluir, o Strategist lê `tasks.md` antes de apresentar o gate:

**Se `tasks.md` estiver vazio ou ausente:**
  emite `[Strategist] phase=approval_gate status=plan_only`, retorna resultado `status: plan_only`.
  O gate **não é apresentado** — a missão está completa.

**Se `tasks.md` contiver tarefas apenas em `<base_path>/`:**
  apresenta o gate uma vez com o plano completo visível.

**Se `tasks.md` contiver tarefas que escrevem fora de `<base_path>/` (código, git, config, sistema):**
  apresenta o gate com aviso explícito de escopo externo.

Apresenta ao usuário (template da persona ativa):
```
Archivist briefing complete. Mission plan at: <artifact_path>
Authorize Sniper deployment? (yes / no / review)
```

Respostas:
- **yes / approve / authorize** → avança para Sniper
- **no / decline / stop** → emite `[Strategist] phase=approval_gate status=plan_only`, retorna resultado `status: plan_only`
- **review** → exibe conteúdo do plano, re-pergunta

Invocar Sniper sem aprovação explícita é um **forbidden behavior**.
```

### T3g — §7 Hunter → Sniper

After rename (T2a), update section to reflect Sniper main execution (unchanged content, but number renaming needed since we inserted §5b-5e):

Sections now are: 5a Ranger, 5b Housekeeping Scan, 5c Mini Gate, 5d Sniper side quests, 5e Archivist, §6 Approval Gate, §7 Sniper main.

The old "§7 Hunter" section → rename to "### 7. Sniper (execution slot)" (rename already covered in T2a).

### T3h — Mission Result (lines ~364-374) — update schema

Find:
```
```yaml
mission_id: <id>
status: completed | plan_only | blocked
artifacts:
  discovery: <path>         # presente se Scout executou
  refined_plan: <path>      # presente se Engineer executou
  execution_report: <path>  # presente se Hunter executou
blockers: []                # códigos de bloqueio se status=blocked
```
```

Replace with:
```
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
```

### T3i — Stop Conditions table — add missing entries

Find:
```
| `discovery_failed` | Scout não produziu artefato | Não avança para Engineer |
| `refinement_failed` | Engineer não produziu artefato | Não apresenta approval gate |
```

Replace with:
```
| `ranger_failed` | Ranger não produziu artefato | Não avança para Archivist |
| `refinement_failed` | Archivist não produziu artefato | Não apresenta approval gate |
| `side_quest_sniper_failed` | Sniper falhou nos side quests | Non-blocking — avança para Archivist |
```

### T3j — Forbidden Behaviors — add missing items

After the last forbidden behavior item, add:

```
8. **Invocar side quest Sniper sem apresentar mini approval gate** — o mini gate é obrigatório quando o housekeeping scan retorna itens. Avançar direto para execução de side quests é um bypass proibido.

9. **Delegar housekeeping scan a um slot provider** — o scan é uma fase interna determinística. Ranger, Archivist e Sniper não executam o scan; apenas Strategist o faz.

10. **Pedir ao Sniper para criar documentos, specs ou planos** — criação de artefatos de análise é responsabilidade do Archivist (contrato: `write_analysis`). Sniper executa; nunca escreve análises.
```

### T3k — Drift Patterns table — add missing patterns

Find the drift patterns table. After the last row, add:
```
| `side_quest_approval_bypass` | Prestes a mover arquivos do housekeeping_scan sem apresentar mini approval gate | Parar. Apresentar mini approval gate com o manifest completo. |
| `route_plan_creation_to_sniper` | Prestes a pedir ao Sniper para criar documento, spec ou plano | Parar. Documento é trabalho do Archivist. Voltar à fase 5e. |
| `housekeeping_scan_as_slot` | Prestes a delegar o housekeeping scan ao Ranger ou outro slot | Parar. Executar o scan diretamente como Strategist. |
```

### T3l — Fluxo de Progresso section — add side_quest events

Find:
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

Replace with:
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

### T3m — Personas table (lines ~384-388) — update phase_labels

Find (after T2a rename):
```
| **Label discovery** | `analysis` | `scout` |
| **Label refinement** | `refinement` | `engineer` |
| **Label execution** | `execution` | `hunter` |
| **Approval prompt** | "Refinement complete. Proceed?" | "Authorize Hunter deployment?" |
```

Replace with:
```
| **Label discovery** | `analysis` | `ranger` |
| **Label refinement** | `refinement` | `archivist` |
| **Label execution** | `execution` | `sniper` |
| **Approval prompt** | "Refinement complete. Proceed?" | "Authorize Sniper deployment?" |
```

---

## T4 — readme_detailed.md: Fluxo de Negócio (novo diagrama)

Add a new section **before** "## Pipeline de Missão" (or as a subsection after "Fluxo Principal"):

### New section: "### Fluxo de Negócio: Iteração entre Papéis"

```markdown
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
                         │    (Strategist interno)    │
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

O Housekeeping Scan detecta artefatos em estado inconsistente **antes** de a análise principal começar. Isso evita que o Archivist trate como "pendente" algo que já foi resolvido.

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
```
```

---

## T5 — readme_detailed.md: Fluxo Técnico Interno (novo diagrama)

Add a new section **after** the "Fluxo de Negócio" section:

### New section: "### Fluxo Técnico Interno"

```markdown
### Fluxo Técnico Interno

```
INVOCAÇÃO
─────────────────────────────────────────────────────────────────────
  prompt do usuário
       │
       ▼
  ┌──────────────────────────────────────────────────────────────┐
  │ 1. Bootstrap                                                 │
  │    • Load active.yaml (fonte única de config)                │
  │    • Resolve persona → tone_directive + phase_labels         │
  │    • SDD injection (se plugin ativo): override Sniper slot,  │
  │      base_path, knowledge_paths, governance_context          │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 2. Preflight                              stop on first fail  │
  │    2a. Load .strategist/index.yaml → load_always files       │
  │    2b. Load identity/what-i-am.yaml + drift-patterns.yaml    │
  │    2c. Resolve slot providers (roles/<config>.yaml)          │
  │        skill_root → .claude/skills → registry                │
  │    2d. Validate risk contracts:                              │
  │        Ranger   → write_pending                              │
  │        Archivist → write_analysis                            │
  │        Sniper   → controlled                                 │
  │    emit: phase=preflight status=done                         │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 3. Intake                                                    │
  │    invoke prompt-intake skill                                │
  │    → mission_contract: task_type, risk_level, constraints    │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 4. Context Enrichment                                        │
  │    invoke context-enrichment (knowledge.index.yaml + hints)  │
  │    invoke dossier-builder → dossier mínimo                   │
  └──────────────────────────┬───────────────────────────────────┘
                             │
FASES DA MISSÃO
─────────────────────────────────────────────────────────────────────
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 5a. RANGER (discovery slot)          contract: write_pending  │
  │     → pending/<mission_id>-discovery.md                      │
  │     emit: phase=<ranger_label> status=done                   │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 5b. Housekeeping Scan (Strategist interno — não é slot)      │
  │     scan: todo/ pending/ refined/                            │
  │     produz: side quest manifest                              │
  │     emit: phase=housekeeping_scan status=done side_quests=N  │
  └────────────┬────────────────────────┬──────────────────────--┘
  N=0          │                        │ N>0
  skip 5c/5d   │                        │
               │         ┌─────────────▼──────────────────────┐
               │         │ 5c. Mini Approval Gate (condicional)│
               │         │     STOP — aguarda resposta         │
               │         └─────────────┬──────────────────────┘
               │                       │ yes/select
               │         ┌─────────────▼──────────────────────┐
               │         │ 5d. SNIPER side quests              │
               │         │     contract: controlled            │
               │         │     mv todo→done / promote pending  │
               │         │     produz: side quest report       │
               │         │     falha: non-blocking             │
               │         └─────────────┬──────────────────────┘
               │                       │ side quest report
               └───────────────────────┤
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 5e. ARCHIVIST (refinement slot)    contract: write_analysis  │
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
  │    • vazio → plan_only (sem gate)                           │
  │    • interno → gate normal                                   │
  │    • externo → gate + aviso de escopo                       │
  │    STOP — aguarda resposta explícita                         │
  └────────────┬──────────────────────────┬──────────────────────┘
  no/decline   │                          │ yes/approve
  plan_only    │                          │
               │         ┌───────────────▼───────────────────┐
               │         │ 7. SNIPER (execution slot)        │
               │         │    contract: controlled            │
               │         │    input: refined plan (tasks.md)  │
               │         │    → done/<mission_id>-report.md   │
               │         │    emit: phase=<sniper_label> done  │
               │         └───────────────┬───────────────────┘
               │                         │
               └─────────────────────────┤
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 8. Learning Phase (non-blocking)                             │
  │    invoke response-critic → invoke learning-curator          │
  │    checkpoint ao usuário antes de qualquer escrita           │
  │    falha: não bloqueia o resultado                           │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 9. Mission Result                                            │
  │    status: completed | plan_only | blocked                   │
  │    artifacts: discovery, side_quest_report?, refined_plan,   │
  │              execution_report?                               │
  └──────────────────────────────────────────────────────────────┘
```
```

---

## Verification

```bash
grep -n "Scout\|Engineer\|Hunter\|scout\|engineer\|hunter" readme.md readme_detailed.md
```
Expected: zero results (except comments in code blocks that intentionally reference old names, if any).

```bash
grep -n "housekeeping_scan\|side_quest\|mini.*gate\|Mini.*Gate" readme_detailed.md
```
Expected: several results (new content added in T3/T4/T5).
