# Recomendação World-Class para Guardrails em uma Skill `strategist` em Go

Para uma skill **Strategist** em Go, eu desenharia os guardrails como um **funil de governança**, não apenas como linters soltos:

```text
intent → scope → evidence → diff → validation → score
```

Essa ideia já está alinhada com o modelo que você vinha construindo no SDD Harness: impedir “hacks” por meio de **Intent Declaration**, **Scope Locking**, **Evidence-Based Change**, **Minimal Valid Change** e **Escalation Contract**. 

---

# 1. Guardrails essenciais

## 1.1 Guardrail contra “hack”

Crie um mandate explícito:

```md
# NO HACK WITHOUT EVIDENCE

The agent MUST NOT introduce tactical shortcuts unless explicitly justified.

Forbidden by default:

- suppressing errors without diagnosis
- weakening tests
- broad `recover()` or silent error handling
- `_ = err`
- changing public contracts without escalation
- adding abstractions without evidence
- bypassing architectural layers
- disabling linters/tests/checks
- introducing global mutable state
- adding dependencies without approval

Any exception requires:

1. diagnosis
2. evidence
3. explicit trade-off
4. temporary marker
5. follow-up task
```

Esse mandate é mais forte do que “não faça hacks”, porque cria uma regra operacional:

> **hack pode até existir como exceção, mas nunca invisível.**

---

## 1.2 Guardrail contra bad code smells em Go

Para Go, eu bloquearia especialmente:

| Smell                                                   | Risco                         |
| ------------------------------------------------------- | ----------------------------- |
| função grande demais                                    | baixa testabilidade           |
| pacote com responsabilidade demais                      | acoplamento e confusão        |
| interfaces prematuras                                   | overengineering               |
| `panic` fora de bootstrap/teste                         | quebra de controle de erro    |
| `context.Background()` dentro de lógica de negócio      | perda de cancelamento/timeout |
| erro ignorado                                           | bug silencioso                |
| dependência concreta onde deveria haver porta/interface | acoplamento                   |
| estado global mutável                                   | testes instáveis              |
| goroutine sem cancelamento                              | leak                          |
| canal sem ownership claro                               | deadlock/race                 |
| teste que só valida implementação interna               | fragilidade                   |
| mock excessivo                                          | teste artificial              |
| alteração de teste para “passar”                        | validation cheating           |

O pacote `testing` da standard library já suporta testes, benchmarks, subtests, fuzzing e exemplos executáveis; eu usaria isso como base, adicionando ferramentas só onde aumentarem segurança ou clareza. ([Go Packages][1])

---

# 2. Ferramentas recomendadas

## Camada mínima obrigatória

```bash
go test ./...
go test -race ./...
go vet ./...
go fmt ./...
go mod tidy
go mod verify
```

## Lint central

Use `golangci-lint` como agregador. Ele roda linters em paralelo, tem cache, YAML config e integra com IDEs/CI. ([GolangCI-Lint][2])

Linters que eu ligaria para uma skill Strategist:

```yaml
linters:
  enable:
    - govet
    - staticcheck
    - errcheck
    - ineffassign
    - unused
    - revive
    - gocritic
    - gosec
    - bodyclose
    - contextcheck
    - cyclop
    - dupl
    - misspell
    - unconvert
    - prealloc
    - whitespace
```

`errcheck`, por exemplo, é importante porque erros ignorados em Go frequentemente viram bugs críticos; o próprio catálogo do `golangci-lint` descreve esse linter como verificador de erros não tratados. ([GolangCI-Lint][3])

## Segurança

```bash
govulncheck ./...
gosec ./...
```

`govulncheck` é a ferramenta oficial do ecossistema Go para detectar vulnerabilidades conhecidas afetando código Go; ela reduz ruído ao identificar chamadas diretas ou indiretas a símbolos vulneráveis. ([Go.dev][4])

`gosec` complementa isso como SAST para Go, inspecionando AST e SSA em busca de problemas de segurança. ([GitHub][5])

## Supply chain

```bash
go mod verify
govulncheck ./...
```

E em GitHub Actions:

* Dependabot
* CodeQL
* OpenSSF Scorecard
* branch protection
* signed releases, quando fizer sentido

OpenSSF Scorecard é voltado a medir a postura de segurança de repositórios open source e possui GitHub Action oficial. ([GitHub][6])

---

# 3. Dependências cíclicas

