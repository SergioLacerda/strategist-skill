# Design: Go Testing World-Class — Gaps Restantes
**Mission ID:** critique-tests2-20260530

---

## Sprint 1 — internal/testutil/

### Arquivo a criar: `internal/testutil/testutil.go`

```
package testutil

WriteGzJSON(t *testing.T, path string, v any)
ReadGzJSON(t *testing.T, path string, v any)
MinimalRoot(t *testing.T, dir string)
```

**MinimalRoot** replica o que hoje está em `compile_test.go` (minimalRoot) e
`tests/compile_test.go` (minimalStrategistRoot): cria `active.yaml`, `personas/epic.yaml`,
`roles/default.yaml` em dir.

**Exclusões:** `captureStdout` e `freshArtifactDir` ficam em `cmd_test.go` — são
específicos do CLI e não têm reutilização fora daquele contexto.

**`helpers_test.go` em `package compile` NÃO é migrado** — usa a função interna
`writeGzJSON` da produção (não é o helper de teste), necessário para whitebox testing.

### Migrações por arquivo

| Arquivo | Remove | Adiciona import |
|---------|--------|-----------------|
| `internal/compile/compile_test.go` | `readGzJSON`, `minimalRoot` | `testutil.ReadGzJSON`, `testutil.MinimalRoot` |
| `internal/stale/stale_test.go` | `writeGzJSON` | `testutil.WriteGzJSON` |
| `cmd/strategist/cmd_test.go` | `writeGzJSON`, `minimalStrategistRoot` | `testutil.WriteGzJSON`, `testutil.MinimalRoot` |
| `tests/compile_test.go` | `readGzJSON`, `minimalStrategistRoot` | `testutil.ReadGzJSON`, `testutil.MinimalRoot` |

`tests/install_test.go` não usa helpers de gz/minimal — sem mudança.

---

## Sprint 2 — testdata/ + build tags

### testdata/ para internal/compile

Criar `internal/compile/testdata/` com fixtures YAML:

```
internal/compile/testdata/
├── valid-minimal/
│   ├── active.yaml          — "mode: full"
│   ├── personas/epic.yaml   — "name: Epic"
│   └── roles/default.yaml   — "name: Default"
├── valid-with-knowledge/
│   ├── active.yaml
│   ├── personas/
│   ├── roles/
│   └── knowledge.index.yaml — sources com 1 entry
└── invalid-yaml/
    └── active.yaml          — YAML malformado: ": invalid:"
```

Adaptar `TestCompileConfig` para usar `valid-minimal/` via `os.DirFS` / cópia para TempDir.

### Build tags em tests/

Adicionar a `tests/compile_test.go` e `tests/install_test.go` e `tests/stale_test.go`:

```go
//go:build integration
```

E no CI, step adicional após o step de race tests:

```yaml
- name: Integration tests
  run: go test -tags=integration -race ./tests/...
```

---

## Sprint 3 — Benchmarks

### `internal/compile/bench_test.go` (novo arquivo)

```
BenchmarkCompileAll(b *testing.B)  — invoca compile.All(dir, outDir) num loop
BenchmarkCompileConfig(b *testing.B) — invoca compile.Config(dir, out)
```

Setup: usa MinimalRoot de testutil, cria o tree uma vez fora do loop, reseta timer.

### `internal/stale/bench_test.go` (novo arquivo)

```
BenchmarkIsStale_Fresh(b *testing.B)  — artifact fresh, sem sources
BenchmarkIsStale_Stale(b *testing.B)  — artifact stale, source modificado
```

### Makefile target

Adicionar ao `.PHONY` e criar target:
```makefile
bench:
	go test -bench=. -benchmem ./...
```

---

## Invariantes que NÃO mudam

- Nenhum teste existente pode ser removido ou enfraquecido (contract: test-integrity.md)
- `helpers_test.go` em `package compile` mantém whitebox access
- Coverage gate ≥ 90% deve continuar passando após cada sprint
- `go test -race ./...` deve continuar verde
