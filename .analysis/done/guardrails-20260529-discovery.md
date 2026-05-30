# Discovery: Avaliação de Guardrails — Strategist Skill em Go
**Mission ID:** guardrails-20260529
**Date:** 2026-05-29
**Task Type:** architecture_analysis
**Status:** discovery_complete

---

## Sumário Executivo

O projeto está em **Fase 1 sólida**. A base de quality gates existe e funciona. Os gaps são concretos, priorizáveis e não bloqueiam desenvolvimento atual — mas alguns (govulncheck, go mod verify) têm risco de segurança real se releases forem publicadas. A adoção completa das 4 fases é viável de forma incremental.

**Gap crítico imediato:** `govulncheck` ausente. É o único gap com risco de supply-chain que pode afetar releases.

---

## Fase 1 — Baseline (Quality Gates)

### ✅ Implementado

| Item | Evidência |
|------|-----------|
| `go test -race ./...` | Makefile + CI (test.yml) |
| `go vet ./...` | CI test.yml |
| Coverage gate ≥ 90% por pacote | Makefile `cover-gate` + CI |
| golangci-lint com: errcheck, gosec, staticcheck, govet, revive, wrapcheck, exhaustive, gocognit, testifylint | `.golangci.yaml` |
| Fuzz tests para parser YAML | `internal/compile/fuzz_test.go` |
| Race tests no CI | `go test -race ./...` |
| SHA pinning nos GitHub Actions | `actions/checkout@34e1...`, `setup-go@d35c...`, `golangci-lint-action@55c2...` |
| `go build ./...` no CI | test.yml (passo final) |

### ⚠️ Parcialmente implementado

**golangci-lint — linters ausentes do `.golangci.yaml`:**

| Linter | Risco sem ele |
|--------|--------------|
| `cyclop` / `gocognit` | gocognit está, cyclop ausente — ambos medem complexidade ciclomática, redundância parcial mas cyclop é mais preciso por função |
| `contextcheck` | `context.Background()` usado em `installer.go:99` (NewInstaller.Install) — em CLI é aceitável, mas linter documentaria intenção |
| `dupl` | duplicação de código entre `tests/` e `internal/` (readGzJSON, writeGzJSON duplicados) |
| `misspell` | sem detecção de typos em comentários/docs |
| `unconvert` | conversões desnecessárias não detectadas |
| `prealloc` | slices sem pré-alocação não sinalizados |
| `bodyclose` | HTTP response body leaks (baixo risco atual, sem HTTP no código) |
| `gocritic` | análise estática complementar ao revive |
| `ineffassign` | atribuições sem uso não detectadas |

### ❌ Ausente

| Item | Impacto | Esforço |
|------|---------|---------|
| `govulncheck ./...` | **Alto** — vulnerabilidades conhecidas em dependências não detectadas antes de releases | Baixo (1 linha no CI) |
| `gofmt -l .` no CI | Médio — formatação não enforced, pode causar diff noise | Baixo (1 passo no CI) |
| `go mod tidy && git diff --exit-code go.mod go.sum` | Médio — go.mod/go.sum podem ficar fora de sync sem detecção | Baixo (2 linhas no CI) |
| `go mod verify` | Médio — integridade do module cache não verificada | Baixo (1 linha no CI) |

---

## Fase 2 — Arquitetura

### ✅ Implementado

| Item | Evidência |
|------|-----------|
| Domain isolation test | `internal/domain/architecture_test.go` — `TestDomainIsolation` via `go list -deps` |
| Estrutura `cmd/internal/domain` | Presente e respeitada |
| `internal/domain` como camada pura de tipos/contratos | `domain/types.go`, `domain/errors.go`, `domain/ports.go` sem imports internos |

### ⚠️ Parcialmente implementado

- `TestDomainIsolation` cobre apenas a direção `domain → outros`. Não cobre:
  - `compile`, `install`, `stale`, `embed` importando uns aos outros desnecessariamente
  - `cmd` importando diretamente lógica que deveria estar em `internal`

### ❌ Ausente

| Item | Impacto | Esforço |
|------|---------|---------|
| `depguard` no `.golangci.yaml` para enforcement declarativo de direção de imports | Médio — drift arquitetural não detectado automaticamente | Médio (configuração depguard) |
| Mandate formal `ARCHITECTURE DEPENDENCY DIRECTION` | Baixo (documentação) | Baixo |
| `TestArchitectureDirection` cobrindo camadas além do domain | Médio — caminho de test > linter para validação versionada | Médio |

**Estrutura de dependência atual (evidenciada):**

```
cmd/strategist
  ↓ imports
internal/compile, internal/install, internal/stale, internal/embed
  ↓ importam
internal/domain
```

Direção está correta. Enforcement é que falta.

---

## Fase 3 — Skill Governance

### ✅ Implementado