Em Go, ciclos de importação reais já são bloqueados pelo compilador. Mesmo assim, eu criaria guardrails para **ciclos arquiteturais indiretos**, que não aparecem necessariamente como `import cycle`, mas indicam design ruim.

Exemplo:

```text
strategist → runtime → governance → strategist
```

Mesmo que tecnicamente compile por interfaces ou pacotes intermediários, isso pode ser um ciclo conceitual.

## Regra recomendada

```md
# ARCHITECTURE DEPENDENCY DIRECTION

Allowed direction:

cmd
 ↓
internal/app
 ↓
internal/domain
 ↓
internal/ports

internal/adapters
 ↓
internal/app / internal/ports

Forbidden:

domain → adapters
domain → infrastructure
app → concrete infrastructure
runtime → strategist implementation
governance → concrete provider
```

## Estrutura Go sugerida

```text
strategist/
├── cmd/
│   └── strategist/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── orchestrator.go
│   │   └── mission_service.go
│   ├── domain/
│   │   ├── mission.go
│   │   ├── contract.go
│   │   └── policy.go
│   ├── ports/
│   │   ├── repository.go
│   │   ├── validator.go
│   │   └── provider.go
│   ├── adapters/
│   │   ├── filesystem/
│   │   ├── github/
│   │   └── llm/
│   └── runtime/
│       ├── state.go
│       └── progress.go
├── test/
│   ├── integration/
│   └── fixtures/
└── go.mod
```

Regra world-class:

```text
domain não conhece infraestrutura.
application orquestra portas.
adapters implementam portas.
cmd só faz wiring.
```

---

# 4. Guardrails específicos para o Strategist

Como o Strategist é uma skill orquestradora, os riscos principais não são só bugs de Go. São também:

* executar sem diagnóstico;
* executar sem plano aprovado;
* expandir escopo;
* gerar plano bonito mas não validável;
* chamar provider errado;
* pular skill registrada;
* misturar análise com execução.

Eu criaria este contrato:

```yaml
execution_contract:
  skill: strategist
  task_type: mission_orchestration
  mode: diagnostic_orchestration

  required_phases:
    - intake
    - discovery
    - refinement
    - approval_gate

  forbidden_by_default:
    - direct_code_write
    - execution_without_approval
    - scope_expansion_without_evidence
    - provider_bypass
    - raw_shell_without_registered_skill
    - modifying_tests_to_fit_behavior

  required_artifacts:
    - mission_intake
    - scout_analysis
    - engineer_plan
    - approval_record

  validation:
    required:
      - go_test
      - go_vet
      - golangci_lint
      - govulncheck
      - architecture_guard
```

Esse modelo conversa com sua arquitetura de skill como **intenção governada**, não apenas comando CLI. O agente deve pensar em skill, e o runtime traduzir para comandos e validações. 

---

# 5. CI/CD recomendado

## Pipeline mínimo

```yaml
name: ci

on:
  pull_request:
  push:
    branches: [main]

jobs:
  quality:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Verify formatting
        run: |
          test -z "$(gofmt -l .)"

      - name: Verify modules
        run: |
          go mod tidy
          git diff --exit-code go.mod go.sum
          go mod verify

      - name: Vet
        run: go vet ./...

      - name: Test
        run: go test ./...

      - name: Race tests
        run: go test -race ./...

      - name: Vulnerability check
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...
```

## Pipeline com quality gate

Adicione:

```yaml
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
```

E configure branch protection para exigir:

```text
quality
security
tests
lint
govulncheck
```

---

# 6. Testes world-class para Go

Para uma skill Strategist, eu faria esta pirâmide:

| Tipo               | Objetivo                                       |
| ------------------ | ---------------------------------------------- |
| unit tests         | regras de domínio, contratos, parser de intake |
| table-driven tests | classificação de intenção, routing, policies   |
| contract tests     | provider/skill adapter obedece schema          |
| integration tests  | leitura de `.sdd`, registry, artifacts         |
| golden tests       | saída Markdown/YAML do plano                   |
| fuzz tests         | parsing de prompt, YAML, manifests             |
| race tests         | runtime state, progress events, cache          |
| e2e mínimo         | fluxo completo: intake → plan → approval gate  |

Fuzzing é especialmente útil se o Strategist processa prompt bruto, YAML, manifests ou logs; a documentação de Go descreve fuzzing como execução de dados aleatórios contra testes para encontrar crashes e vulnerabilidades. ([Go.dev][7])

