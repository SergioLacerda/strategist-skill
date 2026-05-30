# Tasks — Qualidade e Consistência
**Mission ID:** 20260529-criticas
**Analysis group:** qualidade

---

## Checklist de Implementação

### Bloco F — Vocabulário risk_score

- [ ] **F1.** Em `strategist/protocol.md`, localizar a linha `slot_risk_mismatch` na tabela de Stop Conditions. Atualizar a descrição de `"risk_score other than read_only"` para `"risk_score other than write_pending (discovery) ou write_analysis (refinement)"`, e `"controlled_write"` para `"controlled"`. Verificar se há outras ocorrências de `read_only` ou `controlled_write` no arquivo e atualizar para o vocabulário canônico.

### Bloco G — Documentação do housekeeping_scan

- [ ] **G1.** Em `readme_detailed.md`, adicionar seção "Housekeeping Scan (Phase 5b–5d)" após a descrição da fase de discovery (Ranger). A seção deve descrever: o que é scaneado (`todo/`, `pending/`, `refined/`), a lógica de cada verificação, o mini approval gate (o que é apresentado, como responder: yes / no / select), e o side quest report produzido pelo Sniper.

### Bloco H — CHANGELOG

- [ ] **H1.** Criar `CHANGELOG.md` na raiz do projeto. Usando `git log --oneline` como fonte, reconstruir as entradas dos 18 commits existentes agrupados sob `[1.0.0] - 2026-05-28`. Usar formato Keep a Changelog com seções `Added`, `Changed`, `Fixed`. Adicionar cabeçalho `[Unreleased]` vazio acima para uso futuro.

### Bloco I — Hierarquia dos READMEs

- [ ] **I1.** Em `readme.md`, adicionar logo após a introdução/overview uma linha de referência explícita: "Para o pipeline detalhado, fases, schemas e configuração de providers: [readme_detailed.md](readme_detailed.md)."

### Bloco J — mission_id Canônico

- [ ] **J1.** Em `.strategist/schemas/intake.schema.yaml`, adicionar ao final do arquivo um bloco `mission_id:` com os campos: `description`, `format` (`"YYYYMMDD-<slug>"`), `slug_rules` (derivado do prompt, máx 20 chars, `[a-z0-9-]`), `collision_policy` (appender sufixo `-2`, `-3`, etc.), e `known_limitation` (arquitetura serial, missões paralelas não endereçadas).

### Bloco K — Estratégia de Retry

- [ ] **K1.** Em `strategist/protocol.md`, adicionar seção "Slot Failure Classification" após a seção "Slot Failure Handling". A seção deve definir dois tipos de falha (`transient` e `permanent`), exemplos de cada, e o comportamento para cada tipo: transient permite uma re-invocação automática; permanent para imediatamente. Indicar que o slot provider deve declarar `failure_type` no output, e que ausência de `failure_type` é tratada como `permanent`.

- [ ] **K2.** *(Coordenação)* Anotar no design de `seguranca-testes` item D1 (`slot-output.schema.yaml`) que o campo `failure_type: transient | permanent` deve ser incluído como campo opcional. Esta task só pode ser marcada concluída após a análise `seguranca-testes` estar executada.

### Bloco L — Política de Rotation do outcomes.jsonl

- [ ] **L1.** Criar `.strategist/memory/policy.yaml` com os campos: `outcomes_jsonl.max_entries` (500), `outcomes_jsonl.max_size_kb` (256), `outcomes_jsonl.rotation_policy` (descrição em prosa do comportamento de pruning e checkpoint obrigatório com usuário), `outcomes_jsonl.manual_pruning_command` (nota sobre ausência de automação atual).

- [ ] **L2.** Em `.strategist/index.yaml`, adicionar `memory/policy.yaml` sob a chave `load_on_demand`.

### Bloco M — Shell Script: Melhorias

- [ ] **M1.** Em `bootstrap.sh`, adicionar função `require_cmd()` (verifica `command -v $1`, emite erro e exit 1 se ausente) antes do bloco de arg parsing. Adicionar chamadas `require_cmd curl`, `require_cmd tar`, `require_cmd sha256sum` (este último necessário após tarefa A2 da análise seguranca-testes) no início do script, antes de qualquer uso dessas ferramentas.

- [ ] **M2.** Em `bootstrap.sh`, reescrever `resolve_ref()` para capturar o HTTP status code da chamada à GitHub API (usando `curl -w "%{http_code}"`). Para status 403 e 429, emitir aviso explícito de rate limit. Para status inesperado (não 200 nem 404), emitir aviso com o código. Manter o fallback para `$DEFAULT_REF` em todos os casos de erro, mas com mensagem de aviso clara.

---

## Ordem de Implementação Recomendada

1. **Bloco F** (vocabulário) — único item que afeta terceiros implementando slot providers; menor esforço, maior impacto
2. **Bloco I** (README hierarquia) — uma linha, elimina ambiguidade imediata
3. **Bloco H** (CHANGELOG) — reconstrução a partir de git log; autônomo
4. **Bloco G** (housekeeping doc) — documenta fase existente; moderado esforço
5. **Bloco J** (mission_id) — adiciona ao schema existente
6. **Bloco M** (shell improvements) — M1 e M2 independentes, implementar juntos
7. **Bloco L** (rotation policy) — novo arquivo + index update
8. **Bloco K** (retry strategy) — K1 independente; K2 aguarda análise `seguranca-testes`

---

## Critérios de Conclusão

- `strategist/protocol.md` não contém mais os termos `read_only` ou `controlled_write`
- `readme_detailed.md` descreve as fases 5b–5d com o mini approval gate documentado
- `CHANGELOG.md` existe na raiz com todas as versões documentadas
- `readme.md` tem referência explícita para `readme_detailed.md`
- `.strategist/schemas/intake.schema.yaml` tem bloco `mission_id` com formato canônico
- `bootstrap.sh` exibe aviso claro de rate limit quando a GitHub API retorna 403/429
- `bootstrap.sh` falha explicitamente se `curl`, `tar` ou `sha256sum` não estiverem no PATH
- `.strategist/memory/policy.yaml` existe com limites definidos
