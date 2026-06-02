# Strategist Performance Optimization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduzir a latência percebida da missão Strategist, priorizando ciclo do agente, I/O de contexto e volume emitido, sem degradar compliance.

**Architecture:** Executar em duas ondas. Primeiro, cortar custo cognitivo e volume de contexto com `--speed`, carregamento mínimo e emits compactos. Depois, atacar custo de runtime com cache e paralelização em Go, mantendo os contratos e os testes de alinhamento como guardrails.

**Tech Stack:** Go (`cmd/strategist`, `internal/*`), contratos/personas em YAML (`.strategist`, `internal/embed/defaults`), testes de contrato/alinhamento (`strategist/tests`).

---

### Task 1: Baseline de performance e telemetria

**Files:**
- Create: `.analysis/refined/2026-06-01-performance-baseline-metrics.md`
- Modify: `docs/skill-internals.md` (seção de métricas)

**Step 1: Definir métricas canônicas**
- `t_start_to_intake_ms`
- `t_intake_to_ranger_ms`
- `total_wall_time_ms`
- `tokens_in`, `tokens_out`
- `lines_emitted`

**Step 2: Registrar baseline atual**
- Rodar uma missão representativa do Strategist e anotar os números atuais antes de qualquer mudança.

**Step 3: Formalizar alvo de redução**
- Meta inicial: `-30% start_to_feedback`, `-25% tokens_in`, `-20% wall_time`.

### Task 2: Modo de velocidade (`--speed=fast|balanced|full`)

**Files:**
- Modify: `.strategist/skill.yaml`
- Modify: `.strategist/SKILL.md`
- Modify: `.strategist/output-profiles/emit-taxonomy.yaml`
- Modify: `internal/embed/defaults/skill.yaml`
- Modify: `internal/embed/defaults/SKILL.md`
- Modify: `internal/embed/defaults/output-profiles/emit-taxonomy.yaml`
- Test: `strategist/tests/spec_alignment_test.go`

**Step 1: Escrever testes de contrato para `speed`**
- Validar presença de `speed` em contratos/config.

**Step 2: Implementar defaults e política**
- `balanced` default.
- `fast`: reduz narrativa e desabilita não-bloqueantes.
- `full`: mantém detalhamento completo.

**Step 3: Ajustar visibilidade por speed**
- manter compliance mínima sempre visível.

**Step 4: Revalidar testes de alinhamento**
Run: `go test ./strategist/tests -v`
Expected: PASS.

### Task 3: Contexto mínimo por missão (task_type-driven)

**Files:**
- Modify: `.strategist/contracts/context-enrichment.yaml`
- Modify: `.strategist/contracts/bootstrap.yaml`
- Modify: `.strategist/contracts/tests/context-enrichment.test.yaml`
- Modify: `.strategist/contracts/tests/bootstrap.test.yaml`
- Sync: `internal/embed/defaults/contracts/*`

**Step 1: Definir política de carga mínima**
- Carregar apenas persona ativa, roles necessários e contratos da missão.

**Step 2: Adicionar `top_k` e limite por fonte**
- Introduzir corte de fontes no enrichment.

**Step 3: Ajustar testes de contrato**
- Cobrir short-circuit e empty non-blocking.

**Step 4: Validar integridade dos contratos**
Run: `go test ./strategist/tests -v`
Expected: PASS.

### Task 4: Compactação de diagnóstico/compliance no chat

**Files:**
- Modify: `.strategist/personas/pragmatic.yaml`
- Modify: `.strategist/personas/epic.yaml`
- Modify: `.strategist/contracts/tests/compliance-summary.test.yaml`
- Sync: `internal/embed/defaults/personas/*`

**Step 1: Padronizar bloco compacto de diagnóstico**
- Uma linha por bloco obrigatório.

**Step 2: Preservar semântica em ambas personas**
- `epic` mantém estilo; `pragmatic` mantém técnica.

**Step 3: Validar redução de linhas emitidas**
- Medir antes/depois no cenário baseline.

### Task 5: Cache em memória para artefatos compilados (P1)

**Files:**
- Create: `internal/compile/cache.go`
- Modify: `internal/compile/all.go`
- Modify: `internal/compile/*_test.go`

**Step 1: Escrever testes de cache hit/miss/invalidate**
- chave por `path+mtime+size`.

**Step 2: Implementar cache LRU simples**
- limite configurável por entries.

**Step 3: Integrar leitura de artefatos com cache**
- fallback seguro quando cache indisponível.

**Step 4: Validar benchmarks**
Run: `go test -bench . ./internal/compile`
Expected: melhoria nas rotas quentes de parse/descompressão.

### Task 6: Paralelização de compile (P1)

**Files:**
- Modify: `internal/compile/all.go`
- Modify: `internal/compile/compile_test.go`

**Step 1: Escrever teste para atomicidade de outputs**
- sem `.manifest.gz` em falha parcial.

**Step 2: Implementar `errgroup` para config/domain/index**
- escrita atômica de arquivos temporários + rename.

**Step 3: Validar regressão funcional**
Run: `go test ./internal/compile ./internal/stale`
Expected: PASS.

### Task 7: Hardening de execução e governança

**Files:**
- Modify: `.analysis/refined/2026-06-01-strategist-performance-optimization.md` (evidências)
- Optional: `docs/cli-reference.md` (documentar `--speed`)

**Step 1: Regressão completa**
Run: `go test ./internal/install ./internal/compile ./internal/stale ./internal/embed ./strategist/tests ./cmd/strategist`
Expected: PASS.

**Step 2: Validar governança**
Run: `uv run sdd governance validate`
Expected: PASS em todos checks.

**Step 3: Registrar before/after de métricas**
- atualizar baseline com deltas reais.

---

## Sequencing Notes

- Execute Tasks 1-4 primeiro; elas atacam o maior custo percebido com menor risco.
- Execute Tasks 5-6 só depois que os contratos estiverem estáveis e medidos.
- Task 7 fecha a entrega com evidência e validação cruzada.

---

## Definition of Done

- `--speed` implementado e documentado (`fast|balanced|full`).
- Contexto carregado de forma mínima por missão.
- Diagnóstico/compliance compactos sem perda semântica.
- Cache + paralelização implementados sem regressão.
- Ganhos medidos contra baseline e governança verde.

## Execution Notes

- Não executar `git commit` durante esta implementação (preferência do usuário).
- Nenhum teste/código produtivo deve depender da pasta runtime `./.strategist`; usar `internal/embed/defaults` e `strategist/` como fonte canônica de repo.
- A análise de origem está em `.analysis/refined/2026-06-01-strategist-performance-optimization.md`; este plano deve permanecer alinhado a ela.
