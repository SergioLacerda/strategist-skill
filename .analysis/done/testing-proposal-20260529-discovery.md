# Discovery: Proposta de Testes — Strategist Skill
**Mission ID:** testing-proposal-20260529  
**Date:** 2026-05-29  
**Phase:** analysis (Ranger)  
**Input artifacts:** `.analysis/todo/testing_strategy.md`, `.analysis/todo/testing_strategy2.md`, `.analysis/todo/test_architecture.svg`

---

## 1. Síntese das Propostas Existentes

### testing_strategy.md — Proposta Principal (agnóstica)

Propõe cinco pilares:
1. **Fixtures YAML** — estados de missão simulados (`mission-inputs/`, `internal-state/`, `expected-outputs/`)
2. **Gherkin/BDD** — specs de comportamento (`.feature` files)
3. **Validadores jq** — assertions agnósticas de estrutura JSON
4. **Golden files** — comparação de output esperado vs. real
5. **Shell scripts** — harness POSIX-compatible

Pontos positivos: arquitetura clara, não depende de linguagem, CI multi-OS, coverage dos invariantes críticos (approval gate, drift self-correction, slot write scope).

### testing_strategy2.md — Proposta Alternativa (conversa)

Começou com Python/Pydantic (acoplado), pivotou para abordagem agnóstica. Recomendação final:
- **Fase 1:** JSON Schema + shell scripts (`check-jsonschema`, `yq`, `jq`)
- **Fase 2:** CLI Go + Cobra (futuro)

Mais conservadora. Não propõe BDD. Mais focada em validação de schemas do que em testes de comportamento.

### test_architecture.svg — Diagrama de Arquitetura

Confirma e visualiza a arquitetura de `testing_strategy.md`:
- Camada 1: Fixtures (YAML)
- Camada 2: BDD Specs (Gherkin)
- Camada 3: Validadores (jq + shell)
- Camada 4: Test Harness (Unit → Integration → Contract → E2E → CLI)

Nomes consistentes com a arquitetura da estratégia principal.

---

## 2. Gaps e Problemas Identificados

### 2a. Problema Fundamental: CLI Inexistente

**Ambas as propostas assumem um entrypoint `strategist run --fixture ...` que não existe.**

O Strategist é um agente Claude (AI) que lê SKILL.md e age. Não há um processo shell que possa ser invocado como `strategist run`. Isso invalida os integration tests concretos de ambas as propostas na sua forma atual.

**O que pode ser testado hoje:**
- Schemas YAML (estrutura estática)
- Shell scripts (`check-stale.sh`, `compile-*.sh`, `install.sh`)
- Contratos YAML (estrutura + campos obrigatórios)
- Formato de eventos `[Strategist] phase=X status=Y` emitidos no output do agente

**O que requer CLI futura ou mocking:**
- Mission complete flow
- Approval gate bypass detection
- Drift self-correction triggering

### 2b. Nomenclatura Desatualizada

Ambas as propostas usam os nomes antigos dos papéis:
- **Scout** → agora **Ranger**
- **Engineer** → agora **Archivist**
- **Hunter** → agora **Sniper**

Fixtures e specs que referenciam os nomes antigos falharão silenciosamente se validarem contra SKILL.md ou known-providers.yaml.

### 2c. Proposta Python Contradiz o Objetivo

`testing_strategy2.md` começa com Pydantic/Python — contradiz diretamente o objetivo de "agnóstico de linguagem". O próprio documento chegou a essa conclusão no final. A proposta Python deve ser descartada.

### 2d. BDD sem Step Definitions

A proposta de Gherkin é valiosa como documentação viva, mas `.feature` files sem step definitions não são executáveis. Para um agente AI, as step definitions precisariam parsear outputs de texto — frágil. A proposta não endereça isso.

**Solução alternativa:** usar `.feature` files apenas como especificação de comportamento (documentação), e implementar os testes reais como shell scripts que verificam event logs.

### 2e. Golden Files são Frágeis para Agentes AI

