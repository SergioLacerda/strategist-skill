# Design: Big Bang Go Migration
**Mission ID:** bigbang-go-20260529  
**Date:** 2026-05-29 (atualizado 2026-05-29 — rev2: engineering standards)

---

## Princípios de Engenharia (World-Class Gate)

### Clean Architecture — Regra de Dependência

```
cmd/strategist
    └── internal/install ──→ internal/domain (interfaces)
    └── internal/compile ──→ internal/domain
    └── internal/stale   ──→ internal/domain
    └── internal/embed   ──→ internal/domain
                              └── stdlib apenas
```

Nenhum import pode inverter essa direção. `internal/domain` contém:
- Tipos de dados core (não podem importar nada além de stdlib)
- Interfaces (Ports) que os outros pacotes implementam
- Permite trocar implementações sem tocar `cmd/` ou outros pacotes

### Guardrails — `.golangci.yaml`

```yaml
linters:
  enable:
    - errcheck       # todo error deve ser tratado ou explicitamente ignorado (_)
    - gosec          # detecta padrões inseguros (G304: file path injection, etc.)
    - staticcheck    # análise estática avançada (deprecated, unreachable, etc.)
    - govet          # go vet embutido
    - revive         # substituto moderno do golint
    - wrapcheck      # erros de funções externas devem ser wrapped com %w
    - exhaustive     # switch em enums deve cobrir todos os casos
    - gocognit       # complexidade cognitiva máxima por função (limite: 15)
    - noctx          # funções HTTP devem receber context.Context
    - testifylint    # uso correto de testify assertions

linters-settings:
  gocognit:
    min-complexity: 15
  govet:
    enable-all: true
  errcheck:
    check-type-assertions: true
    check-blank: true
```

**Gate de CI:** `golangci-lint run ./...` bloqueia merge se falhar. Nenhum `//nolint` sem comentário explicando o motivo.

### Padrão de Error Handling

```go
// CORRETO: erro wrapped com contexto de onde veio
if err != nil {
    return fmt.Errorf("compile config: read active.yaml: %w", err)
}

// ERRADO: erro sem contexto
if err != nil {
    return err
}

// ERRADO: panic fora de main()
if err != nil {
    panic(err)
}
```

Regras:
- `fmt.Errorf("package/operation: %w", err)` em todos os pontos de retorno
- `panic()` proibido fora de `cmd/strategist/main.go`
- `log.Fatal` proibido fora de `cmd/`
- Sentinel errors com `errors.New()` em `internal/domain/errors.go` para casos que callers precisam checar com `errors.Is()`

### Padrão de Testes Go

```go
func TestIsStale(t *testing.T) {
    tests := []struct {
        name    string
        setup   func(t *testing.T, dir string)
        want    bool
        wantErr bool
    }{
        {
            name:  "absent artifact returns stale",
            setup: func(t *testing.T, dir string) {}, // noop
            want:  true,
        },
        {
            name: "fresh artifact with no sources",
            setup: func(t *testing.T, dir string) {
                // criar artifact válido + manifest
            },
            want: false,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dir := t.TempDir() // cleanup automático
            tt.setup(t, dir)
            got, err := IsStale(filepath.Join(dir, "artifact.gz"))
            if (err != nil) != tt.wantErr {
                t.Fatalf("IsStale() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("IsStale() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

Regras:
- 100% table-driven com `t.Run(tt.name, ...)`
- `t.TempDir()` — sem `os.MkdirTemp` + `defer os.RemoveAll` manual
- Blackbox tests (`package compile_test`) para APIs públicas dos pacotes
- Whitebox tests (`package compile`) apenas para helpers internos
- `require.NoError` (testify) para erros que bloqueiam o teste; `assert.*` para verificações não-bloqueantes
- `go test -race ./...` — race detector sempre habilitado em CI
- Nenhum `time.Sleep` em testes — usar channels ou `t.Helper()` helpers com polling limitado

---

## Sequência de Execução (Big Bang)

Sem fases incrementais. Uma única sequência que leva o repo do estado atual ao estado Go.

### Bloco 0 — Preparação (não muda comportamento)

**0.1** Mover conteúdo de `strategist/` para `defaults/`:
```bash
mkdir -p defaults
cp -r strategist/SKILL.md strategist/personas strategist/roles \
       strategist/contracts strategist/schemas strategist/templates \
       strategist/skills strategist/memory strategist/knowledge.index.yaml \
       strategist/skill.yaml strategist/protocol.md defaults/
