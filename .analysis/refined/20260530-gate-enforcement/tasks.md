# Tasks — Gate Enforcement
**Mission ID:** 20260530-gate-enforcement
**Design:** `.analysis/pending/2026-05-30-gate-enforcement-design.md`
**Status:** ready

---

## Ordem de execução

Tasks 1 e 2 podem ser feitas em paralelo (arquivos independentes).
Tasks 3 e 4 podem ser feitas em paralelo entre si, após Task 1.
Task 5 é a maior e central — pode ser feita em qualquer ordem, mas é melhor deixar por último para incorporar decisões das anteriores.
Task 6 é independente e pequena.

```
T1 (archivist skill.yaml)  ──┐
T2 (drift-patterns.yaml)   ──┼──► T5 (SKILL.md orquestrador)
T3 (personas)              ──┘
T4 (skill.yaml pipeline)   ──────► T5
T6 (archivist SKILL.md)    ──────► independente, pode ser junto com T1
```

---

## Task 1 — Corrigir contrato do archivist skill.yaml

**Arquivo:** `.strategist/skills/archivist/skill.yaml`

1.1. Alterar `risk_score: read_only` → `risk_score: write_analysis`

1.2. Substituir o bloco `output:` de:
```yaml
output:
  reviewed_plan_path: string  # <base_path>/refined/<mission_id>-plan.md
```
para:
```yaml
output:
  output_dir: string  # <base_path>/refined/<mission_id>/ (diretório — estrutura interna livre)
```

1.3. Adicionar ao bloco `input:`:
```yaml
  sniper_skill_yaml: string   # path para skill.yaml do provider de execução
  sniper_skill_md: string     # path para SKILL.md do provider de execução
  mission_docs_dir: string    # opcional — path para documentação base do projeto
```

1.4. Remover `required_sections` e `behavior` atuais — a estrutura interna do output
é responsabilidade da skill que ocupa o papel. Manter apenas o contrato de interface.

1.5. Atualizar `forbidden_behaviors`:
- substituir `produce_output_without_discovery_artifact` por `write_outside_output_dir`
- adicionar `skip_sniper_peer_review`
- adicionar `reload_mission_docs_dir` (esse contexto já vem no artefato do Ranger)

---

## Task 2 — Adicionar novos padrões ao drift-patterns.yaml

**Arquivo:** `.strategist/identity/drift-patterns.yaml`

2.1. Adicionar padrão `ranger_to_sniper_shortcut`:
```yaml
- id: ranger_to_sniper_shortcut
  symptom: >
    Estou prestes a invocar o slot de execução logo após o slot de discovery,
    sem ter invocado o slot de refinement (Archivist).
  correction: >
    Parar. Invocar Archivist com o artefato do Ranger, os paths de skill.yaml
    e SKILL.md do Sniper, e o mission_docs_dir se disponível. Somente após
    Archivist concluir, apresentar o gate de aprovação ao usuário.
```

2.2. Adicionar padrão `gate_artifact_absent_silent`:
```yaml
- id: gate_artifact_absent_silent
  symptom: >
    O diretório refined/<mission_id>/ não existe ou está vazio e estou
    prestes a retornar status plan_only silenciosamente.
  correction: >
    Parar. Ausência do artefato do Archivist é erro de pipeline, não resultado
    válido. Verificar se Archivist foi invocado. Se não: invocar agora.
    Se sim e falhou: emitir evento blocked com reason=archivist_failed.
```

---

## Task 3 — Atualizar approval_prompt no personas/pragmatic.yaml

**Arquivo:** `.strategist/personas/pragmatic.yaml`

3.1. Substituir o `approval_prompt` atual pelo formato do gate único:
```yaml
approval_prompt: |
  [Strategist] Refinamento concluído.
  Plano: {artifact_path}

  Missão principal:
  {mission_tasks_summary}

  Side quests identificados pelo Ranger:
  {side_quests_list}

  Aprovar execução? (yes / no / review)
```