Golden files funcionam bem para outputs determinísticos. Outputs de agentes AI variam por contexto, temperatura e versão de modelo. A proposta de golden files precisa ser limitada aos componentes que têm output determinístico:
- Eventos de progresso (`[Strategist] phase=X status=Y`) — determinísticos
- Estrutura de arquivos gerados por `install.sh` — determinística
- Output de `compile-*.sh` — determinístico (dado mesmo input)

### 2f. Cobertura dos Invariantes de Segurança

`criticas_projeto.md` identifica explicitamente que os invariantes críticos não têm testes:
- Approval gate (nunca bypassável)
- Forbidden behaviors
- Slot write scope contracts

Ambas as propostas identificam isso corretamente mas não propõem como testar sem um CLI. A solução é testar via log analysis de sessões reais + validação estática dos contratos.

---

## 3. O Que Está Pronto para Implementação Imediata

### Testável hoje (shell + jq + yq):

| Componente | Tipo de teste | Ferramentas |
|-----------|--------------|-------------|
| `strategist/schemas/*.yaml` | Schema validation via `yq` | yq |
| `strategist/contracts/*.yaml` | Contract structure validation | yq + jq |
| `strategist/scripts/check-stale.sh` | Shell unit test (mocked gz files) | bash |
| `strategist/scripts/compile-config.sh` | Output structure validation | bash + jq |
| `strategist/scripts/compile-domain.sh` | Output structure validation | bash + jq |
| `strategist/scripts/compile-all.sh` | Integration: produces manifest | bash + jq |
| `strategist/install.sh` | Integration: full install in tmpdir | bash |
| `strategist/SKILL.md` | Static: required sections present | grep |
| `.strategist/known-providers.yaml` | Validates all providers have valid risk scores | yq |
| Event log format | Pattern match on `[Strategist] phase=X status=Y` | grep + jq |

### Testável com behavior specs (não executáveis, mas documentação):

- Gherkin features para: approval gate, forbidden behaviors, slot contracts, drift correction
- Essas specs servem como contrato formal de comportamento — executáveis quando CLI existir

---

## 4. Recomendação de Arquitetura em Duas Camadas

### Camada 1 — Estática + Shell (implementável agora, zero dependência de linguagem)

```
strategist/tests/
├── fixtures/
│   ├── configs/           ← YAMLs de config de teste (active.yaml, roles/)
│   └── compiled/          ← Artifacts gz pré-gerados para testar check-stale.sh
├── specs/
│   ├── approval-gate.feature
│   ├── slot-contracts.feature
│   ├── drift-correction.feature
│   └── forbidden-behaviors.feature
├── validators/
│   ├── validate-contracts.sh    ← valida estrutura de todos os contracts/*.yaml
│   ├── validate-schemas.sh      ← valida YAMLs contra jsonschema
│   ├── validate-compiled.sh     ← valida estrutura dos .gz gerados
│   └── validate-events.sh       ← valida formato de event log lines
├── unit/
│   ├── test-check-stale.sh
│   ├── test-compile-config.sh
│   ├── test-compile-domain.sh
│   └── test-compile-all.sh
├── integration/
│   ├── test-install.sh          ← full install em tmpdir
│   └── test-install-wizard.sh   ← wizard mode em tmpdir
└── harness/
    ├── run-tests.sh
    └── Makefile
```

### Camada 2 — Behavior specs (documentação executável futura)

Gherkin `.feature` files como especificação dos invariantes. Não requerem runner agora — são documentação formal. Quando um CLI ou mock agent existir, tornam-se executáveis.

---

## 5. Decisões para a Fase de Refinamento

1. **Descartar a proposta Python** (testing_strategy2.md) — fora de escopo para um projeto shell+md
2. **Adotar a arquitetura de testing_strategy.md** como base, com as correções acima
3. **Separar claramente Camada 1 (implementável agora) de Camada 2 (futuro)**
4. **Atualizar nomenclatura**: Scout→Ranger, Engineer→Archivist, Hunter→Sniper em todos os fixtures e specs
5. **Prioridade de implementação:** validators de schema e contracts primeiro (maior impacto, zero risco)
6. **BDD specs como documentação** — não como testes executáveis (ausência de runner)
7. **Não criar dependência de linguagem** — dependências permitidas: `bash`, `jq`, `yq`, `gzip`, `grep`, `sed`
