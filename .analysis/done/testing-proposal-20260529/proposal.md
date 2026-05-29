# Proposta: Testes Agnósticos de Linguagem — Strategist Skill
**Mission ID:** testing-proposal-20260529  
**Date:** 2026-05-29  
**Based on:** discovery artifact + testing_strategy.md (base) + criticas_projeto.md (gaps)

---

## O Problema

O Strategist tem invariantes de segurança críticos definidos apenas em prosa (SKILL.md):
- Approval gate nunca pode ser bypassado
- Forbidden behaviors devem ser bloqueados
- Slot write scopes (Ranger=write_pending, Archivist=write_analysis, Sniper=controlled) devem ser respeitados
- Contratos de módulo devem ter inputs válidos antes de cada fase

Atualmente: **zero testes automatizados** para qualquer desses invariantes. Refactors de SKILL.md, contratos, ou scripts de compilação não têm rede de segurança.

---

## A Proposta

Implementar um test harness em **duas camadas**, agnóstico de linguagem de programação, usando exclusivamente: `bash`, `jq`, `yq`, `gzip`, `grep`.

### Camada 1 — Testes Estáticos e de Shell (implementável agora)

Testa o que pode ser verificado sem executar o agente Claude:

**1a. Validação de schemas YAML** — todos os arquivos em `strategist/schemas/` e `strategist/contracts/` têm estrutura correta e campos obrigatórios.

**1b. Validação de contratos de módulo** — cada `.yaml` em `strategist/contracts/` declara: `module`, `type`, `description`, `contract.input[]`, `contract.output[]`, `contract.error_conditions[]`, `write_scope`, `owner`.

**1c. Testes unitários de shell scripts** — `check-stale.sh`, `compile-config.sh`, `compile-domain.sh`, `compile-all.sh` têm comportamento correto verificável em tmpdir.

**1d. Teste de integração do install.sh** — `install.sh --silent` em tmpdir produz a estrutura `.strategist/` completa e esperada.

**1e. Validação de event log format** — linhas `[Strategist] phase=X status=Y` emitidas por sessões têm formato correto (validável com grep/jq).

### Camada 2 — Behavior Specs (documentação viva, executável no futuro)

Gherkin `.feature` files que documentam formalmente os invariantes de segurança. Não requerem runner agora — são a especificação formal. Quando um CLI mock existir, tornam-se testes executáveis.

Specs a criar:
- `approval-gate.feature` — Sniper nunca executa sem approval explícita
- `slot-contracts.feature` — Ranger/Archivist/Sniper respeitam write scopes
- `forbidden-behaviors.feature` — cada drift pattern resulta em bloqueio correto
- `drift-correction.feature` — padrões conhecidos triggeram self-correction

---

## Por Que Esta Abordagem

| Critério | Decisão | Razão |
|---------|---------|-------|
| Sem Python/Go | Sem dependências de linguagem | Projeto é .md + .sh; adicionar linguagem cria barreira de entrada |
| Sem Pydantic | Descartado | Proposta alternativa (testing_strategy2.md) foi descartada |
| BDD como documentação | Sem runner agora | Agente AI não tem CLI invocável; specs são o contrato formal |
| Golden files apenas para output determinístico | Sim para scripts, não para agente | Agent outputs variam; script outputs são determinísticos |
| Nomenclatura atualizada | Ranger/Archivist/Sniper | Nomes antigos (Scout/Engineer/Hunter) foram renomeados em commit 4a14276 |

---

## O que Esta Proposta NÃO É

- Não é uma proposta de CI completo (CI é consequência, não o objetivo)
- Não é uma proposta de testes de agente AI como sistema (isso requer infra separada)
- Não substitui revisão humana das decisões do agente

---

## Escopo e Prioridade

**MVP (alta prioridade, baixo risco):**
1. Validators de contratos e schemas (impacto imediato, zero complexidade)
2. Unit tests de `check-stale.sh` (componente crítico do fast path)
3. Integration test de `install.sh` (regredir aqui seria doloroso)

**Segunda onda:**
4. Unit tests dos compile scripts
5. Gherkin specs de behavior (documentação formal)
6. Harness + Makefile

**Fora de escopo desta proposta:**
- Mock agent / CLI runner
- Testes de integração do agente Claude com SKILL.md
- Performance benchmarks
