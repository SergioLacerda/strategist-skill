# Referência de Configuração — Strategist Skill

Todos os arquivos de configuração ficam em `.strategist/` dentro do repositório instalado. Edições manuais requerem recompilação (`strategist compile`) para que o agente use a versão atualizada.

---

## Estrutura de diretórios

```
.strategist/
  active.yaml                  Configuração principal da skill
  personas/
    pragmatic.yaml             Persona analítica
    epic.yaml                  Persona estratégica
  roles/
    default.yaml               Providers dos slots (gerado pelo install)
    mission.yaml               Exemplo de configuração de missão
    spec-driven.yaml           Configuração para fluxo spec-driven
  knowledge.index.yaml         Fontes de conhecimento por task_type
  memory/
    source-hints.yaml          Ajustes de prioridade aprendidos (learning loop)
  .compiled/                   Artefatos compilados (não editar manualmente)
    .config.gz
    .domain.gz
    .index.gz
    .manifest.gz
```

---

## active.yaml

Arquivo central que define o modo de operação e o binding de configuração.

**Schema:**

```yaml
mode: pragmatic | epic          # Persona ativa. Obrigatório.
base_path: .analysis            # Diretório onde artefatos de missão são escritos.
roles_config: default           # Nome do arquivo em roles/ (sem .yaml).
knowledge_index_path: .strategist/knowledge.index.yaml  # Caminho para o índice de conhecimento.
```

**Campos:**

| Campo | Tipo | Obrigatório | Padrão | Descrição |
|-------|------|-------------|--------|-----------|
| `mode` | string | sim | — | Persona ativa. Aceita `pragmatic` ou `epic`. |
| `base_path` | string | não | `.analysis` | Raiz onde `pending/`, `refined/` e `done/` são criados. |
| `roles_config` | string | sim | `default` | Nome do arquivo de roles a carregar de `roles/<nome>.yaml`. |
| `knowledge_index_path` | string | não | `.strategist/knowledge.index.yaml` | Caminho do knowledge index. |

**Exemplo gerado pelo install (modo pragmatic):**

```yaml
mode: pragmatic
base_path: .analysis
roles_config: default
knowledge_index_path: .strategist/knowledge.index.yaml
```

O `mode` pode ser sobrescrito por missão via parâmetro `--mode=epic` sem alterar este arquivo.

---

## roles/*.yaml

Define quais providers são usados em cada slot do pipeline. O install gera `roles/default.yaml`.

**Schema:**

```yaml
discovery: <provider_id>       # Provider do slot Ranger. risk_score deve ser write_pending.
refinement: <provider_id>      # Provider do slot Archivist. risk_score deve ser write_analysis.
execution: <provider_id>       # Provider do slot Sniper. risk_score deve ser controlled.
```

**Exemplo — roles/default.yaml:**

```yaml
# Slot keys: discovery, refinement, execution
# Nomes internos: Ranger, Archivist, Sniper (epic) / análise, refinamento, execução (pragmatic)
discovery: brainstorm
refinement: openspec-explore
execution: sdd-ask
```

**Resolução de provider:** o Strategist busca `<provider_id>/skill.yaml` nos caminhos configurados. Se não encontrar, para com `slot_provider_not_found`.

**Validação de risk_score:** cada slot tem um risk_score exigido. Mismatches param o pipeline com `slot_risk_mismatch`.

| Slot | risk_score exigido |
|------|-------------------|
| `discovery` | `write_pending` |
| `refinement` | `write_analysis` |
| `execution` | `controlled` |

**Roles disponíveis:**

| Arquivo | Uso |
|---------|-----|
| `default.yaml` | Configuração gerada pelo install (brainstorm + openspec-explore + sdd-ask) |
| `mission.yaml` | Configuração otimizada para missões de alta complexidade |
| `spec-driven.yaml` | Fluxo orientado a especificação (útil com SDD) |

---

## personas/*.yaml

Define o tom, vocabulário e templates de mensagem do agente. A persona não altera o pipeline — apenas o idioma e o estilo das saídas.

**Schema:**