```

**0.2** Inicializar módulo Go:
```bash
go mod init github.com/SergioLacerda/strategist-skill
go mod tidy
```

**0.3** Criar estrutura de diretórios (Clean Architecture):
```bash
mkdir -p cmd/strategist internal/{domain,compile,stale,install,embed} tests/fixtures tests/specs
```

**0.4** Criar `.golangci.yaml` com linters mandatórios (ver seção "Guardrails" acima).

**0.5** Criar `Makefile`:
```makefile
.PHONY: build test lint install-local release clean

build:
	go build -o bin/strategist ./cmd/strategist

test:
	go test -race ./...

lint:
	golangci-lint run ./...

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

install-local: build
	install -m 755 bin/strategist ~/.local/bin/strategist

release:
	goreleaser release --clean

clean:
	rm -rf bin/ dist/
```

### Bloco 1 — internal/domain (zero deps)

```go
// internal/domain/types.go
package domain

type CompiledConfig struct {
    Schema     string                 `json:"schema"`
    CompiledAt string                 `json:"compiled_at"`
    Sources    map[string]int64       `json:"sources"` // path → mtime unix
    Active     map[string]interface{} `json:"active"`
    Personas   map[string]interface{} `json:"personas"`
    Roles      map[string]interface{} `json:"roles"`
}
// CompiledDomain, CompiledIndex, CompiledManifest — análogos

type InstallConfig struct {
    Target string // caminho para .strategist/
    Silent bool
    Wizard bool
}

type WizardConfig struct {
    Mode        string
    RolesConfig map[string]interface{}
    BasePath    string
    Provider    string
}

// internal/domain/ports.go — interfaces (Ports)
type Compiler interface {
    CompileAll(root, indexPath string) error
}

type StaleChecker interface {
    IsStale(artifactPath string) (bool, error)
}

type Installer interface {
    Install(cfg InstallConfig) error
}

type FileExtractor interface {
    Extract(targetDir string) error
}

// internal/domain/errors.go — sentinel errors
var (
    ErrArtifactAbsent  = errors.New("artifact does not exist")
    ErrManifestMissing = errors.New("manifest not found")
    ErrSourceStale     = errors.New("source file modified after artifact")
)
```

### Bloco 2 — internal/embed

```go
// internal/embed/defaults.go
package embed

import (
    "embed"
    "io/fs"
    "os"
    "path/filepath"
)

//go:embed all:../../defaults
var defaultsFS embed.FS

// Extractor implements domain.FileExtractor
type Extractor struct{}

// Extract copies embedded defaults into targetDir.
func (e Extractor) Extract(targetDir string) error {
    return fs.WalkDir(defaultsFS, "defaults", func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return fmt.Errorf("embed: walk %s: %w", path, err)
        }
        // strip "defaults/" prefix, write to targetDir
        ...
    })
}
```

### Bloco 3 — internal/stale

```go
// internal/stale/check.go
package stale

// Checker implements domain.StaleChecker
type Checker struct{}

// IsStale returns true if the artifact is absent, stale, or manifest is missing.
// Mirrors the logic of check-stale.sh exactly.
func (c Checker) IsStale(artifactPath string) (bool, error) {
    // 1. artifact exists? → domain.ErrArtifactAbsent (not an error, just stale=true)
    // 2. .manifest.gz in same dir? → domain.ErrManifestMissing
    // 3. for each source in artifact.sources: mtime <= recorded? → domain.ErrSourceStale
    return false, nil
}
```

### Bloco 4 — internal/compile

```go
// internal/compile/config.go
package compile

// CompileConfig reads .strategist/active.yaml + personas/*.yaml + roles/*.yaml
// and writes a gzipped JSON artifact to outputPath.
func CompileConfig(root, outputPath string) error {
    // error wrapping: fmt.Errorf("compile config: read active.yaml: %w", err)
}

// internal/compile/all.go
// Compiler implements domain.Compiler
type Compiler struct{}

func (c Compiler) CompileAll(root, indexPath string) error {
    if err := CompileConfig(root, ...); err != nil {
        return fmt.Errorf("compile all: config: %w", err)
    }
    // Domain, Index em sequência
    // WriteManifest só se tudo verde
}
```

### Bloco 5 — internal/install

```go
// internal/install/installer.go
package install

// Service implements domain.Installer
type Service struct {
    Extractor domain.FileExtractor
    Compiler  domain.Compiler
}

func (s Service) Install(cfg domain.InstallConfig) error {
    if err := s.Extractor.Extract(cfg.Target); err != nil {
        return fmt.Errorf("install: extract defaults: %w", err)
    }
    if cfg.Wizard {
        wc, err := RunWizard()
        if err != nil {
            return fmt.Errorf("install: wizard: %w", err)
        }
        // apply wc to active.yaml
    }
    // gitignore, shim, non-blocking compile
    return nil
}
```

### Bloco 6 — cmd/strategist

```go
// cmd/strategist/root.go — apenas wiring, zero lógica
var rootCmd = &cobra.Command{Use: "strategist", Short: "Strategist skill CLI"}

