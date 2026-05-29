# Discovery: Esforço de Migração dos Testes para Go
**Mission ID:** go-migration-20260529  
**Date:** 2026-05-29  
**task_type:** architecture_analysis  
**execution_intent:** plan_only

---

## Inventário da Suite Atual

| Componente | Arquivo(s) | Linhas | Dependências externas |
|-----------|-----------|-------|----------------------|
| Validators | 4 `.sh` | ~130 | `yq`, `jq`, `grep` |
| Unit tests | 4 `.sh` | ~248 | `bash`, `gzip`, `yq`, `jq` |
| Integration | 1 `.sh` | 64 | `bash`, invoca `install.sh` |
| Harness | 1 `.sh` | 68 | `bash` |
| Fixture runner | 1 `.sh` | 69 | `python3`, `pyyaml` |
| Fixtures YAML | 5 `.yaml` | — | lidos por runner |
| BDD specs | 4 `.feature` | — | documentação |
| **Total executável** | **11 arquivos** | **~579 linhas** | — |

### O que cada componente testa

**Validators (shell → yq/jq):**
- Estrutura de `contracts/*.yaml` (8 campos obrigatórios)
- Parseabilidade de `schemas/*.yaml`
- Estrutura de artifacts `.gz` (schema + campos)
- Formato de event log lines `[Strategist] phase=X status=Y`

**Unit tests (shell → invoca scripts):**
- `check-stale.sh` — 4 casos (absent, sem manifest, fresh, stale)
- `compile-config.sh` — output schema + campos active/personas/roles
- `compile-domain.sh` — output schema + campos
- `compile-all.sh` — pipeline completo, 4 artifacts presentes

**Integration:**
- `install.sh --silent` em tmpdir → 10 paths verificados

**Fixture runner (python3):**
- Carrega 5 YAML fixtures, valida formato de `expected_event`
- Cada fixture declara: scenario, input state, trigger, expected_event

---

## Duas Estratégias de Migração

### Estratégia A — Go test harness, shell scripts intactos

Migra apenas a camada de testes para Go (`testing` package + `testify`).
Os scripts testados (`check-stale.sh`, `compile-*.sh`, `install.sh`) continuam em shell.
Os testes Go invocam esses scripts via `os/exec`.

**O que muda:**
- Validators reescritos como funções Go (YAML: `gopkg.in/yaml.v3`, JSON: `encoding/json`, gzip: stdlib)
- Unit tests como funções `TestXxx` que criam tmpdir e chamam scripts via `exec.Command`
- Integration test como `TestInstallSilent` com `os/exec`
- Fixtures YAML carregados como structs Go
- Harness substituído por `go test ./...`

**O que NÃO muda:**
- `check-stale.sh`, `compile-*.sh`, `install.sh`, `bootstrap.sh` — permanecem shell
- `.feature` files — permanecem como documentação
- `fixtures/*.yaml` — permanecem como input para os testes Go

**Mapeamento shell → Go:**

| Shell | Go equivalente |
|-------|---------------|
| `yq ".$field" file.yaml` | `yaml.Unmarshal` + struct access |
| `jq -r ".schema" <<< $json` | `json.Unmarshal` + struct field |
| `gunzip -c file.gz` | `compress/gzip.NewReader` |
| `grep -qE "pattern"` | `regexp.MustCompile(...).MatchString` |
| `mktemp -d` + `trap EXIT` | `t.TempDir()` (cleanup automático) |
| `exec.Command("sh", "script.sh")` | `exec.Command` + `Output()` |

**Esforço estimado:**
- Setup do módulo Go (`go.mod`, estrutura de pacotes): 1h
- Migrar validators (4): ~4h
- Migrar unit tests (4): ~6h
- Migrar integration test: ~2h
- Migrar fixture runner: ~3h
- **Total: 1,5–2 dias**

**Resultado:** `go test ./strategist/tests/...` substitui `bash harness/run-tests.sh`

---

### Estratégia B — Go completo (scripts + testes)

Reescreve os shell scripts que estão sendo testados em Go também.
`check-stale.sh` → função Go. `compile-*.sh` → funções Go. `install.sh` → CLI Go.

**O que muda:**
- Tudo da Estratégia A
- `check-stale.sh` → `pkg/stale/check.go`
- `compile-config.sh` → `pkg/compile/config.go`
- `compile-domain.sh` → `pkg/compile/domain.go`
- `compile-all.sh` → `pkg/compile/all.go`
- `install.sh` → CLI Go (Cobra) com commands: `install`, `compile`, `check-stale`
- Testes unitários diretos (sem `os/exec`) — muito mais rápidos e robustos

**Benefícios sobre A:**
- Testes sem `os/exec` (testa funções Go diretamente, não processos)
- Single binary: `strategist install`, `strategist compile`
- Windows sem WSL ou PS1
- Tipagem forte end-to-end

**Esforço estimado:**
- Estratégia A (base): 2 dias
- Reescrever `check-stale.sh` → Go: 4h
- Reescrever 3 compile scripts → Go: 1,5 dias
- Reescrever `install.sh` como CLI Cobra: 3–4 dias
- Ajustar `bootstrap.sh` para baixar binário Go: 4h
- **Total: 2–3 semanas**

---

## Análise de Trade-offs

| Critério | Atual (shell) | Estratégia A | Estratégia B |
|---------|--------------|-------------|-------------|
| Esforço de migração | — | 1,5–2 dias | 2–3 semanas |
| Dependências de dev | jq, yq, python3 | Go 1.21+ | Go 1.21+ |
| `go test ./...` | ✗ | ✓ | ✓ |
| Sem `os/exec` nos testes | ✓ (shell puro) | ✗ (ainda invoca .sh) | ✓ |
| Windows sem PS1/WSL | ✗ | ✗ | ✓ |
| Single binary distribuível | ✗ | ✗ | ✓ |
| Retém simplicidade shell | ✓ | Parcial | ✗ |
| Risco de quebrar installs | Baixo | Baixo | Alto |

---

## Motivações que justificam cada estratégia

**Estratégia A faz sentido se:**
- Já há Go na toolchain de dev
- Quer `go test` como interface padrão de CI
- Quer melhor output de erros e subtests
- Não quer reescrever os shell scripts

**Estratégia B faz sentido se:**
- Há um plano para distribuir `strategist` como CLI binária (mencionado em testing_strategy2.md)
- Suporte Windows sem WSL é requisito
- A equipe já escreve Go e shell é uma barreira de entrada

**Nenhuma migração faz sentido se:**
- O projeto continua `.md + .sh`
- Os testes atuais funcionam e estão verdes no CI
- Não há driver específico (velocidade, Windows, tipagem)

---

## Conclusão para o Refinamento

A pergunta-chave não é "como migrar" mas "por que migrar". O esforço é proporcional ao escopo:
- **A:** viável num sprint, zero risco para os scripts em produção
- **B:** projeto de 2–3 semanas, transforma o repositório num projeto Go — decisão arquitetural significativa

Recomendação: se a motivação for CI/DX, Estratégia A. Se a motivação for CLI distribuível, B. Se não houver driver claro, manter shell.