| Item | Evidência |
|------|-----------|
| Protocol com stop conditions e forbidden behaviors | `strategist/protocol.md` |
| Schemas de slots (active, roles, intake, slot-output) | `strategist/schemas/` |
| Progress event contract | `strategist/schemas/progress-contract.yaml` |
| known-providers.yaml (risk registry) | `.strategist/known-providers.yaml` |
| Approval gate obrigatório | `strategist/SKILL.md` §6 |
| Drift patterns auto-correction | `.strategist/identity/drift-patterns.yaml` |
| Execution contracts nas skill.yaml internas | `strategist/skills/*/skill.yaml` com risk_score |

### ❌ Ausente

| Item | Impacto | Esforço |
|------|---------|---------|
| Mandate formal `NO HACK WITHOUT EVIDENCE` como arquivo `.sdd/` | Médio — regra existe implicitamente no protocol.md mas não como mandate rastreável | Baixo |
| `execution_contract.schema.json` para o binário Go | Baixo — binário atual é CLI simples, sem contrato de execução | Médio |
| Mandates de test integrity e scope locking como governance files | Baixo — complementa SDD governance já existente | Baixo |
| `golangci.yml` quality file em `.sdd/quality/` | Baixo — config já existe em raiz do projeto | Baixo (reorganização) |

---

## Fase 4 — Scoring

### Estado: não iniciado (adequado para a fase atual)

| Score | Descrição | Prioridade |
|-------|-----------|------------|
| Hack risk score | Avalia mudanças por evidência/diagnose | Baixa — requer maturidade de fases anteriores |
| Scope drift score | Detecta expansão não autorizada | Baixa |
| Validation confidence | % de mudanças com artefatos validados | Baixa |
| Architecture compliance | Taxa de violações de direção de imports | Baixa — implementar após Fase 2 |
| Test integrity score | Qualidade e cobertura dos testes | Baixa |

---

## Anti-patterns Ativos no Projeto

### Código
| Anti-pattern | Localização | Severidade |
|-------------|------------|------------|
| `context.Background()` em service | `internal/install/installer.go:99` (`NewInstaller.Install`) | Baixa — CLI tool; aceitável mas documentar |

### Testes
| Anti-pattern | Localização | Severidade |
|-------------|------------|------------|
| Helpers duplicados entre pacotes | `writeGzJSON`/`readGzJSON` em `tests/`, `internal/compile/`, `cmd/strategist/`, `internal/stale/` | Média — test helpers não compartilhados, risco de divergência |

### Arquitetura
Nenhum detectado. Direção de dependências está correta.

### Agente/Skill
| Anti-pattern | Localização | Severidade |
|-------------|------------|------------|
| Skill root resolution falha silenciosamente | `~/.claude/skills/strategist/SKILL.md` → `~/.strategist/` (inexistente) | **Alta** — causa direct_execution sem protocol (ver .analysis/todo/falha_strategist.md) |

---

## Proposta de Adoção Incremental

### Sprint 1 — Fechar gaps Fase 1 (esforço: ~2h)

1. Adicionar ao CI (`test.yml`):
   - `go mod tidy && git diff --exit-code go.mod go.sum`
   - `test -z "$(gofmt -l .)"`
   - `go mod verify`
   - `govulncheck ./...`

2. Adicionar ao `.golangci.yaml`:
   - `misspell`, `dupl`, `unconvert`, `ineffassign`, `gocritic`
   - Avaliar `contextcheck` com exceção para CLI bootstrap
   - Defer `prealloc`, `bodyclose` para depois (baixo valor no código atual)

3. Adicionar ao `Makefile`:
   - `vuln: govulncheck ./...`

### Sprint 2 — Enforcement arquitetural (esforço: ~3h)

4. Configurar `depguard` no `.golangci.yaml` com as regras de direção
5. Expandir `TestArchitectureDirection` para cobrir camadas além do domain
6. Criar mandate `ARCHITECTURE DEPENDENCY DIRECTION` em `.sdd/` ou `strategist/`

### Sprint 3 — Governance formal (esforço: ~2h)

7. Criar mandate `NO HACK WITHOUT EVIDENCE` em `.sdd/source/mandates/` ou `strategist/contracts/`
8. Criar `test-integrity.md` e `scope-locking.md` como mandates complementares
9. Resolver o gap do skill root (`~/.strategist/` ausente) — ver `.analysis/todo/falha_strategist.md`

### Fase 4 — Scoring (backlog, sem sprint definido)

Implementar após Sprints 1-3 estarem estáveis.

---

## Diagnóstico Final

```
Fase 1 (Baseline):        85% implementado — gaps pequenos, sprint curta
Fase 2 (Arquitetura):     40% implementado — base existe, falta enforcement
Fase 3 (Skill Governance): 70% implementado — protocol e schemas presentes, mandates ausentes
Fase 4 (Scoring):          0% implementado — adequado, não é prioridade agora
```

**Sequência recomendada:** Sprint 1 → Sprint 2 → Sprint 3 → Fase 4 como backlog.
O risco mais alto hoje é `govulncheck` ausente com releases Go planejadas.
