# Padrões *world-class engineering* para testes em Go

Em Go, o padrão mais forte é **usar o `testing` package da standard library como base** e adicionar ferramentas só onde elas aumentam segurança, legibilidade ou automação. O pacote `testing` é feito para trabalhar com `go test` e suporta testes, benchmarks, subtests, fuzzing e exemplos executáveis. ([Go Packages][1])

---

# 1. Pirâmide de testes recomendada

Para projetos Go maduros:

```text
Unit tests rápidos
  ↓
Table-driven tests
  ↓
Integration tests com build tags
  ↓
Contract/API tests
  ↓
Fuzz tests para parsers, codecs, validators
  ↓
E2E tests mínimos
```

A maior parte deve ser:

```text
unit + table-driven + contract tests
```

E uma parte menor:

```text
integration + e2e
```

---

# 2. Estrutura de diretórios

## Padrão idiomático Go

```text
myapp/
├── go.mod
├── internal/
│   └── user/
│       ├── service.go
│       ├── service_test.go
│       └── service_external_test.go
├── pkg/
├── cmd/
│   └── myapp/
│       ├── main.go
│       └── main_test.go
└── test/
    ├── integration/
    ├── fixtures/
    └── testdata/
```

## Regra prática

Coloque testes unitários perto do código:

```text
service.go
service_test.go
```

Use `test/` ou `tests/` só para cenários mais amplos:

```text
test/integration
test/e2e
test/fixtures
```

---

# 3. Testes no mesmo package vs `_test`

Em Go, você tem dois estilos.

## Mesmo package

```go
package user
```

Bom para testar comportamento interno quando necessário.

## Package externo

```go
package user_test
```

Melhor para testar API pública como consumidor real.

Minha recomendação:

| Caso                           | Use                                                        |
| ------------------------------ | ---------------------------------------------------------- |
| Testar API pública             | `package x_test`                                           |
| Testar detalhe interno crítico | `package x`                                                |
| Testar helper privado demais   | prefira extrair lógica ou testar via comportamento público |

Padrão world-class:

```text
teste comportamento público primeiro
teste internals só quando houver valor real
```

---

# 4. Table-driven tests

Esse é um dos padrões mais idiomáticos em Go.

```go
func TestNormalizeEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "lowercase and trim",
			input: " USER@Example.COM ",
			want:  "user@example.com",
		},
		{
			name:    "empty email",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeEmail(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
```

Use table-driven tests para:

```text
validações
parsers
normalização
regras de domínio
mapeamentos
edge cases
```

---

# 5. Subtests com `t.Run`

Use `t.Run` para dar nome claro a cenários.

```go
func TestCreateUser(t *testing.T) {
	t.Run("valid input creates user", func(t *testing.T) {
		// ...
	})

	t.Run("duplicate email returns conflict", func(t *testing.T) {
		// ...
	})
}
```

Isso melhora:

```text
debug
logs
CI output
test selection
legibilidade
```

---

# 6. `t.Parallel()` com cuidado

Use paralelismo para acelerar suíte, mas com regras:

```text
bom:
- testes puros
- testes sem estado global
- testes com DB isolado
- testes com filesystem temporário próprio

evitar:
- testes que mexem em env global
- testes que alteram working directory
- testes que usam portas fixas
- testes que compartilham mocks mutáveis
```

Se usar `t.Setenv`, cuidado: ele interage mal com paralelismo quando há alteração de ambiente compartilhado.

---

# 7. Fixtures e `testdata`

Go tem convenção forte para diretórios `testdata`.

```text
parser/
├── parser.go
├── parser_test.go
└── testdata/
    ├── valid.json
    ├── invalid.json
    └── legacy-format.json
```

Use para:

```text
arquivos de entrada
golden files
payloads JSON
fixtures de parser
respostas HTTP simuladas
```

---

# 8. Golden files

Use golden files quando a saída é grande ou estrutural:

```text
testdata/
├── expected_report.golden.json
└── expected_markdown.golden.md
```

Exemplo:

```go
func TestGenerateReport(t *testing.T) {
	got := GenerateReport(sampleInput())

	wantBytes, err := os.ReadFile("testdata/expected_report.golden.md")
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(string(wantBytes), got); diff != "" {
		t.Fatalf("report mismatch (-want +got):\n%s", diff)
	}
}
```

Para diff bom, use:

```bash
go get github.com/google/go-cmp/cmp
```

`go-cmp` é praticamente padrão de fato para comparar estruturas complexas com diffs legíveis.

---

# 9. Evite assert libraries pesadas por padrão

Em Go world-class, prefira:

```go
if got != want {
	t.Fatalf("got %v, want %v", got, want)
}
```

Use bibliotecas como `testify` com parcimônia. Elas são úteis, mas podem deixar testes menos idiomáticos se usadas como muleta.

Minha regra:

```text
standard library primeiro
go-cmp para diff complexo
testify/mock só se realmente necessário
```

