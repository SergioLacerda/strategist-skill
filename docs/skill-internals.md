# Internals da Skill — Sub-skills, Contratos e Schemas

Este documento descreve os componentes internos do runtime da skill Strategist: as sub-skills invocadas automaticamente pelo orchestrador, os contratos de fase, e os schemas de entrada/saída.

Para o pipeline geral e comportamento dos slots, veja [readme_detailed.md](../readme_detailed.md).  
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

### archivist (slot de refinement)

**Categoria:** refinamento  
**Quando:** fase de refinement (slot configurável)

Lê o artefato de discovery e produz um plano revisado e implementável. É o provider padrão do slot `refinement`.

**Entrada:**
- `discovery_artifact_path` — caminho para o artefato do Ranger em `pending/`
- `base_path` — diretório base da missão
- `mission_contract` — `planning_rules` extraído pelo prompt-intake

**Saída:**
- `reviewed_plan_path` — `<base_path>/refined/<mission_id>-plan.md`

**Seções obrigatórias no plano de saída:**
- `executive_summary`
- `tasks_with_subitems`
- `technical_details`
- `modules_documents_index`
- `design` (context, goals, non_goals, do, do_not)
- `execution_checklist`
- `hunter_instructions`

Se uma seção não puder ser preenchida com evidências do artefato de discovery, é marcada com `[INSUFFICIENT EVIDENCE]` + nota de bloqueio. Especulação não fundamentada é proibida.

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

Os contratos em `strategist/contracts/` definem o contrato formal de cada fase interna do orchestrador.

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

`confidence_threshold: 0.65` — aliases com confiança abaixo deste valor recebem o default.

### progress-contract.yaml

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
| `plan_only` | `phase`, `status` | Missão parou no approval gate |

**Exemplos:**
```
[Strategist] phase=preflight status=done slots=ok
[Strategist] phase=discovery status=running skill=brainstorm checklist=0/3
[Strategist] phase=discovery status=done artifact=.analysis/pending/abc123-discovery.md
[Strategist] phase=approval_gate status=blocked reason=user_declined action=none
[Strategist] phase=execution status=done artifact=.analysis/done/abc123-report.md
```

**Caminhos de artefatos:**

| Fase | Caminho |
|------|---------|
| discovery | `<base_path>/pending/<mission_id>-discovery.md` |
| refinement | `<base_path>/refined/<mission_id>-plan.md` |
| execution | `<base_path>/done/<mission_id>-report.md` |

Os `phase_labels` (Ranger/Archivist/Sniper vs análise/refinamento/execução) são resolvidos da persona ativa em runtime — o schema define apenas os campos obrigatórios, não os valores dos labels.

---

## Write Scopes dos Slots

Cada slot tem um escopo de escrita declarado no `skill.yaml`. Escrever fora do escopo para a missão com `slot_write_scope_violation`.

| Slot | Escopo de escrita | Tipos permitidos |
|------|------------------|-----------------|
| `discovery` | `<base_path>/pending/` | `.md` |
| `refinement` | `<base_path>/` e `<base_path>/refined/` | `.md` |
| `execution` | Declarado pelo provider (`controlled`) | definido pelo provider |

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
