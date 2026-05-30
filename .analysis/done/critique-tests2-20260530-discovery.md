# Discovery — critique_tests2: Go Testing World-Class
**Mission ID:** critique-tests2-20260530
**Date:** 2026-05-30
**Source:** `.analysis/todo/critique_tests2.md`
**Task type:** architecture_analysis

---

## Executive Summary

O projeto já implementa a maioria dos padrões world-class descritos em `critique_tests2.md`.
A base é sólida: `t.Parallel()` em 82 locais, table-driven tests, pacotes `_test` externos,
testify, 2 fuzz tests, testes de arquitetura, race detector no CI e coverage gate 90%.

**4 gaps concretos** identificados — nenhum é bloqueante, todos são melhorias de manutenção,
performance e confiança:

| Gap | Severidade | Esforço |
|-----|-----------|---------|
| Helpers duplicados (writeGzJSON, readGzJSON, minimalRoot) | Alta | ~1h |
| testdata/ para fixtures YAML complexas | Média | ~1h |
| Build-tag integration tests | Média | ~30min |
| Benchmarks para hot paths | Baixa | ~45min |

---

## O que já está implementado

| Recomendação critique_tests2 | Status | Evidência |
|------------------------------|--------|-----------|
| `t.Parallel()` | ✅ | 82 ocorrências em todos os pacotes |
| Table-driven tests | ✅ | `tests := []struct{}` em compile_test.go, stale_test.go |
| `package x_test` externo | ✅ | compile_test, stale_test, domain_test, embed_test |
| testify/assert + require | ✅ | 30+ imports, uso consistente |
| Fuzz tests para parsers | ✅ | `FuzzCompileConfig`, `FuzzCompileIndex` em compile/ |
| Testes de arquitetura (`go list`) | ✅ | TestDomainIsolation + TestLateralIsolation |
| Race detector no CI | ✅ | `go test -race ./...` em test.yml |
| Coverage gate 90% | ✅ | Script de gate por pacote em test.yml |
| golangci-lint v2 | ✅ | CI + Makefile |
| govulncheck | ✅ | CI + Makefile |
| Format check (gofmt) | ✅ | CI step |
| Module hygiene | ✅ | `go mod tidy` + `git diff` no CI |

---

## Gaps identificados

### Gap 1 — Helpers duplicados (ALTA severidade)

`writeGzJSON`, `readGzJSON` e variantes de `minimalRoot` estão definidos em 4 pacotes:

| Helper | Ocorrências |
|--------|-------------|
| `writeGzJSON` | compile_test.go, stale_test.go, cmd_test.go |
| `readGzJSON` | compile_test.go, tests/compile_test.go |
| `minimalRoot` / `minimalStrategistRoot` | compile_test.go, cmd_test.go |
| `captureStdout`, `freshArtifactDir` | cmd_test.go (únicos) |

Custo: qualquer mudança no formato de artifact exige update em 3–4 arquivos.
Fix: extrair para `internal/testutil/` — acessível por todos os pacotes de teste.

### Gap 2 — testdata/ para fixtures YAML (MÉDIA severidade)

`compile_test.go` define YAML inline como strings Go:
```go
require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: full\n"), 0o644))
```

Para inputs complexos, isso dificulta leitura e edição. Os `testdata/` seriam:
- `internal/compile/testdata/valid-minimal/active.yaml`
- `internal/compile/testdata/valid-full/` (tree completa)
- `internal/compile/testdata/invalid-*.yaml` (casos de erro)

### Gap 3 — Build-tag integration tests (MÉDIA severidade)

`tests/install_test.go` e `tests/compile_test.go` são testes de integração reais
(criam dirs em disco, rodam o pipeline completo), mas não têm `//go:build integration`.
Isso mistura unit tests rápidos com integração no `go test ./...` padrão.

Fix: adicionar `//go:build integration` ao package `tests` e configurar CI para rodar:
```yaml
- run: go test ./...
- run: go test -tags=integration ./...
```

### Gap 4 — Benchmarks para hot paths (BAIXA severidade)

`compile.CompileAll` é chamado em toda invocação do strategist CLI (via `check-stale`).
Não há nenhum `func Benchmark*` no projeto. Uma regressão de performance seria invisível.

Candidatos para benchmark:
- `BenchmarkCompileAll` — end-to-end compile pipeline
- `BenchmarkIsStale` — chamado no bootstrap de cada missão
- `BenchmarkCompileConfig` — maior função de compile

---

## Anti-patterns da critique_tests2 — verificação

| Anti-pattern | Presente no projeto? |
|-------------|---------------------|
| Testes dependentes de ordem | ✅ Não (t.Parallel() em todos) |
| Portas fixas | ✅ Não (uso de t.TempDir()) |
| Sleep arbitrário | ✅ Não encontrado |
| Comparar JSON como string crua | ✅ Não (uso de assert.Equal em structs) |
| Mock excessivo | ✅ Não (fakes e interfaces mínimas) |
| Testar implementação em vez de comportamento | ✅ Maioria usa package externo |
| helpers_test.go usa internal package | ⚠️ `package compile` (acesso a internals) |

O `compile/helpers_test.go` usa `package compile` (não `_test`) — mas os helpers são privados
e não há caminho melhor para testá-los. Aceitável.

---

## O que NÃO fazer

- **Não migrar de testify para go-cmp** — testify funciona bem, ganho marginal
- **Não adicionar golden files ainda** — nenhum output textual complexo existe no projeto
- **Não adicionar property-based testing** — fuzzing nativo já cobre os parsers críticos

---

## Priorização final

```
Sprint 1 (~2h):
  T1. Criar internal/testutil/ com writeGzJSON, readGzJSON, minimalStrategistRoot
  T2. Migrar compile_test.go para usar testutil
  T3. Migrar stale_test.go para usar testutil
  T4. Migrar cmd_test.go para usar testutil
  T5. Migrar tests/ para usar testutil

Sprint 2 (~1.5h):
  T6. Criar testdata/ em internal/compile/ com fixtures YAML reais
  T7. Refatorar casos de table-driven em compile_test.go para usar testdata/
  T8. Adicionar //go:build integration a tests/ + CI step

Sprint 3 (~45min):
  T9. Adicionar BenchmarkCompileAll em internal/compile/
  T10. Adicionar BenchmarkIsStale em internal/stale/
```