---

# 10. Mocks: prefira interfaces pequenas

Em Go, o melhor mock é uma interface pequena.

```go
type UserStore interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	Save(ctx context.Context, user User) error
}
```

Teste com fake simples:

```go
type fakeUserStore struct {
	users map[string]User
	err   error
}

func (f *fakeUserStore) Save(ctx context.Context, user User) error {
	if f.err != nil {
		return f.err
	}
	f.users[user.Email] = user
	return nil
}
```

Padrão recomendado:

| Situação             | Melhor                      |
| -------------------- | --------------------------- |
| Interface pequena    | fake manual                 |
| Muitos métodos       | repensar interface          |
| API externa complexa | contract test + fake server |
| Mock gerado          | só quando ganho for claro   |

---

# 11. Testes HTTP

Use `httptest`.

```go
func TestHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	rec := httptest.NewRecorder()

	handler := NewHandler(fakeService{})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
```

Para cliente HTTP externo, use `httptest.Server`.

```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}))
defer server.Close()
```

---

# 12. Testes com banco

Padrão maduro:

```text
unit test:
  fake repository

integration test:
  banco real em container ou service CI

contract test:
  garante comportamento esperado do repository
```

Evite testar SQL crítico só com mock. SQL, migrations, índices e constraints precisam de teste real.

Estratégias:

```text
- testcontainers-go
- docker compose no CI
- banco temporário por pacote
- transação rollback por teste
- schema/migration aplicada no setup
```

Para integração, use build tags:

```go
//go:build integration

package repository_test
```

Rodar:

```bash
go test ./...
go test -tags=integration ./...
```

---

# 13. Fuzz tests

Use fuzzing para funções que recebem input externo:

```text
parsers
decoders
validators
normalizers
protocol handlers
URL/path processing
JSON/YAML processing
```

Go possui fuzzing integrado ao `testing` package; a documentação oficial descreve `testing.F` e o uso via `go test` com fuzzing. ([Go Packages][1])

Exemplo:

```go
func FuzzParseUserID(f *testing.F) {
	f.Add("123")
	f.Add("user-456")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		_, _ = ParseUserID(input)
	})
}
```

Um fuzz test básico não precisa provar resultado exato em todos os casos; ele pode garantir propriedades:

```text
não panica
não faz loop infinito
não aceita input inválido perigoso
round-trip preserva valor
```

---

# 14. Property-based testing

Use quando há invariantes fortes.

Exemplo:

```text
Encode(Decode(x)) == x
Normalize(Normalize(x)) == Normalize(x)
Sort(x) is ordered
Parse(Format(x)) == x
```

Ferramentas possíveis:

```text
testing/quick
fuzzing nativo
rapid
gopter
```

Mas comece com fuzzing nativo antes de adicionar dependência.

---

# 15. Race detector

Sempre rode race detector no CI pelo menos em suite crítica:

```bash
go test -race ./...
```

Use especialmente quando há:

```text
goroutines
channels
maps compartilhados
caches
workers
HTTP servers
background jobs
```

---

# 16. Coverage com critério inteligente

Go suporta coverage com:

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

A documentação oficial também descreve suporte para coverage em integração desde Go 1.20, útil para testes mais pesados que rodam binários completos. ([Go.dev][2])

Mas cuidado:

```text
coverage alto ≠ qualidade alta
coverage baixo em domínio crítico = risco
```

Minha regra:

| Camada             |            Meta sugerida |
| ------------------ | -----------------------: |
| Domínio puro       |                   85–95% |
| Parsers/validators |              90%+ + fuzz |
| Use cases críticos |                   80–90% |
| Handlers HTTP      |                   70–85% |
| Infra/adapters     |    integração > coverage |
| CLI                | golden/integration tests |

---

# 17. Benchmarks

Use benchmarks para código sensível a performance.

```go
func BenchmarkNormalizeEmail(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NormalizeEmail(" USER@Example.COM ")
	}
}
```

Rodar:

```bash
go test -bench=. ./...
```

Use para:

```text
serialização
parsers
hot paths
cache
algoritmos
roteamento
compressão
```

Não transforme benchmark em vanity metric. Use para impedir regressões onde performance importa.

---

# 18. Lint e qualidade no pipeline

Use `golangci-lint`. Ele é um runner de linters para Go com execução paralela, cache, configuração YAML e integração com IDEs. ([GolangCI-Lint][3])

Checks recomendados:

```bash
gofmt
go vet
golangci-lint run
go test ./...
go test -race ./...
go test -coverprofile=coverage.out ./...
```

Linters úteis:

```text
govet
staticcheck
errcheck
ineffassign
unused
gocritic
bodyclose
misspell
revive
unparam
prealloc
nilerr
contextcheck
```

---

# 19. Testes de arquitetura

Para projetos grandes, teste arquitetura também.

Exemplos de regras:

```text
domain não importa infrastructure
internal/app não importa cmd
handlers não acessam DB diretamente
packages públicos não dependem de internal indevido
```

