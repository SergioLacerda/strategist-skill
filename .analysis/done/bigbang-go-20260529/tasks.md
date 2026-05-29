# Tasks: Big Bang Go Migration
**Mission ID:** bigbang-go-20260529  
**Date:** 2026-05-29 (atualizado 2026-05-29 — rev2: engineering standards)  
**Scope:** escreve fora de `.analysis/` — modifica/deleta arquivos em todo o repositório

---

## Bloco 0 — Preparação

- [ ] **0.1** `mkdir -p defaults && cp -r` conteúdo de `strategist/` (SKILL.md, personas, roles, contracts, schemas, templates, skills, memory, knowledge.index.yaml, skill.yaml, protocol.md) para `defaults/`
- [ ] **0.2** `go mod init github.com/SergioLacerda/strategist-skill` na raiz
- [ ] **0.3** Criar estrutura de diretórios seguindo Clean Architecture:
  ```
  cmd/strategist/
  internal/compile/
  internal/stale/
  internal/install/
  internal/embed/
  internal/domain/
  tests/
  ```
  > **Nota:** `internal/` (não `pkg/`) — impede importação externa acidental; `domain/` centraliza tipos e interfaces que outros pacotes dependem.
- [ ] **0.4** Criar `.golangci.yaml` com linters mandatórios (errcheck, gosec, staticcheck, govet, revive, wrapcheck, exhaustive, gocognit) — **bloqueia CI se falhar**
- [ ] **0.5** Criar `Makefile` com targets: `build`, `test`, `lint`, `install-local`, `release`, `clean`

## Bloco 1 — internal/domain (camada central, zero dependências externas)

- [ ] **1.1** Criar `internal/domain/types.go` — tipos e structs core (CompiledConfig, CompiledDomain, CompiledIndex, CompiledManifest, InstallConfig, WizardConfig)
- [ ] **1.2** Criar `internal/domain/ports.go` — interfaces que outros pacotes implementam:
  ```go
  type Compiler interface { CompileAll(root, indexPath string) error }
  type StaleChecker interface { IsStale(artifactPath string) (bool, error) }
  type Installer interface { Install(cfg InstallConfig) error }
  type FileExtractor interface { Extract(targetDir string) error }
  ```
- [ ] **1.3** `go build ./internal/domain/...` verde — **sem imports externos além de stdlib**

## Bloco 2 — internal/embed

- [ ] **2.1** Criar `internal/embed/defaults.go` com `//go:embed all:../../defaults` e implementação de `domain.FileExtractor`
- [ ] **2.2** `go build ./...` verde

## Bloco 3 — internal/stale

- [ ] **3.1** Criar `internal/stale/check.go` — implementa `domain.StaleChecker`; porta exata da lógica de `check-stale.sh` (mtime, manifest check, sources)
- [ ] **3.2** Criar `tests/stale_test.go` — **table-driven** com `t.Run()`:
  ```go
  tests := []struct {
      name    string
      setup   func(dir string)
      want    bool
      wantErr bool
  }{
      {"absent artifact", setupAbsent, true, false},
      {"missing manifest", setupNoManifest, true, false},
      {"fresh empty sources", setupFresh, false, false},
      {"stale source file", setupStale, true, false},
  }
  for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) { ... })
  }
  ```
  Usar `t.TempDir()` — sem `defer os.RemoveAll()` manual.
- [ ] **3.3** `go test ./internal/stale/...` verde

## Bloco 4 — internal/compile

- [ ] **4.1** Criar `internal/compile/config.go` — porta de `compile-config.sh`; recebe `root string`, retorna `error` com wrapping (`fmt.Errorf("compile config: %w", err)`)
- [ ] **4.2** Criar `internal/compile/domain.go` — porta de `compile-domain.sh`
- [ ] **4.3** Criar `internal/compile/index.go` — porta de `compile-knowledge-index.sh`
- [ ] **4.4** Criar `internal/compile/all.go` — implementa `domain.Compiler`; orquestra 4.1-4.3, escreve `.manifest.gz` só no sucesso total
- [ ] **4.5** Criar `tests/compile_test.go` — **table-driven**; testa Config, Domain, All com `t.TempDir()`; usa blackbox test (`package compile_test`)
- [ ] **4.6** `go test ./internal/compile/...` verde

## Bloco 5 — internal/install (maior bloco)

- [ ] **5.1** Criar `internal/install/installer.go` — implementa `domain.Installer`; modo silent: extrai via `domain.FileExtractor`, escreve `active.yaml`, adiciona `.gitignore`; **context.Context** no signature para cancelamento futuro
- [ ] **5.2** Criar `internal/install/wizard.go` — modo interativo com `charmbracelet/huh`; retorna `domain.WizardConfig`; se `huh` travar, wizard cai back para stdin simples (sem perder silent mode)
- [ ] **5.3** Criar `internal/install/shim.go` — cria `.claude/skills/strategist/skill.yaml` (agent shim)
- [ ] **5.4** Criar `internal/install/compile.go` — chama `domain.Compiler` ao final do install (non-blocking: log warn on failure, nunca fatal)
- [ ] **5.5** Criar `tests/install_test.go` — **table-driven**; testa silent mode em `t.TempDir()`; valida estrutura de `.strategist/` via paths esperados; usa interface `domain.FileExtractor` mockada — **sem tocar filesystem real**
- [ ] **5.6** `go test ./internal/install/...` verde

