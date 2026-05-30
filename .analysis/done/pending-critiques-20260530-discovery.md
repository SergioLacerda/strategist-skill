# Ranger Discovery — Pending Critiques Evaluation
**Mission ID:** pending-critiques-20260530
**Date:** 2026-05-30
**Phase:** Ranger (write_pending)
**Sources evaluated:**
- `.analysis/pending/critique_archtecture.md`
- `.analysis/pending/critique_strategist.md`
- `.analysis/pending/critique_strategistV2.md`
- `.analysis/pending/critique-tests2-20260530-discovery.md` (already implemented)

---

## Executive Summary

Dos 3 documentos de crítica ativos, a maioria dos itens concretos já foi implementada em missões anteriores. Restam **3 gaps de alta/média prioridade** e **1 quick fix de CI**.

O documento `critique_archtecture.md` propõe uma refatoração Clean/Hexagonal massiva. Avaliação: **não se aplica** ao projeto atual — o CLI já tem separação de responsabilidades adequada e a estrutura proposta seria over-engineering para uma ferramenta CLI com 5 pacotes internos.

---

## Front 1 — CI/CD

### O que já está implementado ✅
| Item | Evidência |
|------|-----------|
| govulncheck no CI | test.yml: step "Vulnerability check" |
| gofmt check no CI | test.yml: step "Format check" |
| go mod tidy + verify | test.yml: step "Module hygiene" |
| go test -race | test.yml: step "Test (with race detector)" |
| Integration tests separados | test.yml: `-tags=integration ./tests/...` |
| Coverage gate 90% por pacote | test.yml: step "Coverage gate" |
| Build verification | test.yml: step "Build" |
| go-version-file: go.mod | test.yml: setup-go |
| golangci-lint v2 via go install | test.yml: step "Lint" |

### Gap identificado 🔴
- **release.yml** ainda usa `go-version: "1.22"` (hardcoded) — inconsistente com `test.yml`
- Fix: mudar para `go-version-file: "go.mod"` (1 linha)

---

## Front 2 — Custom Error Types

### O que existe atualmente
`internal/domain/errors.go` — apenas 3 sentinel errors simples:
```go
var (
    ErrArtifactAbsent  = errors.New("artifact does not exist")
    ErrManifestMissing = errors.New("manifest not found")
    ErrSourceStale     = errors.New("source file modified after artifact")
)
```

### Gap identificado 🟡
`critique_strategistV2.md` propõe `SkillError` struct com:
- `Type ErrorType` — categoriza por stop condition do protocolo
- `Message`, `Cause`, `Context map[string]any`
- `Transient bool` + retry logic

Os tipos de erro mapeiam diretamente as stop conditions do `protocol.md` Strategist:
`slot_provider_not_found`, `preflight_failed`, `discovery_failed`, etc.

**Avaliação:** Genuinamente útil. Mas o código CLI atual não usa o protocolo de missão — essas stop conditions são tratadas pelo agente, não pelo binário Go. O valor imediato é limitado.

**Decisão:** Implementar `SkillError` apenas se o projeto pretende expor erros estruturados para consumo programático. Para CLI pura, sentinel errors são suficientes.

---

## Front 3 — Validate Command

### O que existe atualmente
Nenhum subcomando `strategist validate` existe. O `cmd/strategist/` tem:
`check_stale.go`, `compile.go`, `install_global.go`, `install.go`, `root.go`, `version.go`

### Gap identificado 🟡
`critique_strategistV2.md` propõe `validate.go` que verifica:
- `active.yaml` — campos obrigatórios
- `personas/*.yaml` — tone_directive, phase_labels
- `roles/*.yaml` — slots: discovery, refinement, execution
- `knowledge.index.yaml` — YAML válido

**Avaliação:** Útil para UX — usuário pode rodar `strategist validate` após editar `.strategist/` manualmente. Atualmente, erros de config só aparecem no momento de uso.

