# Proposal: Docs Nomenclature Update + Flow Diagrams
**Date:** 2026-05-28
**Scope:** readme.md + readme_detailed.md

## O quê
1. Renomear Scout/Engineer/Hunter → Ranger/Archivist/Sniper em toda a documentação
2. Corrigir 2 valores factuais de risk_score
3. Corrigir lacunas de coerência: pipeline simplificado no doc não reflete o pipeline real (housekeeping_scan, mini gate, side quests, subdirectório refined, conditional approval gate)
4. Adicionar diagrama de fluxo de negócio com iteração entre papéis + sub-fluxo de side quests
5. Adicionar diagrama técnico interno completo da skill

## Por quê
A documentação atual descreve um pipeline de 3 fases lineares (Scout → Engineer → Hunter) que já não corresponde à implementação real, que possui 7 fases (Ranger → housekeeping_scan → [mini gate] → Sniper side quests → Archivist → approval gate → Sniper main). Sem os diagramas de fluxo, o onboarding de novos usuários/contribuidores requer leitura do SKILL.md completo para entender o comportamento do sistema.