Onde:
- `{mission_tasks_summary}` = lista numerada de tasks extraída pelo orquestrador do artefato do Archivist
- `{side_quests_list}` = lista de side quests do Ranger, ou "nenhum" se vazio

---

## Task 4 — Atualizar pipeline no skill.yaml do orquestrador

**Arquivo:** `.strategist/skill.yaml`

4.1. Remover stages do pipeline:
- `side_quest_approval` (mini gate eliminado)
- `side_quest_execution` (consolidado no Sniper principal)

4.2. Atualizar stage `refinement`:
```yaml
- stage: refinement
  slot: refinement
  input:
    - discovery_artifact
    - sniper_skill_yaml     # resolvido no preflight (2c)
    - sniper_skill_md       # resolvido no preflight (2c)
    - mission_docs_dir      # de active.yaml ou mission_contract (opcional)
  artifact_path: "<base_path>/refined/<mission_id>/"
  produces: "<base_path>/refined/<mission_id>/"  # diretório, estrutura livre
```

4.3. Atualizar stage `approval_gate`:
```yaml
- stage: approval_gate
  type: mandatory_pause
  checks:
    - refined_dir_exists: "<base_path>/refined/<mission_id>/"
    - refined_dir_non_empty: true
  on_missing: blocked  # nunca plan_only silencioso
  description: >
    Apresentar plano refinado + side quests ao usuário.
    Aguardar aprovação explícita antes de invocar Sniper.
```

4.4. Adicionar ao bloco `forbidden_behaviors`:
```yaml
- invoke_sniper_before_archivist
- skip_archivist_for_simple_missions
- silent_plan_only_on_missing_artifact
- invoke_side_quest_sniper_without_main_gate
```

4.5. Adicionar campo `mission_docs_dir` ao bloco `sdd_injection.fields` (opcional):
```yaml
sdd_injection:
  optional: true
  fields:
    - execution_provider
    - base_path
    - knowledge_paths
    - governance_context
    - mission_docs_dir    # ← novo
```

---

## Task 5 — Atualizar SKILL.md do orquestrador (maior mudança)

**Arquivo:** `.strategist/SKILL.md`

5.1. **Seção 2c** — Resolver slot providers: ao resolver o Sniper, armazenar os paths
de `skill.yaml` e `SKILL.md` do provider para injeção no Archivist.

5.2. **Seção 5a** — Invocar Ranger com instrução explícita de usar todas as ferramentas
disponíveis, incluindo `mission_docs_dir` se declarado:
```
Instrução ao Ranger: antes de concluir a discovery, use todas as ferramentas
disponíveis — leitura de arquivos, busca no codebase, consulta ao mission_docs_dir.
Discovery incompleta por falta de consulta ao contexto disponível é um erro de Ranger.
```

5.3. **Seção 5a → 5e transição** — Substituir o bloco `<HARD-GATE>` inexistente por:
```
<HARD-GATE>
Ranger concluiu. PROIBIDO invocar Archivist ou qualquer outro slot agora.
PROIBIDO executar qualquer tarefa identificada pelo Ranger.
Ação permitida: emitir evento done do Ranger. Depois: invocar Archivist.
Esta parada não tem exceção — nem para missões simples, nem para side quests.
</HARD-GATE>
```

5.4. **Seções 5b, 5c, 5d** — Eliminar mini gate e side quest execution separados.
Substituir por nota: side quests são catalogados pelo Ranger no artefato de discovery
e apresentados no gate único após o Archivist. Não há phase separada.

5.5. **Seção 5e** — Atualizar invocação do Archivist com novos inputs:
- `discovery_artifact_path`
- `sniper_skill_yaml` + `sniper_skill_md` (resolvidos no preflight)
- `mission_docs_dir` (se disponível em active.yaml ou mission_contract)
- `mission_contract.planning_rules`
- `artifact_path`: `<base_path>/refined/<mission_id>/` (diretório)

5.6. **Seção 6** — Reescrever lógica do gate:

Remover:
- toda a lógica de verificação de `tasks.md`
- o caminho `plan_only` silencioso por artefato ausente

