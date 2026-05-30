# Design — Qualidade e Consistência
**Mission ID:** 20260529-criticas
**Analysis group:** qualidade

---

## Modules Index

| Módulo | Arquivo | Papel no sistema |
|--------|---------|-----------------|
| Protocol doc | `strategist/protocol.md` | Regras mandatórias de roteamento; define vocabulário de `risk_score` para stop conditions |
| Skill instructions | `.strategist/SKILL.md` | Instrução canônica do agente; fonte de verdade do vocabulário de `risk_score` |
| Intake schema | `.strategist/schemas/intake.schema.yaml` | Define campos e aliases de `mission_contract`; local correto para spec de `mission_id` |
| README principal | `readme.md` | Entry point público; renderizado pelo GitHub |
| README detalhado | `readme_detailed.md` | Referência técnica; deve documentar todas as fases do pipeline |
| Changelog | `CHANGELOG.md` | **NOVO** — histórico de versões |
| Curl installer | `bootstrap.sh` | Resolve versão e baixa tarball; `resolve_ref()` faz chamada à GitHub API |
| Memory policy | `.strategist/memory/policy.yaml` | **NOVO** — define limites e política de rotation do `outcomes.jsonl` |

### Boundaries

- `SKILL.md` é a fonte de verdade para vocabulário do agente. `protocol.md` é a documentação do protocolo mandatório para implementadores externos. **Boundary inconsistente identificada:** os dois documentos definem o mesmo vocabulário de `risk_score` com termos diferentes. Não há mecanismo de sincronização — divergência silenciosa.
- `intake.schema.yaml` define o schema de intake do agente mas não define `mission_id`, que aparece em `SKILL.md`, `progress-contract.yaml` e em paths de artefatos. **Boundary ambígua:** `mission_id` cruza múltiplos módulos sem um ponto de definição.
- `readme.md` e `readme_detailed.md` têm escopo superposto. **Boundary ausente:** não há documentado qual é o público alvo ou escopo de cada um.

---

## Item 1 — Vocabulário risk_score

### Diagnóstico

Confirmado por leitura direta dos dois arquivos:

| Arquivo | Vocabulário usado |
|---------|-----------------|
| `.strategist/SKILL.md` §2d | `write_pending`, `write_analysis`, `controlled` |
| `strategist/protocol.md` §Stop Conditions | `read_only`, `controlled_write` |

Os dois vocabulários não têm overlap — são termos completamente diferentes para o mesmo conceito.

### Decisão de design

`.strategist/SKILL.md` é a fonte de verdade do agente. `strategist/protocol.md` é o documento de referência para implementadores de slot providers. O correto é atualizar `protocol.md` para usar o vocabulário canônico de SKILL.md.

```
Vocabulário canônico (SKILL.md):
  Discovery slot   → write_pending
  Refinement slot  → write_analysis
  Execution slot   → controlled

Em protocol.md, na tabela de Stop Conditions, linha slot_risk_mismatch:
  ANTES: "risk_score other than read_only [...] controlled_write"
  DEPOIS: "risk_score other than write_pending (discovery), write_analysis (refinement),
           or controlled (execution)"
```

**Arquivo afetado:**
- `strategist/protocol.md`: atualizar descrição de `slot_risk_mismatch` e qualquer outra ocorrência de `read_only` / `controlled_write` para o vocabulário canônico.

---

## Item 2 — readme_detailed.md: housekeeping_scan

### Diagnóstico

A fase `housekeeping_scan` (SKILL.md §5b) e o mini approval gate (§5c) são fases importantes do pipeline — têm comportamento visível ao usuário (apresenta uma lista de side quests, requer aprovação antes de mover arquivos). Não aparecem em `readme_detailed.md`.

### Decisão de design

Adicionar uma seção "Housekeeping Scan" em `readme_detailed.md` descrevendo:
- Quando ocorre (entre Ranger e Archivist)
- O que verifica (todo/ → done? pending/ → refined? refined/ → done?)
- O mini approval gate: o que é, como responder (yes / no / select)
- O side quest report que Sniper produz

**Arquivo afetado:**
- `readme_detailed.md`: nova seção no pipeline diagram / phases description

---

## Item 3 — CHANGELOG

