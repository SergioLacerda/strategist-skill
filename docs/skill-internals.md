# Internals da Skill — Sub-skills, Contratos e Schemas

**Status:** Accepted
**Last Updated:** 2026-06-26

Este documento descreve os componentes internos do runtime da skill Strategist: as sub-skills invocadas automaticamente pelo orchestrador, os contratos de fase, e os schemas de entrada/saída.

Para o pipeline geral e comportamento dos slots, consulte `docs/architecture.md`.
Para a ordem canônica de leitura dos contratos, consulte `strategist/SKILL.md` e `docs/adr/0010-ordered-contracts-and-mission-observability.md`.
Para configuração, veja [configuration.md](configuration.md).

---

## Sub-skills internas

O Strategist invoca 6 sub-skills internas em cada missão. Todas têm `risk_score: read_only` — não escrevem em disco diretamente, exceto `learning-curator` (com aprovação obrigatória).

### prompt-intake

**Categoria:** classificação  
**Quando:** antes do pipeline, logo após o bootstrap

Classifica o prompt do usuário em `task_type`, `risk_level` e extrai as restrições de missão (`delivery_strategy`, `legacy_compatibility`, `execution_intent`).

**Entrada:**
- `user_prompt` — texto livre do usuário
- `intake_schema_path` — caminho para `schemas/intake.schema.yaml`

**Saída:**
- `task_type` — tipo da tarefa (ex: `architecture_analysis`, `refactor`, `general`)
- `risk_level` — `low`, `medium` ou `high`
- `constraints` — objeto com os 3 campos de restrição

**Comportamento especial:** se dois aliases mutuamente exclusivos para o mesmo campo forem detectados no prompt, retorna `conflict=true` com o campo conflitante. O pipeline para e pede ao usuário que resolva o conflito antes de prosseguir.

---

### context-enrichment

**Categoria:** conhecimento  
**Quando:** após prompt-intake, antes de discovery

Consulta `knowledge.index.yaml` pelo `task_type` da missão. Aplica ajustes de `source-hints.yaml`. Retorna excerpts ranqueados dentro do token budget configurado.

**Entrada:**
- `task_type` — da saída do prompt-intake
- `token_budget` — número máximo de tokens para excerpts
- `knowledge_index_path` — caminho do index
- `source_hints_path` — caminho de `memory/source-hints.yaml`

**Saída:**
- `excerpts` — lista ranqueada de excerpts (highest priority first)
- `rubric` — rubrica do task_type (de `.strategist/rubrics/`) ou `null`
- `sources_queried` / `sources_matched` — contadores

**Resultado vazio é válido:** se nenhuma fonte corresponder ao `task_type`, retorna `excerpts: []` e o pipeline continua normalmente.

Prioridade efetiva = prioridade declarada no index + `priority_adjustment` do source-hints.

---

### dossier-builder

**Categoria:** assembly  
**Quando:** após context-enrichment, antes de discovery

Monta o dossier que é passado aos slot providers como contexto de conhecimento. Garante que o dossier não exceda o token budget e nunca inclui os arquivos de identidade brutos (`what-i-am.yaml`, `drift-patterns.yaml`).

**Entrada:**
- `task_type`
- `enrichment_output` — saída do context-enrichment
- `identity_files` — `what-i-am.yaml` + `drift-patterns.yaml` (se disponíveis)
- `token_budget`

**Saída — estrutura do dossier:**

```yaml
task_type: string
directives: string | null
good_examples: array          # máximo 2 itens
bad_examples: array           # máximo 1 item
rubric: object | null
output_template: string | null
token_count: integer
```

**Ordem de corte quando budget é ultrapassado:** bad_examples → good_examples (mantém o de maior score) → directives. `task_type` e `output_template` nunca são cortados.

---

### ranger (slot de discovery)

**Categoria:** discovery  
**Quando:** fase de discovery (slot configurável)

Produz o artefato canônico de análise que abre a missão formalmente para o usuário e serve de base direta para o Archivist.

**Entrada:**
- `user_prompt`
- `mission_contract`
- `dossier`
- `treasure_chests`

