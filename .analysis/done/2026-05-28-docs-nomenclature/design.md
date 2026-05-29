# Design: Docs Nomenclature Update
**Data:** 2026-05-28

## Decisões tomadas

- Toda prosa em pt-BR; nomes de papéis, variáveis e termos técnicos sem tradução consagrada preservados em inglês
- Diagramas ASCII embutidos diretamente no markdown (sem imagens externas)
- Novos diagramas inseridos antes do "Fluxo Principal" para hierarquia conceitual: Negócio → Técnico → Sequência de eventos
- Fluxo de Negócio usa linguagem orientada a propósito ("O que precisa ser feito?") para diferenciar de fluxo técnico
- Fluxo Técnico usa bordas de caixa ASCII contínuas para distinguir visualmente fases internas (Strategist) de fases delegadas (slots)
- `tasks.md` usado como critério determinístico do approval gate (leitura explícita antes do gate) em vez de heurística sobre "requires Sniper execution"
