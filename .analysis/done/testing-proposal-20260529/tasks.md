# Tasks: Implementação do Test Harness
**Mission ID:** testing-proposal-20260529  
**Date:** 2026-05-29  
**Scope:** escreve fora de `.analysis/` — requer approval gate

---

## Fase 1 — Validators (baixo risco, alto impacto)

**1.1** Criar `strategist/tests/validators/validate-contracts.sh`
- Itera `strategist/contracts/*.yaml`
- Verifica campos obrigatórios: `module`, `type`, `description`, `write_scope`, `owner`
- Verifica seções não-vazias: `contract.input`, `contract.output`, `contract.error_conditions`
- Saída: ok/FAIL por arquivo, exit code 0 se tudo ok

**1.2** Criar `strategist/tests/validators/validate-schemas.sh`
- Itera `strategist/schemas/*.yaml`
- Valida que cada arquivo é YAML parseable via `yq eval '.'`
- Saída: ok/FAIL por arquivo

**1.3** Criar `strategist/tests/validators/validate-compiled.sh`
- Recebe: path de arquivo `.gz` + schema esperado
- Valida: JSON válido, campo `schema` correto, campos `compiled_at` e `sources` presentes
- Usado pelos unit tests dos compile scripts

**1.4** Criar `strategist/tests/validators/validate-events.sh`
- Lê stdin (log de sessão do agente)
- Para cada linha que começa com `[Strategist]`, valida formato:
  `[Strategist] phase=<word> status=(running|done|failed|blocked|plan_only)`
- Saída: contagem de válidas/inválidas

---

## Fase 2 — Unit Tests dos Shell Scripts

**2.1** Criar `strategist/tests/unit/test-check-stale.sh`
- Case 1: arquivo ausente → exit 1
- Case 2: artifact presente, sem `.manifest.gz` → exit 1
- Case 3: artifact + manifest, `sources` vazio → exit 0 (fresh)
- Case 4: artifact + manifest, source com mtime adulterado → exit 1 (stale)

**2.2** Criar `strategist/tests/unit/test-compile-config.sh`
- Cria `.strategist/` mínimo em tmpdir (active.yaml + 1 persona + 1 role)
- Executa `compile-config.sh $TMPDIR $TMPDIR/out.gz`
- Valida output via `validate-compiled.sh`: schema correto, active.mode presente, personas key presente

**2.3** Criar `strategist/tests/unit/test-compile-domain.sh`
- Cria `.strategist/` mínimo com `index.yaml` (sem arquivos `load_always` — evita dependência de paths reais)
- Executa `compile-domain.sh $TMPDIR $TMPDIR/out.gz`
- Valida output: schema `strategist-compiled-domain/1.0`, campo `load_always` presente

**2.4** Criar `strategist/tests/unit/test-compile-all.sh`
- Cria estrutura completa mínima em tmpdir
- Executa `compile-all.sh $TMPDIR $TMPDIR/knowledge.index.yaml`
- Valida: `.manifest.gz` existe, os 3 artifacts existem, manifest tem SHA checksums

---

## Fase 3 — Integration Tests

**3.1** Criar `strategist/tests/integration/test-install-silent.sh`
- Executa `bash strategist/install.sh --silent` em tmpdir
- Verifica que `.strategist/` contém:
  - `SKILL.md`
  - `active.yaml`
  - `personas/pragmatic.yaml`
  - `roles/default.yaml`
  - `schemas/` (diretório não-vazio)
  - `contracts/` (diretório não-vazio)
  - `scripts/check-stale.sh` (executável)
  - `scripts/compile-all.sh` (executável)

---

## Fase 4 — Behavior Specs (documentação)

**4.1** Criar `strategist/tests/specs/approval-gate.feature`
- 3 scenarios: Sniper bloqueado, Sniper liberado após "yes", plan_only após "no"

**4.2** Criar `strategist/tests/specs/slot-contracts.feature`
- 3 scenarios: Ranger boundary, Archivist boundary, Sniper risk_score obrigatório

**4.3** Criar `strategist/tests/specs/forbidden-behaviors.feature`
- 3 scenarios: direct_execution, silent_phase_advance, approval_bypass

**4.4** Criar `strategist/tests/specs/drift-correction.feature`
- 3 scenarios: pattern triggered, pattern below threshold, manual clear

---

## Fase 5 — Harness e Makefile

**5.1** Criar `strategist/tests/harness/run-tests.sh`
- Roda: validators → unit tests → integration tests
- Saída colorida: PASS/FAIL por teste
- Exit code 0 se tudo ok, 1 se qualquer falha

**5.2** Criar `strategist/tests/harness/Makefile`
- Targets: `make test`, `make test-validators`, `make test-unit`, `make test-integration`, `make clean`

---

## Ordem Recomendada de Implementação

```
1.1 → 1.2 → 2.1 → 3.1 → 1.3 → 2.2 → 2.3 → 2.4 → 1.4 → 4.1 → 4.2 → 4.3 → 4.4 → 5.1 → 5.2
```

Os validators (1.x) primeiro porque são usados pelos unit tests (2.x). O integration test (3.1) pode ser feito cedo pois não depende dos validators.

---

## Dependências a Verificar Antes da Implementação

```bash
command -v jq   || echo "jq not found"
command -v yq   || echo "yq not found"
command -v gzip || echo "gzip not found"
```

Se `yq` não estiver disponível: `pip install yq` ou `brew install yq` ou `snap install yq`.
