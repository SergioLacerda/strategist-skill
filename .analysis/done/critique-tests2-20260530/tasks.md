# Tasks: Go Testing World-Class — Gaps Restantes
**Mission ID:** critique-tests2-20260530

---

## Sprint 1 — internal/testutil/ (~2h)

**T1.1** Criar `internal/testutil/testutil.go` com:
- `WriteGzJSON(t *testing.T, path string, v any)` — cria dirs necessários, escreve gzip JSON
- `ReadGzJSON(t *testing.T, path string, v any)` — abre, descomprime, decodifica JSON
- `MinimalRoot(t *testing.T, dir string)` — cria `active.yaml` (mode: full), `personas/epic.yaml`, `roles/default.yaml`
- Package: `package testutil`

**T1.2** Migrar `internal/compile/compile_test.go`:
- Remover funções locais `readGzJSON` e `minimalRoot`
- Adicionar import `github.com/SergioLacerda/strategist-skill/internal/testutil`
- Substituir todos os usos: `readGzJSON` → `testutil.ReadGzJSON`, `minimalRoot(t, dir)` → `testutil.MinimalRoot(t, dir)`

**T1.3** Migrar `internal/stale/stale_test.go`:
- Remover função local `writeGzJSON`
- Adicionar import testutil
- Substituir todos os usos: `writeGzJSON` → `testutil.WriteGzJSON`

**T1.4** Migrar `cmd/strategist/cmd_test.go`:
- Remover funções locais `writeGzJSON` e `minimalStrategistRoot`
- Adicionar import testutil
- Substituir usos (preservar `captureStdout` e `freshArtifactDir` — ficam locais)

**T1.5** Migrar `tests/compile_test.go`:
- Remover funções locais `readGzJSON` e `minimalStrategistRoot`
- Adicionar import testutil
- Substituir usos

**T1.6** Validar: `go test -race ./...` deve passar sem regressões

---

## Sprint 2 — testdata/ + build tags (~1.5h)

**T2.1** Criar `internal/compile/testdata/valid-minimal/`:
- `active.yaml` — `mode: full\n`
- `personas/epic.yaml` — `name: Epic\n`
- `roles/default.yaml` — `name: Default\n`

**T2.2** Criar `internal/compile/testdata/valid-with-knowledge/`:
- mesmos arquivos de valid-minimal
- `knowledge.index.yaml` — `sources:\n  - id: doc\n    tags: [go]\n    path: doc.yaml\n`

**T2.3** Criar `internal/compile/testdata/invalid-yaml/active.yaml`:
- Conteúdo: `: invalid yaml: :`

**T2.4** Adicionar caso `"testdata: valid-minimal"` em `TestCompileConfig`:
- setup: copiar `testdata/valid-minimal/` para TempDir (usando `os.CopyFS` ou helper)
- Verificar que artifact é produzido corretamente

**T2.5** Adicionar `//go:build integration` como primeira linha (antes de `package`) em:
- `tests/compile_test.go`
- `tests/install_test.go`
- `tests/stale_test.go`

**T2.6** Adicionar step no `.github/workflows/test.yml` após "Test (with race detector)":
```yaml
- name: Integration tests
  run: go test -tags=integration -race ./tests/...
```

**T2.7** Validar: `go test ./...` (sem tag) não deve rodar os testes de `tests/`; `go test -tags=integration ./tests/...` deve passar

---

## Sprint 3 — Benchmarks (~45min)

**T3.1** Criar `internal/compile/bench_test.go`:
- Package: `package compile_test`
- `BenchmarkCompileAll` — usa `testutil.MinimalRoot`, reseta timer antes do loop, chama `compile.All`
- `BenchmarkCompileConfig` — usa `testutil.MinimalRoot`, chama `compile.Config`

**T3.2** Criar `internal/stale/bench_test.go`:
- Package: `package stale_test`
- `BenchmarkIsStale_Fresh` — artifact fresh (sem sources), `checker.IsStale(path)`
- `BenchmarkIsStale_Stale` — artifact stale (source com mtime diferente do registrado)

**T3.3** Adicionar ao `Makefile`:
- `.PHONY` line: adicionar `bench`
- Target após `vuln`:
  ```makefile
  bench:
  	go test -bench=. -benchmem ./...
  ```

**T3.4** Validar: `make bench` roda sem erros; benchmarks reportam ns/op e B/op

---

## Critérios de aceite da missão

- `go test -race ./...` verde (cobertura ≥ 90% por pacote)
- `go test -tags=integration -race ./tests/...` verde
- Nenhuma duplicação de `writeGzJSON`, `readGzJSON`, `minimalRoot` entre pacotes
- `make bench` roda sem erros
- `golangci-lint run ./...` sem novas issues
