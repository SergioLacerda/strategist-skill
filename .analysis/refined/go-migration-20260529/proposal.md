# Proposta: Migração dos Testes para Go
**Mission ID:** go-migration-20260529  
**Date:** 2026-05-29  
**Status:** plan_only — análise de esforço sem execução

---

## Resposta Direta

Há duas estratégias com esforços muito distintos. A escolha depende inteiramente da motivação.

---

## Estratégia A — Go test harness, scripts shell intactos

**Esforço: 1,5–2 dias**

Substitui os runners shell (`harness/run-tests.sh`, `run-tests.sh`) e validators por código Go. Os scripts que estão sendo testados (`check-stale.sh`, `compile-*.sh`, `install.sh`) continuam em shell — invocados via `os/exec`.

**Resultado:** `go test ./strategist/tests/...` no lugar de `bash harness/run-tests.sh`

### O que fica igual

- `check-stale.sh`, `compile-config.sh`, `compile-domain.sh`, `compile-all.sh`
- `install.sh`, `bootstrap.sh`, `bootstrap.ps1`
- Fixtures YAML (`fixtures/*.yaml`) — lidos como structs Go
- BDD specs (`.feature`) — documentação, sem mudança

### O que muda

| Atual (shell) | Go equivalente |
|--------------|---------------|
| `validate-contracts.sh` (yq) | `gopkg.in/yaml.v3` + struct fields |
| `validate-schemas.sh` (yq) | `yaml.Unmarshal` |
| `validate-compiled.sh` (jq + gzip) | `compress/gzip` + `encoding/json` |
| `validate-events.sh` (grep) | `regexp.MustCompile` |
| `test-check-stale.sh` | `exec.Command("sh", "check-stale.sh")` + `t.TempDir()` |
| `test-compile-*.sh` | `exec.Command` + fixture tmpdir |
| `test-install-silent.sh` | `exec.Command("bash", "install.sh", "--silent")` |
| `run-tests.sh` (python3) | `TestFixtures` com `yaml.Unmarshal` |
| `harness/run-tests.sh` | `go test -v ./...` |
| `Makefile` | Opcional — Go tem `go test` nativo |

### Dependências novas

```
go.mod:
  gopkg.in/yaml.v3        # YAML parsing
  github.com/stretchr/testify  # assertions (opcional mas recomendado)
```

### Estrutura proposta

```
strategist/tests/
├── go.mod
├── fixtures_test.go       # carrega e valida *.yaml fixtures
├── validators/
│   ├── contracts_test.go  # itera contracts/*.yaml
│   ├── schemas_test.go
│   ├── compiled_test.go
│   └── events_test.go
├── unit/
│   ├── check_stale_test.go
│   ├── compile_config_test.go
│   ├── compile_domain_test.go
│   └── compile_all_test.go
└── integration/
    └── install_test.go
```

---

## Estratégia B — Go completo (scripts + testes)

**Esforço: 2–3 semanas**

Reescreve os shell scripts em Go também. O repositório passa a ter um pacote Go que é distribuído como binário.

### Escopo adicional sobre A

| Script atual | Go |
|-------------|-----|
| `check-stale.sh` | `pkg/stale/check.go` — testa freshness sem subprocess |
| `compile-config.sh` | `pkg/compile/config.go` |
| `compile-domain.sh` | `pkg/compile/domain.go` |
| `compile-all.sh` | `pkg/compile/all.go` |
| `install.sh` | `cmd/strategist/install.go` (Cobra command) |

Testes ficam mais rápidos e robustos — sem `os/exec`, testam funções Go diretamente.

### Benefício exclusivo de B

- **Windows sem WSL** — binário Go nativo dispensa `bootstrap.ps1`
- **Single binary**: `strategist install`, `strategist compile`, `strategist check-stale`
- **Tipagem end-to-end** — erros de schema pegos em compile time, não runtime

### Risco

- Engenharia significativa (reescrever `install.sh` = 300+ linhas de shell com muita lógica de wizard)
- Quebra a filosofia atual "zero dependência além de bash" para quem contribui
- Requer CI para compilar o binário por plataforma

---

## Matriz de Decisão

| Driver | Estratégia recomendada |
|--------|----------------------|
| Quero `go test ./...` no CI | A |
| Já tenho Go na toolchain | A |
| Quero melhor DX de erros/subtests | A |
| Não quero reescrever os scripts | A |
| Quero CLI binária distribuível | B |
| Preciso suportar Windows sem PS1/WSL | B |
| A equipe prefere Go a shell | B |
| Não há driver específico | Manter shell atual |

---

## Recomendação

**Se o objetivo é modernizar os testes → Estratégia A (1,5–2 dias).**  
É cirúrgica: substitui apenas a camada de teste, sem tocar nos scripts em produção.

**Se o objetivo é um CLI distribuível → Estratégia B (2–3 semanas).**  
Vale o investimento somente se já há plano concreto de releases por plataforma.

**Se não há driver claro → não migrar.**  
O custo de adicionar Go como dependência de dev é real; a suite shell atual funciona e é suficiente.