Você pode criar teste que roda `go list` e analisa imports:

```go
func TestDomainDoesNotImportInfrastructure(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "./internal/domain/...").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(out), "/internal/infrastructure/") {
		t.Fatal("domain must not import infrastructure")
	}
}
```

Esse padrão é muito alinhado ao que você vem chamando de governança: teste não é só validar comportamento; é também conter drift arquitetural.

---

# 20. Testes de CLI

Para CLI Go:

```text
- teste parsing de args
- teste stdout/stderr
- teste exit code
- teste config loading
- teste golden output
```

Idealmente, separe:

```text
core logic testável
cmd thin layer
```

Exemplo:

```go
func TestRootCommandVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--version"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(stdout.String(), "version") {
		t.Fatalf("expected version output, got %q", stdout.String())
	}
}
```

---

# 21. Testes de configuração

Para sistemas governados ou skills, teste configs como contrato.

```text
- YAML válido
- campos obrigatórios
- defaults aplicados
- erro em config inválida
- compatibilidade entre versões
```

Em Go, você pode validar com structs:

```go
type SkillManifest struct {
	ID          string   `yaml:"id"`
	Version     string   `yaml:"version"`
	Risk        string   `yaml:"risk"`
	Triggers    []string `yaml:"triggers"`
	Description string   `yaml:"description"`
}
```

E testar:

```go
func TestSkillManifestIsValid(t *testing.T) {
	manifest := loadManifest(t, "skill.yaml")

	if manifest.ID == "" {
		t.Fatal("id is required")
	}
	if len(manifest.Triggers) == 0 {
		t.Fatal("triggers are required")
	}
}
```

---

# 22. CI/CD world-class para Go

Pipeline recomendado:

```text
format
→ lint
→ unit tests
→ race tests
→ integration tests
→ coverage
→ build
→ vulnerability scan
→ release artifact
```

Exemplo:

```yaml
name: go-ci

on:
  pull_request:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Format check
        run: |
          test -z "$(gofmt -l .)"

      - name: Vet
        run: go vet ./...

      - name: Unit tests
        run: go test ./...

      - name: Race tests
        run: go test -race ./...

      - name: Coverage
        run: go test -coverprofile=coverage.out ./...

      - name: Build
        run: go build ./...
```

Para produção, adicione:

```text
golangci-lint
govulncheck
integration tests
release build
SBOM/checksum
```

---

# 23. Padrões de nome

Use nomes descritivos:

```go
func TestUserService_Create(t *testing.T)
func TestUserService_Create_DuplicateEmail(t *testing.T)
func TestParseToken_InvalidSignature(t *testing.T)
```

Para table tests:

```go
{
	name: "expired token returns unauthenticated",
}
```

Nunca use:

```text
case1
test2
happy path
bad input
```

Prefira nomes que expliquem o comportamento.

---

# 24. Anti-patterns em testes Go

Evite:

```text
- testar implementação em vez de comportamento
- mocks gigantes
- sleeps em testes concorrentes
- testes dependentes de ordem
- portas fixas
- arquivos temporários fora de t.TempDir()
- env global sem t.Setenv()
- comparar JSON como string crua sem normalizar
- coverage como única métrica
- testes lentos misturados com unit tests
- usar assert library para tudo
- testar private helpers sem necessidade
```

---

# 25. Checklist final

Para Go world-class, eu usaria este checklist:

```text
[ ] Unit tests perto do código
[ ] Table-driven tests para regras e edge cases
[ ] Subtests com nomes claros
[ ] `package x_test` para API pública
[ ] `testdata/` para fixtures
[ ] Golden files para outputs grandes
[ ] `go-cmp` para diff estruturado
[ ] Fakes manuais para interfaces pequenas
[ ] `httptest` para HTTP
[ ] Integration tests com build tag
[ ] Fuzz tests para input externo
[ ] Race detector no CI
[ ] Coverage com foco em código crítico
[ ] Benchmarks para hot paths
[ ] `gofmt`, `go vet`, `golangci-lint`
[ ] Testes de arquitetura com `go list`
[ ] CI separando unit, integration, race e release
```

# Resumo

O padrão world-class em Go é:

```text
standard library first
table-driven tests
small interfaces + fakes
test behavior, not internals
integration tests isolados
fuzzing para input externo
race detector para concorrência
coverage como sinal, não objetivo
CI rápido, determinístico e com qualidade incremental
```

Para o seu contexto de SDD/skill, eu adicionaria uma camada extra:

```text
contract tests
governance tests
routing tests
golden output tests
architecture drift tests
```

[1]: https://pkg.go.dev/testing?utm_source=chatgpt.com "testing package - testing"
[2]: https://go.dev/doc/build-cover?utm_source=chatgpt.com "Coverage profiling support for integration tests"
[3]: https://golangci-lint.run/?utm_source=chatgpt.com "Golangci-lint"
