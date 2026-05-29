# Proposal: Sincronizar .strategist/SKILL.md com canônico
**Data:** 2026-05-28
**Escopo:** `.strategist/SKILL.md` (runtime copy usada pelo agente)

## O quê
Sincronizar 4 divergências identificadas entre `strategist/SKILL.md` (canônico) e `.strategist/SKILL.md` (runtime):
1. Tabela de contratos ausente (Ranger/Archivist/Sniper ↔ slot key ↔ contract ↔ progress label)
2. Formato de artefato do Archivist stale (`-plan.md` flat → subdiretório `proposal.md + design.md + tasks.md`)
3. Lógica do Approval Gate stale (heurística → leitura determinística de `tasks.md`)
4. Drift pattern `route_plan_creation_to_sniper` ausente

## Por quê
O agente usando o runtime copy produziria artefatos no formato errado e tomaria decisões de gate incorretas. O canônico (`strategist/SKILL.md`) foi atualizado em sessões anteriores mas o runtime nunca foi ressincronizado.