## Bloco 6 — cmd/strategist

- [ ] **6.1** Criar `cmd/strategist/main.go` e `root.go` (cobra root); sem lógica — apenas wiring
- [ ] **6.2** Criar `cmd/strategist/install.go` — wraps `internal/install` com flags `--silent`, `--wizard`, `--target`
- [ ] **6.3** Criar `cmd/strategist/compile.go` — wraps `internal/compile` com flag `--root`
- [ ] **6.4** Criar `cmd/strategist/check_stale.go` — wraps `internal/stale`; exit 0=fresh, exit 1=stale
- [ ] **6.5** `go build -o bin/strategist ./cmd/strategist` verde
- [ ] **6.6** Smoke test manual: `bin/strategist install --silent --target /tmp/test-install && bin/strategist check-stale /tmp/test-install/.compiled/.config.gz`

## Bloco 7 — Testes de fixtures

- [ ] **7.1** Criar `tests/fixtures_test.go` — porta de `tests/run-tests.sh` em Go puro; **table-driven** carregando `tests/fixtures/*.yaml`; subtests por fixture file (`t.Run(fixture.Name, ...)`)
- [ ] **7.2** `go test ./tests/...` verde

## Bloco 8 — goreleaser + CI

- [ ] **8.1** Criar `.goreleaser.yaml` — 5 plataformas, binário direto, SHA256SUMS
- [ ] **8.2** Atualizar `.github/workflows/release.yml` — substituir passo "Package release assets" por `goreleaser release`
- [ ] **8.3** Criar `.github/workflows/test.yml` — `go vet ./...` + `golangci-lint run` + `go test ./...` + `go build ./...` em push/PR; **lint gate bloqueia merge**

## Bloco 9 — Engineering Review (world-class gate)

- [ ] **9.1** `golangci-lint run ./...` verde sem supressões (nenhum `//nolint` sem justificativa documentada)
- [ ] **9.2** Revisar error handling em todos os pacotes: todo `error` retornado deve usar `fmt.Errorf("pkg/op: %w", err)`; nenhum `panic()` fora de `main()`; nenhum `log.Fatal` fora de `cmd/`
- [ ] **9.3** Revisar interfaces: todo pacote `internal/` deve expor behavior via interface em `internal/domain/ports.go`; `cmd/` só depende de interfaces, nunca de implementações diretamente
- [ ] **9.4** Revisar testes: 100% table-driven; nenhum `time.Sleep` em testes; nenhum `os.Exit` em test helpers; cobertura mínima 80% nos pacotes `internal/compile` e `internal/stale`
- [ ] **9.5** Godoc: toda função/tipo exportado tem comentário; `go doc ./...` sem warnings
- [ ] **9.6** `go test -race ./...` verde — race detector habilitado

## Bloco 10 — Limpeza e finalização

- [ ] **10.1** `git rm` dos arquivos shell removidos:
  - `strategist/install.sh`
  - `strategist/scripts/` (diretório completo)
  - `bootstrap.ps1`
  - `strategist/tests/harness/`, `validators/`, `unit/`, `integration/`, `run-tests.sh`
- [ ] **10.2** Editar `defaults/SKILL.md` — substituir 4 referências shell por `strategist <command>`
- [ ] **10.3** Reescrever `bootstrap.sh` — ~30 linhas: detect OS/arch, download binary, verify SHA, `strategist install`
- [ ] **10.4** Atualizar `readme.md` — seção de instalação e seção de testes
- [ ] **10.5** `go test -race ./...` verde na suite completa com race detector
- [ ] **10.6** Commit: `feat: migrate to Go binary distribution (big bang)`

---

## Ordem crítica

`0 → 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10`

`internal/domain` primeiro — todos os outros pacotes dependem das interfaces definidas ali. O repo só "quebra" temporariamente no bloco 10.1 (remoção dos shell scripts) — que é o último passo.

**Regra de dependência (Clean Architecture):**
```
cmd/strategist → internal/{install,compile,stale,embed}
internal/install → internal/{compile,stale,embed,domain}
internal/{compile,stale,embed} → internal/domain
internal/domain → (stdlib apenas)
```
Nenhum import pode inverter essa direção. `go vet` + lint detectam ciclos.

---

## Critérios de conclusão

- `go build ./...` sem erros
- `go test -race ./...` verde (race detector habilitado)
- `golangci-lint run ./...` verde sem supressões não documentadas
- `bin/strategist install --silent` produz estrutura correta em tmpdir
- `bin/strategist check-stale` exit 0 para artifact fresco, exit 1 para stale
- `bin/strategist compile` produz 4 artifacts `.gz` em `.compiled/`
- `bootstrap.sh` baixa e verifica binário sem depender de jq/yq
- Nenhum arquivo `.sh` em `strategist/scripts/` ou como instalador raiz (exceto `bootstrap.sh`)
- `defaults/SKILL.md` referencia `strategist <cmd>` — não `sh .strategist/scripts/*.sh`
- Cobertura mínima 80% em `internal/compile` e `internal/stale`
- Toda função exportada tem godoc

---

## Nota — Sniper como referência de boas práticas

> O Sniper já opera sob governança world-class (forbidden behaviors, contratos, zero side effects não autorizados). Em caso de dúvida sobre decisão de engenharia Go (naming, error strategy, interface design), consultar o Sniper offline — ele aplica os mesmos padrões que são esperados no código produzido aqui.
