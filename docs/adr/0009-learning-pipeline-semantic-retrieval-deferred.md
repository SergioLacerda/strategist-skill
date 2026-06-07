# ADR-0009 — Semantic Retrieval Deferred In Learning Pipeline

**Status:** Accepted  
**Date:** 2026-06-07

---

## Contexto

O Strategist já possui:

- buffer de outcomes em `.strategist/memory/outcomes.tmp`
- flush governado para `.strategist/memory/outcomes.jsonl`
- `source-hints.yaml` como overlay manual
- fase de learning não-bloqueante

A análise refinada de `2026-06-07-learning-pipeline-embeddings` concluiu que o gap atual
não é “ausência de embeddings”, e sim maturidade insuficiente do corpus histórico.

No estado atual do workspace:

- `outcomes.tmp` contém `12` entradas
- `outcomes.jsonl` ainda não apresenta corpus consolidado observável
- `source-hints.yaml` não possui hints aprendidos

Não há evidência de que retrieval por tags e busca lexical já falhem com frequência
suficiente para justificar indexação semântica.

## Decisão

Recuperação semântica por embeddings fica **explicitamente deferida**.

O baseline canônico continua sendo:

1. outcomes estruturados em `outcomes.jsonl`
2. retrieval por tags e hints
3. busca lexical sobre o corpus histórico

Qualquer índice semântico futuro deve obedecer às regras:

- é opcional, nunca obrigatório
- é derivado de `outcomes.jsonl`
- é local e rebuildável
- sua falha nunca bloqueia a missão
- `context-enrichment` deve degradar para tags ou lexical search

## Consequências

**Positivas:**

- evita complexidade operacional prematura
- mantém o learning pipeline auditável e simples
- força decisão baseada em benchmark, não em hipótese
- preserva fallback robusto sem dependência de modelo externo

**Negativas:**

- recuperação entre missões continua limitada a tags, hints e busca lexical
- similaridade semântica não fica disponível no curto prazo
- benchmark futuro ainda precisa ser desenhado e executado quando o corpus amadurecer

## Critérios de Reavaliação

Reabrir a decisão apenas quando todos os critérios abaixo forem satisfeitos:

1. `outcomes.jsonl` possuir pelo menos `50` missões reais
2. existirem pelo menos `3` casos documentados onde tags ou hints falharam
3. houver benchmark comparando tags, lexical e semantic retrieval
4. a operação local do índice semântico estiver explicitamente aceita