### Decisão de design

Criar `CHANGELOG.md` na raiz do projeto seguindo o formato [Keep a Changelog](https://keepachangelog.com/). Reconstruir entradas a partir do git log (18 commits existentes). Versão inicial: `1.0.0`.

Estrutura:
```
## [Unreleased]
## [1.0.0] - 2026-05-28
### Added
  - curl installer (bootstrap.sh / bootstrap.ps1)
  - housekeeping_scan phase with side quest pipeline
  - slot write contracts (write_pending / write_analysis)
  - validate_provider() in install wizard
  - GitHub Actions SHA-pinning
  [...]
```

**Arquivo afetado:**
- `CHANGELOG.md` (NOVO)

---

## Item 4 — Hierarquia dos READMEs

### Diagnóstico

`readme.md` é o entry point (GitHub o renderiza por padrão). `readme_detailed.md` existe mas não há referência entre eles.

### Decisão de design

Manter ambos os arquivos (não consolidar — a separação entre visão geral e referência técnica é útil). Adicionar em `readme.md` um link explícito para `readme_detailed.md` com escopo declarado:

```markdown
**Referência técnica completa:** [readme_detailed.md](readme_detailed.md) —
pipeline detalhado, fases, schemas, configuração de providers.
```

**Arquivo afetado:**
- `readme.md`: adicionar seção ou link de referência cruzada

---

## Item 5 — mission_id: Formato Canônico

### Diagnóstico

O campo `mission_id` aparece em:
- `SKILL.md`: paths de artefatos (`<mission_id>-discovery.md`)
- `progress-contract.yaml`: `output_paths` (`<mission_id>-plan.md`)
- Missão atual: `20260529-criticas` (formato ad hoc)

Nenhum arquivo define o formato gerado.

### Decisão de design

Definir o formato em `intake.schema.yaml` como um campo adicional de documentação (não é um constraint de intake, mas é o local correto por coesão com o schema de missão):

```yaml
mission_id:
  description: Identificador único gerado no intake. Não é um constraint — é documentação do formato.
  format: "YYYYMMDD-<slug>"
  slug_rules:
    - Derivado das primeiras palavras-chave do prompt do usuário
    - Máximo 20 caracteres
    - Apenas [a-z0-9-]
    - Gerado pelo prompt-intake skill
  collision_policy: >
    Arquitetura serial por design — colisão é improvável. Se detectada
    (artefato já existe no path), appender sufixo numérico: YYYYMMDD-<slug>-2.
  known_limitation: >
    Race conditions em missões paralelas não são endereçadas. Arquitetura
    assume execução single-user serial. Ver robustez/parallelism para contexto.
```

**Arquivo afetado:**
- `.strategist/schemas/intake.schema.yaml`: adicionar bloco `mission_id` como documentação

---

## Item 6 — Estratégia de Retry para Slots

### Diagnóstico

O `protocol.md` atual trata todas as falhas de slot como definitivas: "If discovery slot fails: stop." Não há distinção entre falha transiente (timeout de rede, LLM temporariamente indisponível) e falha permanente (configuração inválida, contrato violado).

### Decisão de design

Adicionar a distinção no `protocol.md` sem alterar o comportamento default (still stop):

```markdown
## Slot Failure Classification

Falhas de slot têm dois tipos:

| Tipo | Exemplos | Comportamento |
|------|----------|--------------|
| Transient | timeout de rede, API rate limit, LLM unavailable | Strategist PODE re-invocar o slot uma vez após 0s delay. Se falhar novamente: stop. |
| Permanent | contrato violado, output inválido, slot_risk_mismatch | Strategist para imediatamente. Retry não autorizado. |

O slot provider é responsável por indicar o tipo de falha no campo `failure_type`
do seu output. Se `failure_type` estiver ausente, tratar como permanent.
```

Esta mudança é aditiva ao `protocol.md` e requer que o `slot-output.schema.yaml` (criado na análise de segurança) inclua `failure_type` como campo opcional.

**Arquivo afetado:**
- `strategist/protocol.md`: adicionar seção "Slot Failure Classification"
- Nota: coordenação com `.strategist/schemas/slot-output.schema.yaml` da análise seguranca-testes (item D1)

---

## Item 7 — outcomes.jsonl: Política de Rotation

### Decisão de design

Criar `.strategist/memory/policy.yaml` documentando limites e política de pruning:

```yaml
outcomes_jsonl:
  max_entries: 500
  max_size_kb: 256
  rotation_policy: >
    Quando max_entries ou max_size_kb for excedido, learning-curator deve
    apresentar checkpoint ao usuário antes de qualquer pruning. Pruning padrão:
    remover as entradas mais antigas até ficar abaixo de 80% do limite.
  manual_pruning_command: >
    Não implementado automaticamente. Usuário pode truncar manualmente ou
    invocar learning-curator com --prune.
```

**Arquivo afetado:**
- `.strategist/memory/policy.yaml` (NOVO)
- `.strategist/index.yaml`: adicionar `memory/policy.yaml` sob `load_on_demand`

---

## Item 8 — Shell Script: Verificação de Dependências

### Decisão de design

Adicionar uma função `require_cmd` no início de `bootstrap.sh` que verifica a presença de cada dependência antes de usar:

```bash
require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "[Strategist] Erro: '$1' não encontrado. Instale-o antes de continuar." >&2
    exit 1
  }
}
# chamadas antes de qualquer uso:
require_cmd curl
require_cmd tar
require_cmd sha256sum  # necessário após item A2 da análise segurança
```

**Arquivo afetado:**
- `bootstrap.sh`: adicionar função `require_cmd` e chamadas no início do script

---

## Item 9 — GitHub API Rate Limit em resolve_ref()

### Diagnóstico

`resolve_ref()` faz `curl` para `api.github.com/repos/.../releases/latest` sem autenticação. Em CI com muitas execuções, o rate limit (60 req/hora para IP público) pode ser atingido. A função atual captura a falha com `|| true` e cai silenciosamente para `$DEFAULT_REF` (`main`):

```bash
latest="$(curl ... 2>/dev/null ... )" || true
```

### Decisão de design

Distinguir entre "sem release" (legítimo) e "erro de API" (rate limit, rede):

```bash
resolve_ref() {
  if [ -n "$REF" ]; then echo "$REF"; return; fi

  local http_status response
  response="$(curl -fsSL -w "\n%{http_code}" \
    "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null)" || true
  http_status="$(echo "$response" | tail -1)"
  body="$(echo "$response" | head -n -1)"

  case "$http_status" in
    200)
      local tag
      tag="$(echo "$body" | grep '"tag_name"' | head -1 | sed '...')"
      [ -n "$tag" ] && echo "$tag" && return
      ;;
    404) ;; # sem releases — fallback legítimo
    403|429)
      echo "[Strategist] WARN: GitHub API rate limit atingido. Usando ${DEFAULT_REF}." >&2
      ;;
    *)
      echo "[Strategist] WARN: GitHub API retornou HTTP ${http_status}. Usando ${DEFAULT_REF}." >&2
      ;;
  esac

  echo "[Strategist] No release found, using branch: ${DEFAULT_REF}" >&2
  echo "$DEFAULT_REF"
}
```

**Arquivo afetado:**
- `bootstrap.sh`: reescrever `resolve_ref()` com tratamento de status HTTP

---

## Dependency Map

```
Item 1 (vocabulário)     → protocol.md                 — independente
Item 2 (housekeeping doc)→ readme_detailed.md           — independente
Item 3 (changelog)       → CHANGELOG.md (novo)          — independente
Item 4 (readme hierarquia)→ readme.md                   — independente
Item 5 (mission_id)      → intake.schema.yaml           — independente
Item 6 (retry strategy)  → protocol.md + coordena com
                           slot-output.schema.yaml       — coordena com análise seguranca item D1
Item 7 (rotation policy) → memory/policy.yaml (novo)    — independente
Item 8 (require_cmd)     → bootstrap.sh                 — independente; item 9 é no mesmo arquivo
Item 9 (rate limit)      → bootstrap.sh                 — independente; implementar junto com item 8
```

Todos os itens são independentes entre si, exceto item 6 que coordena com a análise `seguranca-testes` item D1 (`slot-output.schema.yaml`). Recomendação: implementar item 6 após a análise de segurança estar concluída.