Substituir por:
```
<HARD-GATE>
Archivist concluiu. PROIBIDO invocar Sniper agora.
PROIBIDO executar qualquer tarefa do plano refinado.
Ação permitida: apresentar o gate de aprovação ao usuário. Aguardar resposta explícita.
Esta parada não tem exceção — nem se o plano parece simples ou óbvio.
</HARD-GATE>

Verificar:
1. <base_path>/refined/<mission_id>/ existe e tem conteúdo?
   - Não → emitir blocked com reason=archivist_failed. NUNCA plan_only silencioso.
   - Sim → apresentar gate.

2. Extrair resumo de tasks do artefato do Archivist (qualquer estrutura).
3. Extrair lista de side quests do artefato do Ranger.
4. Apresentar approval_prompt com ambas as listas.
5. Aguardar resposta explícita do usuário.

Respostas aceitas:
- yes / approve / authorize → invocar Sniper
- no / decline / stop → retornar status: plan_only (explícito, não silencioso)
- review → exibir conteúdo de refined/<mission_id>/, re-apresentar gate
```

---

## Task 6 — Atualizar SKILL.md do Archivist

**Arquivo:** `.strategist/skills/archivist/SKILL.md`

6.1. Atualizar seção de Input para incluir:
- `sniper_skill_yaml` e `sniper_skill_md`: paths para skill definition do Sniper
- `mission_docs_dir`: opcional, não recarregar — contexto L1 já vem no artefato do Ranger

6.2. Atualizar seção de Output:
- Remover path fixo `<base_path>/refined/<mission_id>-plan.md`
- Substituir por: "escrever no diretório `<base_path>/refined/<mission_id>/`; estrutura interna definida por esta skill"

6.3. Adicionar seção **"Peer Review com Sniper"** (após rascunhar o plano):
```
Após produzir o rascunho do plano:

1. Ler skill.yaml e SKILL.md do Sniper (injetados pelo orquestrador).
2. Para cada task do plano, verificar:
   - Conflita com algum mandate ou forbidden_behavior do Sniper?
   - Está dentro do risk_score e write_scope do Sniper?
3. Apresentar ao Sniper (como contexto de revisão, não como invocação):
   - Lista de tasks propostas
   - Pergunta: "Você consegue executar essas tasks dentro dos seus mandates?
     Alguma task vai te bloquear?"
4. Incorporar feedback: reformular tasks bloqueadas, registrar limitações conhecidas.
5. Finalizar plano com nota de revisão: "Peer review com Sniper: OK / N tasks ajustadas"
```

6.4. Adicionar seção **"Side Quests"**:
```
O artefato de discovery pode conter uma seção "Side Quests". Se presente:
- Não analisar profundamente — são itens pontuais já identificados pelo Ranger
- Transcrever como lista estruturada no artefato refinado
- O orquestrador os apresentará no gate junto com a missão principal
```

6.5. Atualizar sinalização de conclusão:
```
archivist complete.
artifact: <base_path>/refined/<mission_id>/
peer_review: ok | N tasks ajustadas
blockers: <count>
side_quests: <count>
```

---

## Checklist de verificação pós-implementação

- [ ] Preflight não bloqueia mais por `risk_score: read_only` no Archivist
- [ ] Pipeline não tem mais stages `side_quest_approval` / `side_quest_execution`
- [ ] SKILL.md do orquestrador tem dois blocos `<HARD-GATE>` explícitos
- [ ] Gate verifica diretório `refined/<mission_id>/`, não arquivo `tasks.md`
- [ ] Gate nunca retorna `plan_only` silencioso por artefato ausente — sempre `blocked`
- [ ] Gate único apresenta missão + side quests em uma só prompt
- [ ] Archivist recebe `sniper_skill_yaml` e `sniper_skill_md` como input
- [ ] Archivist faz peer review com Sniper antes de finalizar o plano
- [ ] Archivist escreve em diretório, não arquivo único
- [ ] `drift-patterns.yaml` tem os dois novos padrões
- [ ] `forbidden_behaviors` no skill.yaml cobre os novos casos
