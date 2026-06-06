# ADR-0007 — Structural Compression Pipeline — Agent Contract vs Go Runtime

**Status:** Accepted
**Date:** 2026-06-02

---

## Contexto

O design de token economy especificou a Phase 4 como uma interface Go `CompressionProvider` com um provider `builtin-structural`. Esse componente ficaria entre a recuperação de fontes e a invocação do LLM, produzindo scored source cards de forma determinística.

Na época da decisão, o `context-enrichment/skill.yaml` já declarava `source_cards` e `compression_metrics` como campos de saída no contrato do agente. Nenhuma implementação Go existia em `internal/`.

O contexto que levou à avaliação: o pipeline de enriquecimento de contexto busca fontes de conhecimento por `task_type` e as entrega ao LLM. Sem compressão estruturada, o LLM recebe todas as fontes dentro do orçamento de tokens e decide internamente quais priorizar. A alternativa seria um componente Go que scorea e filtra as fontes antes de passá-las — garantindo determinismo e auditabilidade ao custo de complexidade de manutenção.

## Decisão

A Phase 4 foi adiada. A compressão estrutural é implementada como **contrato comportamental do agente** via `context-enrichment/skill.yaml`, e não como runtime Go.

O LLM implementa compressão semântica nativamente ao seguir o contrato da skill (score por `task_type` match, trust tier, keyword overlap; aplicar budget limits por mode; produzir source cards no formato evidence → interpretation → impact).

A interface Go `CompressionProvider` permanece no backlog — a ser considerada quando:
1. A Phase 2 (chest index files) estiver completa e fornecer inputs de ranking determinísticos
2. Compressão não-determinística do agente causar regressão de qualidade mensurável
3. Requisitos de auditoria/testabilidade exigirem compressão reproduzível

## Consequências

**Trade-offs aceitos:**
- Comportamento de compressão é não-determinístico e não é unit-testável
- Sem garantia de fallback se o LLM ignorar o contrato
- Métricas em `chest-usage.jsonl` são auto-reportadas pelo agente, não computadas

**O que fica mais fácil:**
- Zero custo de manutenção Go para compressão
- Contrato evolui em YAML sem recompilação
- Funciona com todos os providers LLM que seguem o contrato da skill

**O que fica mais difícil:**
- Auditar exatamente quais fontes foram selecionadas e por quê
- Testar regressão de qualidade de compressão
- Comportamento offline determinístico (sem rede/LLM)
