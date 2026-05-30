# Proposta: Guardrails em 3 Sprints — Strategist Skill Go
**Mission ID:** guardrails-20260529
**Date:** 2026-05-29
**Status:** refined

---

## O Quê

Implementar os guardrails ausentes identificados no discovery em três sprints incrementais,
cobrindo Fases 1 a 3 do modelo de maturidade proposto no documento de crítica.

## Por Quê

O projeto tem uma base sólida (Fase 1 em 85%), mas três gaps têm consequências concretas:

1. **`govulncheck` ausente** — se uma dependência tiver CVE conhecido, nenhum step de CI vai
   barrar a release. Risco real para releases Go públicas.

2. **Enforcement arquitetural ausente** — `TestDomainIsolation` verifica só a camada domain.
   Cruzamentos laterais entre `compile`, `install`, `stale`, `embed` são silenciosos.
   O drift estrutural se acumula até virar débito caro.

3. **Skill root resolve silenciosamente para caminho errado** — `~/.strategist/` não existe,
   então a skill bypassa o protocol e executa diretamente. Este é o anti-pattern
   `direct_execution` proibido pelo `protocol.md`. Documentado em `.analysis/todo/falha_strategist.md`.

Os gaps da Fase 3 (mandates formais) são baixo risco operacional mas fecham inconsistências
entre o que o `protocol.md` proíbe implicitamente e o que está registrado como mandate rastreável.

## Escopo

### Sprint 1 — CI + Linters (~2h)
- `govulncheck` no CI e Makefile
- Format check (`gofmt`) no CI
- Module hygiene (`go mod tidy`, `go mod verify`) no CI
- 5 linters adicionais no `.golangci.yaml`: `misspell`, `dupl`, `unconvert`, `ineffassign`, `gocritic`
- `contextcheck` com exclusão para bootstrap CLI

### Sprint 2 — Enforcement Arquitetural (~3h)
- `depguard` no `.golangci.yaml` para enforcement declarativo de imports
- `TestArchitectureDirection` expandido para cruzamentos laterais
- `strategist/contracts/architecture-rules.yaml` como mandate formal

### Sprint 3 — Governance + Skill Root Fix (~2h)
- Mandates formais: `no-hack-without-evidence.md`, `test-integrity.md`, `scope-locking.md`
- Fix do skill root: `strategist install --global` cria `~/.strategist/` com defaults

## Fora do Escopo

- Fase 4 (Scoring) — backlog, sem sprint definido
- `bodyclose`, `prealloc` — valor baixo no código atual (sem HTTP, sem slices grandes)
- `cyclop` — redundante com `gocognit` já presente
- Refatoração dos helpers duplicados (`writeGzJSON`) — débito a endereçar separadamente
- `execution_contract.schema.json` para o binário Go — CLI simples, complexidade não justificada agora