Exemplo de teste table-driven:

```go
func TestClassifyIntent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want Intent
	}{
		{
			name: "analysis request",
			in:   "analise o projeto e sugira melhorias",
			want: IntentAnalysis,
		},
		{
			name: "execution request",
			in:   "implemente o plano aprovado",
			want: IntentExecution,
		},
		{
			name: "test failure",
			in:   "go test falhou com stacktrace",
			want: IntentFix,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ClassifyIntent(tt.in)
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
```

Seu material anterior já apontava esse padrão: em Go, a base world-class é `testing` + table-driven tests + integração seletiva + fuzzing onde há parser/validator. 

---

# 7. Arquivos que eu criaria no projeto

```text
.sdd/
├── governance/
│   └── guardrails/
│       ├── no-hack-without-evidence.md
│       ├── no-suppression-without-justification.md
│       ├── dependency-direction.md
│       ├── scope-locking.md
│       └── test-integrity.md
│
├── runtime/
│   └── contracts/
│       ├── execution-contract.schema.json
│       ├── mission-contract.schema.json
│       └── validation-result.schema.json
│
├── skills/
│   └── strategist/
│       ├── skill.yaml
│       ├── protocol.md
│       ├── guardrails.yaml
│       └── output.schema.json
│
└── quality/
    ├── golangci.yml
    ├── architecture-rules.yaml
    └── ci-checks.yaml
```

---

# 8. `golangci.yml` inicial

```yaml
run:
  timeout: 5m
  tests: true

linters:
  enable:
    - govet
    - staticcheck
    - errcheck
    - ineffassign
    - unused
    - revive
    - gocritic
    - gosec
    - bodyclose
    - contextcheck
    - cyclop
    - dupl
    - misspell
    - unconvert
    - prealloc
    - whitespace

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0

linters-settings:
  cyclop:
    max-complexity: 12

  revive:
    severity: warning

  errcheck:
    check-type-assertions: true
    check-blank: true

  gosec:
    excludes:
      - G104 # prefer errcheck for unchecked errors
```

Eu evitaria começar com uma config “absurdamente severa”. Melhor:

```text
fase 1: warning
fase 2: blocking para novos problemas
fase 3: blocking global
```

---

# 9. Anti-patterns que eu bloquearia

## Código

```text
- panic como controle de fluxo
- recover silencioso
- erro ignorado
- `context.Background()` em service/usecase
- função com múltiplas responsabilidades
- interface definida pelo consumidor errado
- pacote `utils` genérico
- global mutable state
- goroutine sem cancelamento
- canal sem owner claro
```

## Testes

```text
- enfraquecer assertion para passar
- remover caso de teste sem justificativa
- snapshot/golden atualizado sem diff explicado
- mockando tudo
- teste que não falha quando comportamento quebra
- teste dependente de ordem
- teste com sleep arbitrário
```

## Arquitetura

```text
- domínio importando infraestrutura
- application importando adapter concreto
- provider externo vazando para core
- config/env dentro de domínio
- import para resolver pressa
- abstração nova sem evidência
- pacote grande chamado common/utils/shared
```

## Agente/skill

```text
- executar sem approval gate
- pular diagnose
- alterar fora do scope
- chamar shell cru quando existe skill
- tratar baixa confiança como certeza
- gerar plano sem critérios de aceite
- declarar sucesso sem validação
```

---

# 10. Modelo de maturidade

Eu implementaria em 4 fases:

## Fase 1 — Baseline

```text
go test
go vet
gofmt
go mod tidy
golangci-lint
govulncheck
```

## Fase 2 — Arquitetura

```text
dependency direction
package boundaries
scope locking
no direct infra import
```

## Fase 3 — Skill Governance

```text
execution_contract
approval_gate
progress events
artifact validation
diagnose before rewrite
```

## Fase 4 — Scoring

```text
hack risk score
scope drift score
validation confidence
architecture compliance
test integrity score
```

---

# Veredito

Para uma skill **Strategist em Go**, o padrão world-class seria:

```text
Go quality gates
+ architecture dependency rules
+ security/supply-chain checks
+ test integrity
+ execution contracts
+ SDD runtime governance
```

O ponto mais importante:

> Não confie só em lint.
> Faça o Strategist provar intenção, escopo, evidência, diff e validação antes de aceitar qualquer mudança.