**Saída:**
- `analysis_artifact_path` — `<base_path>/pending/<mission_id>-analysis.md`

**Contrato obrigatório no artefato:**
- `mission_id`
- `objective`
- `analysis_summary`
- `known_facts`
- `uncertainties`
- `recommended_refinement_focus`

---

### archivist (slot de refinement)

**Categoria:** refinamento  
**Quando:** fase de refinement (slot configurável)

Lê o artefato de discovery e produz um plano revisado e implementável. É o provider padrão do slot `refinement`.

**Entrada:**
- `analysis_artifact_path` — caminho para o artefato canônico do Ranger em `refined/`
- `base_path` — diretório base da missão
- `mission_contract` — `planning_rules` extraído pelo prompt-intake

**Saída:**
- `analysis.md` — `<base_path>/refined/<mission_id>/analysis.md`
- `proposal.md` — `<base_path>/refined/<mission_id>/proposal.md`
- `design.md` — `<base_path>/refined/<mission_id>/design.md`
- `tasks.md` — `<base_path>/refined/<mission_id>/tasks.md`

O output canônico do refinement é um pacote de quatro artefatos. `refined/<mission_id>-plan.md` é drift histórico, não o contrato atual.

Após escrever os quatro artefatos, o Archivist executa a **Opportunity Attack** (avaliação ADR): verifica se os artefatos refinados justificam a abertura de um ADR. Essa avaliação é interna ao Archivist — não é delegada a slot.

---

### response-critic

**Categoria:** avaliação  
**Quando:** fase de learning (não-bloqueante)

Avalia a saída do slot contra a rubrica do `task_type`. Produz score e lista de gaps — alimenta o `learning-curator`.

**Entrada:**
- `slot_output` — conteúdo do artefato de saída do slot
- `task_type`
- `rubric` — do context-enrichment; se `null`, retorna `result=no_rubric`

**Saída:**
- `result` — `pass`, `fail` ou `no_rubric`
- `score` — 0.0–1.0 (null quando `no_rubric`)
- `must_have_present` / `must_have_missing` — itens da rubrica encontrados/ausentes
- `must_not_present` — itens proibidos encontrados (violações)

`result=pass` quando `score >= rubric.score_threshold` E `must_not_present` está vazio.

---

### learning-curator

**Categoria:** aprendizado  
**Quando:** fase de learning, após execution (não-bloqueante)

Propõe entradas para `memory/outcomes.jsonl` e `memory/source-hints.yaml`. **Não escreve nada sem aprovação explícita do usuário.**

**Entrada:**
- `mission_result` — resultado da missão
- `critic_evaluation` — saída do response-critic
- `task_type`
- `outcomes_path` e `source_hints_path`

**Checkpoint obrigatório:**
```
Learning checkpoint:
1. Record mission outcome? [mission_id / task_type / score / status]
   (yes / no)
2. Adjust source priority? [source_id / annotation / adjustment]
   (yes / no)
```

Aprovação é independente para cada item — o usuário pode aprovar outcomes e rejeitar source hints (e vice-versa).

**Falha na fase de learning nunca bloqueia o resultado da missão.** Se o checkpoint expirar ou a fase falhar, nada é escrito e a missão retorna normalmente.

---

## Contratos de Fase

Os contratos em `.strategist/contracts/` definem o contrato formal de cada fase interna do orchestrador.

### Sinais funcionais no pipeline único

`quick_draw`, `opportunity_attack`, `critical_hit`, side quests e `treasure_chests`
não abrem pipelines paralelos. Eles se encaixam no fluxo único
`Ranger -> Archivist -> approval gate -> Sniper`.

- **Quick Draw**: captura rápida de ideia/TODO via rota dedicada; escreve somente após gate; `todo/` é write-only do ponto de vista da skill.
- **Opportunity Attack**: avaliação ADR executada pelo Archivist após escrever os quatro artefatos refinados. Não é delegada a slot.
- **Critical Hit**: rota de gerenciamento de artefatos de análise (`.md`) dentro das pastas `pending/`, `refined/` e `archived/` do `<base_path>`.
- **Side Quests**: observações de escopo detectadas durante qualquer fase; Ranger, Archivist e Sniper podem detectar; Archivist consolida no gate; Sniper reporta side quests recém-descobertas.