**Atenção:** O código template tem 1 bug:
```go
if !entry.Name()[len(entry.Name())-5:] == ".yaml" {  // PANIC se nome < 5 chars
```
Precisa ser reescrito com `strings.HasSuffix(entry.Name(), ".yaml")`.

**Decisão:** Implementar — utilidade real para o usuário final.

---

## Front 4 — Structured Logging (EventLogger)

### Gap identificado 🟢 (baixa prioridade)
`critique_strategistV2.md` propõe `internal/strategist/events.go` com `EventLogger`
para emitir `ProgressEvent` por fase do pipeline.

**Avaliação:** O pipeline do Strategist é orquestrado pelo agente Claude, não pelo código Go. O binário executa operações atômicas (compile, check-stale, install) — não tem um "pipeline" interno a ser logado. Este padrão seria relevante se o Go orchestrasse as fases Ranger→Archivist→Sniper.

**Decisão:** DEFER — não aplicável à arquitetura atual.

---

## Front 5 — Arquitetura Clean/Hexagonal

### Proposta em critique_archtecture.md
Refatoração completa para:
```
internal/application/    ← Use cases / Orchestration
internal/ports/          ← Interfaces (hexagonal)
internal/infrastructure/ ← Concrete adapters (LLM, Git, FS)
internal/presentation/   ← CLI commands
internal/config/         ← Config + validation
```
Com google/wire para DI.

### Avaliação ❌ DEFER

**O projeto já tem:**
- `internal/domain/ports.go` — interfaces (Compiler, StaleChecker, Installer, FileExtractor)
- `internal/domain/types.go` — tipos core
- `internal/domain/errors.go` — erros domain
- Separação clara: domain → compile/install/stale (implementações)
- `depguard` no `.golangci.yaml` enforcement lateral isolation

**Por que não aplicar agora:**
1. CLI com 5 pacotes internos não precisa de hexagonal architecture
2. google/wire adiciona complexidade de build sem benefício para operações atômicas CLI
3. application/, infrastructure/, presentation/ teriam ~1 arquivo cada — overhead de indireção sem ganho real
4. O projeto já segue Clean Architecture conceitualmente (domain separado das implementações)

**Revisitar quando:** o projeto começar a orquestrar fases do pipeline em Go (hoje feito pelo agente).

---

## Front 6 — GoReleaser / Release Pipeline

### O que existe
`.goreleaser.yaml` funcional com builds multi-OS, checksums, changelog.
`release.yml` usa goreleaser-action.

### Gaps identificados 🟢 (baixa prioridade)
- Sem brew formula (requer homebrew-tap separado)
- Sem docker images
- Sem cosign signing (supply chain security)

**Decisão:** DEFER — projeto ainda não tem usuários externos que requeiram essas features.

---

## Plano de Execução Proposto

### Sprint 1 — Quick fix + Validate (estimativa: ~1.5h)

| ID | Tarefa | Arquivo | Esforço |
|----|--------|---------|---------|
| T1 | Fix release.yml: go-version hardcoded → go-version-file | .github/workflows/release.yml | 5min |
| T2 | Criar cmd/strategist/validate.go com validateCmd | cmd/strategist/validate.go | ~45min |
| T3 | Adicionar TestValidateCmd_* em cmd_test.go | cmd/strategist/cmd_test.go | ~30min |
| T4 | Verificar coverage gate ainda passa | — | 5min |

### Deferred (não executar agora)
- SkillError struct — útil só quando Go começar a orquestrar o pipeline
- EventLogger — idem
- Clean/Hexagonal architecture — over-engineering para CLI atual
- GoReleaser enhancements — sem demanda

---

## Decisão sobre documentos pending

Após execução do Sprint 1, mover para done:
- `critique_archtecture.md` → avaliado, arquitetura adiada com justificativa documentada
- `critique_strategist.md` → avaliado, é resumo do critique_archtecture.md
- `critique_strategistV2.md` → avaliado, T1–T4 executados, rest deferred
- `critique-tests2-20260530-discovery.md` → já implementado, mover para done