```yaml
id: pragmatic | epic

phase_labels:                   # Rótulos exibidos nas mensagens de progresso
  discovery: <label>
  refinement: <label>
  execution: <label>

tone_directive: >               # Instrução de tom para o agente
  <texto livre>

progress_prefix: "[Strategist]" # Prefixo fixo em todos os eventos de progresso

prompt_templates:               # Templates interpolados em runtime
  intake_summary: >
    <template com {task_type}, {delivery_strategy}, ...>
  phase_start: >
    <template com {phase_label}, {provider}, {n}, {total}>
  phase_done: >
    <template com {phase_label}, {artifact_path}>
  approval_prompt: >
    <template com {artifact_path}>
  plan_only_result: >
    <template com {artifact_path}>
```

**Diferenças entre personas:**

| Aspecto | pragmatic | epic |
|---------|-----------|------|
| Fase de discovery | `análise` | `Ranger` |
| Fase de refinement | `refinamento` | `Archivist` |
| Fase de execution | `execução` | `Sniper` |
| Tom | Analítico, direto | Estratégico, autoritário |
| Ambiguidades | "questões abertas" | "riscos a resolver" |

---

## knowledge.index.yaml

Define fontes de conhecimento consultadas pelo `context-enrichment` antes de cada missão. Filtragem por `task_type`.

**Schema:**

```yaml
schema_version: "1"
description: >
  <descrição>

sources:
  - id: <identificador único>
    type: docs | examples | directives
    path: <caminho absoluto ou relativo ao repositório>
    tags: [<task_type>, ...]    # "all" faz a fonte ser carregada para qualquer task_type
    priority: high | medium | low
```

**Campos de source:**

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `id` | string | Identificador único. Referenciado em `source-hints.yaml`. |
| `type` | string | `docs` (documentação), `examples` (exemplos), `directives` (diretrizes) |
| `path` | string | Caminho para arquivo ou diretório. Diretórios são lidos recursivamente. |
| `tags` | string[] | `task_type` que ativam esta fonte. Use `all` para sempre incluir. |
| `priority` | string | Prioridade de carregamento. Ajustável via learning loop. |

**task_types reconhecidos:**

| task_type | Quando é usado |
|-----------|----------------|
| `architecture_analysis` | Análise de arquitetura de sistemas |
| `refactor` | Refatoração de código |
| `general` | Fallback quando nenhum padrão corresponde |

**Exemplo:**

```yaml
sources:
  - id: project-architecture
    type: docs
    path: /abs/path/to/docs/architecture
    tags: [architecture_analysis, architecture]
    priority: high

  - id: team-directives
    type: directives
    path: /abs/path/to/team-directives.md
    tags: [all]
    priority: high

  - id: past-good-examples
    type: examples
    path: .analysis/.strategist/patterns/good
    tags: [examples, refactor]
    priority: medium
```

Um `knowledge.index.yaml` vazio (sem `sources`) é válido — o pipeline continua sem enriquecimento de contexto.

---

## memory/source-hints.yaml

Ajustes de prioridade aprendidos pelo `learning-curator` após missões anteriores. Gerado e gerenciado automaticamente — não editar manualmente.

**Schema:**

```yaml
- source_id: <id da fonte em knowledge.index.yaml>
  annotation: <observação do curator>
  priority_adjustment: +1 | -1 | 0
  task_type: <task_type da missão que gerou este hint>
  recorded_at: <ISO 8601 timestamp>
```

Estes hints são aplicados sobre a prioridade base declarada em `knowledge.index.yaml` antes de cada missão.

---

## Parâmetros de missão (flags em runtime)

Passados diretamente ao invocar o Strategist, sem alterar arquivos de configuração:

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `mode` | `pragmatic \| epic` | Sobrescreve `active.yaml.mode` apenas para esta missão |
| `roles` | string | Nome do arquivo de roles a usar (padrão: `default`) |

---

## Validação

Use `strategist validate` para verificar a configuração antes de rodar uma missão:

```bash
strategist validate                     # valida .strategist/ no diretório atual
strategist validate --root=/outro/path  # valida uma raiz alternativa
```

Veja [cli-reference.md](cli-reference.md) para detalhes dos erros reportados.