Guardrail principal: nenhuma materialização aprovada ocorre sem aprovação no gate da missão.
A restrição de escopo aplica-se somente para impedir materialização documental fora do escopo aprovado.

### Treasure Chests (baú do tesouro)

`treasure_chests` são fontes de conhecimento offline declaradas em `active.yaml`.
O Strategist só repassa ao slot os baús com escopo compatível:

- `discovery` → Ranger
- `refinement` → Archivist
- `execution` → Sniper
- `all` → todos os slots

Ausência de baú aplicável não bloqueia a missão.

### bootstrap

Carrega a configuração ativa (`active.yaml`, persona, roles) antes de qualquer missão.

| | |
|-|-|
| **Entradas** | `skill_root`, `mode_override` (opcional), `roles_override` (opcional) |
| **Saídas** | `active`, `persona`, `roles`, `sdd_injection` (opcional) |
| **Fast path** | `.strategist/.compiled/.config.gz` — se fresco, carrega o artefato compilado diretamente |
| **Fallback** | Se `.config.gz` estiver corrompido: carrega YAML diretamente, emite `bootstrap=standard_path` |

Erros que param: `active_yaml_not_found`, `persona_not_found`.

### preflight

Valida providers dos slots e carrega o domínio interno. Roda após bootstrap, antes do intake.

| | |
|-|-|
| **Entradas** | `active`, `persona`, `roles` |
| **Saídas** | `domain`, `slot_providers`, `preflight_status` |
| **Fast path** | `.strategist/.compiled/.domain.gz` |
| **Write scope** | Read-only |

Erros que param: `slot_provider_not_found`, `slot_risk_mismatch`.  
`index_yaml_not_found` é não-bloqueante — pipeline continua sem domínio interno.

### Demais contratos

| Contrato | O que garante |
|----------|--------------|
| `check-stale.yaml` | Formato e comportamento do check de staleness |
| `compile-config.yaml` | Fontes e schema do `.config.gz` |
| `compile-domain.yaml` | Fontes e schema do `.domain.gz` |
| `compile-knowledge-index.yaml` | Fontes e schema do `.index.gz` |
| `compile-all.yaml` | Sequência e dependências da compilação completa |
| `context-enrichment.yaml` | Contrato de entrada/saída do context-enrichment |
| `learning-buffer.yaml` | Comportamento do buffer de outcomes (tamanho máximo, flush) |
| `learning-curator.yaml` | Checkpoint obrigatório antes de escrever em memory/ |
| `preflight.yaml` | Validação de slots e carregamento do domínio |
| `bootstrap.yaml` | Carregamento de active.yaml, persona e roles |

---

## Schemas

### intake.schema.yaml

Define os campos de restrição reconhecidos pelo `prompt-intake` e seus aliases em linguagem natural.

**Campos:**

| Campo | Default | Valores aceitos |
|-------|---------|-----------------|
| `delivery_strategy` | `sprint` | `sprint`, `total` |
| `legacy_compatibility` | `required` | `required`, `not_required` |
| `execution_intent` | `review_only` | `review_only`, `execute` |

**Aliases por valor:**

`delivery_strategy: sprint` → "por sprint", "faseado", "iterativo", "incremental", "fase a fase", "entrega faseada"  
`delivery_strategy: total` → "big bang", "sem prazo", "entrega total", "tudo de uma vez"

`legacy_compatibility: required` → "retrocompatível", "backwards compatible", "sem breaking changes", "não pode quebrar"  
`legacy_compatibility: not_required` → "pode quebrar", "breaking ok", "clean break"

`execution_intent: execute` → "executar", "implementar", "aplicar", "rodar", "fazer"  
`execution_intent: review_only` → "só análise", "sem execução", "apenas revisar", "só plano"

---