// cmd/strategist/install.go — injeta dependências concretas
var installCmd = &cobra.Command{
    Use:  "install",
    RunE: func(cmd *cobra.Command, args []string) error {
        svc := install.Service{
            Extractor: embed.Extractor{},
            Compiler:  compile.Compiler{},
        }
        return svc.Install(domain.InstallConfig{...})
    },
}

// cmd/strategist/check_stale.go
var checkStaleCmd = &cobra.Command{
    Use:  "check-stale <artifact.gz>",
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        isStale, err := stale.Checker{}.IsStale(args[0])
        if err != nil {
            return err
        }
        if isStale {
            os.Exit(1)
        }
        return nil
    },
}
```

### Bloco 7 — tests/

Todos os testes seguem o padrão table-driven com `t.TempDir()`.

```go
// tests/stale_test.go — blackbox (package stale_test)
func TestIsStale(t *testing.T) {
    tests := []struct { name string; setup func(*testing.T, string); want bool }{
        {"absent artifact", func(t *testing.T, dir string) {}, true},
        {"fresh artifact",  setupFreshArtifact, false},
        {"stale source",    setupStaleSource,   true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dir := t.TempDir()
            tt.setup(t, dir)
            got, err := stale.Checker{}.IsStale(filepath.Join(dir, "artifact.gz"))
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}

// tests/install_test.go — usa mock de domain.FileExtractor para isolar de filesystem real
type mockExtractor struct{ called bool }
func (m *mockExtractor) Extract(targetDir string) error { m.called = true; return nil }
```

### Bloco 8 — goreleaser + CI

```yaml
# .goreleaser.yaml
builds:
  - main: ./cmd/strategist
    binary: strategist
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    env: [CGO_ENABLED=0]

checksum:
  name_template: SHA256SUMS
```

```yaml
# .github/workflows/test.yml — lint gate bloqueia merge
jobs:
  test:
    steps:
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go vet ./...
      - uses: golangci/golangci-lint-action@v6
        with: { version: latest }
      - run: go test -race ./...
      - run: go build ./...
```

Atualizar `.github/workflows/release.yml` para usar `goreleaser/goreleaser-action`.

### Bloco 9 — Engineering Review Gate

Antes de qualquer `git rm`, todos os pacotes passam pela revisão:

- `golangci-lint run ./...` verde sem `//nolint` não documentado
- `go test -race ./...` verde
- `go tool cover -func=coverage.out` — mínimo 80% em `internal/compile` e `internal/stale`
- Toda função/tipo exportado tem comentário godoc

### Bloco 10 — Limpeza e finalização

**10.1** Deletar arquivos shell substituídos:
```bash
git rm strategist/install.sh
git rm strategist/scripts/
git rm bootstrap.ps1
git rm strategist/tests/harness/ strategist/tests/validators/ \
       strategist/tests/unit/ strategist/tests/integration/ \
       strategist/tests/run-tests.sh
```

**10.2** Atualizar `defaults/SKILL.md` — substituir as 4 linhas de referência shell.

**10.3** Reescrever `bootstrap.sh` (30 linhas).

**10.4** Atualizar `readme.md` — seção de instalação e seção de testes.

**10.5** `go test -race ./...` verde final.

**10.6** `git commit -m "feat: migrate to Go binary distribution (big bang)"`

---

## Makefile

```makefile
.PHONY: build test lint cover install-local release clean

build:
	go build -o bin/strategist ./cmd/strategist

test:
	go test -race ./...

lint:
	golangci-lint run ./...

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

install-local: build
	install -m 755 bin/strategist ~/.local/bin/strategist

release:
	goreleaser release --clean

clean:
	rm -rf bin/ dist/
```

---

## Riscos do Big Bang

| Risco | Mitigação |
|-------|----------|
| Wizard interativo complexo em Go | Usar `charmbracelet/huh` — cobre os casos bem; se travar, wizard pode ser simplificado sem perder silent mode |
| `//go:embed all:defaults` com muitos arquivos | Testar tamanho do binário; tipicamente <15MB com YAML embedded |
| PATH do binário em ambientes CI/CD | Documentar: `~/.local/bin` ou `go install` são as rotas padrão |
| Tests quebram durante a construção | Construir por blocos (0→1→2→3→4→5→6→7→8); cada bloco tem `go test` verde antes de avançar |