## Métricas de performance e baseline

As métricas canônicas para otimização de performance do Strategist são:

- `t_start_to_intake_ms`
- `t_intake_to_ranger_ms`
- `total_wall_time_ms`
- `tokens_in`
- `tokens_out`
- `lines_emitted`

O baseline atual está registrado em `docs/performance-baseline.md` e deve ser atualizado sempre que houver mudança de contrato, telemetria ou política de emissão que possa alterar custo percebido.

As métricas são expostas pelo sinal de saída `mission_metrics` no checkpoint de intake e em cada transição de fase. Isso mantém a telemetria de custo disponível sem alterar a ordem visível do pipeline.

`confidence_threshold: 0.65` — aliases com confiança abaixo deste valor recebem o default.

Define o formato obrigatório dos eventos de progresso emitidos pelo Strategist em cada transição de fase.

**Formato:**
```
[Strategist] phase=<phase_label> status=<status> [campos adicionais]
```

**Statuses:**

| Status | Campos obrigatórios | Quando |
|--------|---------------------|--------|
| `running` | `phase`, `status`, `skill`, `checklist` | Fase iniciou |
| `done` | `phase`, `status`, `artifact` | Fase completou com sucesso |
| `blocked` | `phase`, `status`, `reason`, `action` | Fase não pode continuar |
| `analysis_delivered` | `phase`, `status` | Missão entregou análise/refinamento sem materialização |

**Exemplos:**
```
[Strategist] phase=preflight status=done slots=ok
[Strategist] phase=discovery status=running skill=brainstorm checklist=0/3
[Strategist] phase=discovery status=done artifact=.analysis/pending/abc123-analysis.md
[Strategist] phase=approval_gate status=blocked reason=user_declined action=none
[Strategist] phase=execution status=done artifact=.analysis/archived/abc123-report.md
```

**Caminhos de artefatos:**

| Fase | Caminho |
|------|---------|
| discovery | `<base_path>/pending/<mission_id>-analysis.md` |
| refinement | `<base_path>/refined/<mission_id>/` |
| execution | `<base_path>/archived/<mission_id>-report.md` |

## OTEL e contrato rico

O contrato rico de telemetria agora cobre:

- `phase`
- `status`
- `component`
- `mission_id`
- `artifact_path`
- `selected_skill`
- `runtime_mode`
- `output_profile`
- `gate.type`
- `gate.status`
- `gate.response`
- `transition_group`
- `reason`

O namespace canônico fica em `internal/telemetry/schema.go` e o shape-alvo está em `strategist/schemas/telemetry-event.schema.yaml`.

Os `phase_labels` (Ranger/Archivist/Sniper vs análise/refinamento/execução) são resolvidos da persona ativa em runtime — o schema define apenas os campos obrigatórios, não os valores dos labels.

---

## Write Scopes dos Slots

Cada slot tem um escopo de escrita declarado no `skill.yaml`. Escrever fora do escopo para a missão com `slot_write_scope_violation`.

| Slot | Escopo de escrita | Tipos permitidos |
|------|------------------|-----------------|
| `discovery` | `<base_path>/` | `.md` |
| `refinement` | `<base_path>/` e `<base_path>/refined/` | `.md` |
| `execution` | `<base_path>/archived/` e documentação `.md` aprovada | `.md` |

---

## Fixtures de Teste de Segurança

Os fixtures em `strategist/tests/fixtures/` representam cenários de violação dos invariantes de segurança. São usados pelos testes de formato (`tests/fixtures_test.go`) e servem como documentação executável dos comportamentos proibidos.

| Fixture | Invariante testado |
|---------|-------------------|
| `approval-bypass.yaml` | Invocação do execution slot sem aprovação |
| `side-quest-bypass.yaml` | Side quest executada sem passar pelo approval gate |
| `slot-risk-mismatch.yaml` | Provider com risk_score incorreto para o slot |
| `discovery-failed.yaml` | Prosseguir para refinement após falha no discovery |
| `yaml-null-field.yaml` | Campo YAML nulo em posição obrigatória |
