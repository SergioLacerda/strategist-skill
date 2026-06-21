# Critical Hit Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Introduzir a rota "Critical Hit" — uma via rápida (bootstrap → preflight → intake → gate direto → sniper) para tarefas de baixa complexidade como edições de doc, corrigindo o problema de missões simples levando 10+ minutos no pipeline completo.

**Architecture:** O Intake classifica automaticamente a tarefa usando uma `routing_matrix`; tarefas `doc_edit`/`content_update` de baixo risco com ≤ 5 arquivos disparam a rota `direct_execute` (Critical Hit). Dois novos estados FSM (`DIRECT_GATE`, `DIRECT_EXEC`) cobrem o ciclo de vida. O Dojo ganha suporte a `timing_criteria` para detectar regressões de latência. Nenhum artefato intermediário é escrito na rota Critical Hit — o gate é inline, o Sniper executa diretamente.

**Tech Stack:** Go 1.26, testify, gopkg.in/yaml.v3, arquivos YAML de contrato (sem runtime adicional).

---

## Visão geral dos arquivos impactados

| Ação | Arquivo |
|------|---------|
| Criar | `internal/embed/defaults/contracts/critical-hit.yaml` |
| Criar | `internal/embed/defaults/contracts/11-critical-hit.md` |
| Criar | `.analysis/dojo/critical-hit/input.yaml` |
| Criar | `.analysis/dojo/critical-hit/criteria.yaml` |
| Modificar | `internal/domain/policy.go` — novos estados, eventos, constante de rota |
| Modificar | `internal/domain/pipeline_bypass.go` — exceção para `direct_execute` |
| Modificar | `internal/domain/state_machine.go` — novos estados e transições |
| Modificar | `internal/domain/types.go` — `DojoTimingCriteria`, campo em `DojoCriteria` |
| Modificar | `internal/dojo/checker.go` — `CheckTiming`, chamada em `Run` |
| Modificar | `internal/embed/defaults/contracts/00-routing.md` — 3ª rota |
| Modificar | `internal/embed/defaults/contracts/intake.yaml` — `routing_matrix` |
| Modificar | `internal/embed/defaults/contracts/02-intake.md` — mencionar routing_matrix |

---

## Task 1: Constantes de domínio — rota e eventos Critical Hit

**Files:**
- Modify: `internal/domain/policy.go:22-82`
- Test: `internal/domain/pipeline_bypass_test.go`

### Step 1: Adicionar constante de rota e novos estados FSM

No bloco de `MissionState` constants em `policy.go`, após `StateRetrying`, adicionar:

```go
// Critical Hit route states — fast path for low-risk doc/content tasks.
StateDirectGate MissionState = "DIRECT_GATE"
StateDirectExec MissionState = "DIRECT_EXEC"
StateDirectDone MissionState = "DIRECT_DONE"
```

No bloco de `MissionRoute*` constants em `pipeline_bypass.go` (linha 6-10), adicionar:

```go
MissionRouteDirectExecute = "direct_execute"
```

No bloco de `TransitionEvent` constants em `policy.go`, após `EventSniperOA`, adicionar:

```go
// Critical Hit route events.
EventDirectHitIntent    TransitionEvent = "direct_hit_intent"
EventDirectGateApproved TransitionEvent = "direct_gate_approved"
EventDirectGateDeclined TransitionEvent = "direct_gate_declined"
```

### Step 2: Rodar testes existentes para confirmar que nada quebrou

```bash
go test ./internal/domain/... -v -count=1
```

Expected: todos PASS (nenhum test usa os novos símbolos ainda).

### Step 3: Commit

```bash
git add internal/domain/policy.go internal/domain/pipeline_bypass.go
git commit -m "feat(domain): add Critical Hit route constants, FSM states and events"
```

---

## Task 2: Exceção de bypass para rota `direct_execute`

**Files:**
- Modify: `internal/domain/pipeline_bypass.go:47-81`
- Modify: `internal/domain/pipeline_bypass.go` — struct `PipelineEvidence`
- Test: `internal/domain/pipeline_bypass_test.go`

### Step 1: Escrever testes que falham

Adicionar em `pipeline_bypass_test.go`:

```go
func TestEvaluatePipelineBypass_DirectExecuteBlockedWithoutGate(t *testing.T) {
    t.Parallel()
    decision := domain.EvaluatePipelineBypass(domain.PipelineEvidence{
        Route:           domain.MissionRouteDirectExecute,
        BasePath:        ".analysis",
        MissionID:       "m-readme",
        AttemptedAction: "edit readme.md directly",
    })
    assert.False(t, decision.Allowed)
    assert.Equal(t, "direct_gate", decision.ExpectedPhase)
    assert.Contains(t, decision.MissingEvidence, "direct_gate:approved")
}

func TestEvaluatePipelineBypass_DirectExecuteAllowedAfterGate(t *testing.T) {
    t.Parallel()
    decision := domain.EvaluatePipelineBypass(domain.PipelineEvidence{
        Route:             domain.MissionRouteDirectExecute,
        BasePath:          ".analysis",
        MissionID:         "m-readme",
        AttemptedAction:   "edit readme.md directly",
        DirectGateApproved: true,
    })
    assert.True(t, decision.Allowed)
    assert.Empty(t, decision.Reason)
}
```

### Step 2: Rodar para confirmar que falham

```bash
go test ./internal/domain/... -run TestEvaluatePipelineBypass_Direct -v
```

Expected: FAIL — `DirectGateApproved` não existe em `PipelineEvidence` ainda.

### Step 3: Adicionar campo `DirectGateApproved` em `PipelineEvidence`

Em `pipeline_bypass.go`, adicionar campo à struct `PipelineEvidence`:

```go
DirectGateApproved bool
```

### Step 4: Adicionar handler no `EvaluatePipelineBypass`

Logo após o check de `MissionRouteQuickDraw` em `EvaluatePipelineBypass` (linha ~50), adicionar:

```go
if e.Route == MissionRouteDirectExecute {
    return evaluateDirectExecuteBypass(e)
}
```

Adicionar a função:

```go
func evaluateDirectExecuteBypass(e PipelineEvidence) PipelineBypassDecision {
    if !e.DirectGateApproved {
        return blockedBypassDecision(
            e,
            "direct_gate",
            fmt.Sprintf("direct_gate:approved (mission %s)", e.MissionID),
            fmt.Sprintf("present the Critical Hit gate for mission %s and wait for user confirmation before writing", e.MissionID),
        )
    }
    return PipelineBypassDecision{Allowed: true}
}
```

### Step 5: Rodar testes

```bash
go test ./internal/domain/... -v -count=1
```

Expected: todos PASS incluindo os dois novos.

### Step 6: Commit

```bash
git add internal/domain/pipeline_bypass.go internal/domain/pipeline_bypass_test.go
git commit -m "feat(domain): add Critical Hit bypass exception in pipeline_bypass"
```

---

## Task 3: FSM — estados e transições da rota Critical Hit

**Files:**
- Modify: `internal/domain/state_machine.go`
- Test: `internal/domain/state_machine_test.go`

### Step 1: Escrever teste que falha

Adicionar em `state_machine_test.go`:

```go
func TestFSMCriticalHitRoute(t *testing.T) {
    t.Parallel()
    policy := domain.NewMissionPolicy(domain.ExecutionModeApplyWorkspace, domain.GitPersistenceModeExplicitCommit)

    // Intent detectado → DirectGate
    s := domain.NextState(domain.StateInit, domain.EventDirectHitIntent, policy)
    assert.Equal(t, domain.StateDirectGate, s)

    // Usuário aprova → DirectExec
    s = domain.NextState(s, domain.EventDirectGateApproved, policy)
    assert.Equal(t, domain.StateDirectExec, s)

    // Sniper termina → Done
    s = domain.NextState(s, domain.EventSniperDone, policy)
    assert.Equal(t, domain.StateDirectDone, s)

    // Done é absorvente
    s = domain.NextState(s, domain.EventManifestNonEmpty, policy)
    assert.Equal(t, domain.StateDirectDone, s)
}

func TestFSMCriticalHitDeclinedGoesToDoneAnalysis(t *testing.T) {
    t.Parallel()
    policy := domain.NewMissionPolicy(domain.ExecutionModeApplyWorkspace, domain.GitPersistenceModeExplicitCommit)

    s := domain.NextState(domain.StateInit, domain.EventDirectHitIntent, policy)
    assert.Equal(t, domain.StateDirectGate, s)

    s = domain.NextState(s, domain.EventDirectGateDeclined, policy)
    assert.Equal(t, domain.StateDoneAnalysis, s)
}

func TestFSMCriticalHitNeverRunsInPlanOnly(t *testing.T) {
    t.Parallel()
    policy := domain.NewMissionPolicy(domain.ExecutionModePlanOnly, domain.GitPersistenceModeForbidden)

    s := domain.NextState(domain.StateInit, domain.EventDirectHitIntent, policy)
    assert.Equal(t, domain.StateDirectGate, s)

    // plan_only: gate aprovado mas sem execução → DoneAnalysis
    s = domain.NextState(s, domain.EventDirectGateApproved, policy)
    assert.Equal(t, domain.StateDoneAnalysis, s)
}
```

### Step 2: Rodar para confirmar que falham

```bash
go test ./internal/domain/... -run TestFSMCriticalHit -v
```

Expected: FAIL.

### Step 3: Adicionar handlers em `state_machine.go`

Em `NextState`, adicionar casos ao switch:

```go
case StateDirectGate:
    return nextFromDirectGate(event, p)
case StateDirectExec:
    return nextFromDirectExec(event)
case StateDirectDone:
    return current
```

No `nextFromInit`, adicionar:

```go
case EventDirectHitIntent:
    return StateDirectGate
```

Adicionar as duas novas funções:

```go
func nextFromDirectGate(event TransitionEvent, p MissionPolicy) MissionState {
    switch event {
    case EventDirectGateApproved:
        if p.CanExecute {
            return StateDirectExec
        }
        return StateDoneAnalysis
    case EventDirectGateDeclined:
        return StateDoneAnalysis
    }
    return StateDirectGate
}

func nextFromDirectExec(event TransitionEvent) MissionState {
    switch event {
    case EventSniperDone:
        return StateDirectDone
    case EventSlotTransient:
        return StateRetrying
    case EventSlotPermanent:
        return StateBlocked
    }
    return StateDirectExec
}
```

### Step 4: Rodar todos os testes de domínio

```bash
go test ./internal/domain/... -v -count=1
```

Expected: todos PASS.

### Step 5: Commit

```bash
git add internal/domain/state_machine.go internal/domain/state_machine_test.go
git commit -m "feat(domain): add Critical Hit FSM states DIRECT_GATE, DIRECT_EXEC, DIRECT_DONE"
```

---

## Task 4: Contrato YAML `critical-hit.yaml`

**Files:**
- Create: `internal/embed/defaults/contracts/critical-hit.yaml`
- Create: `internal/embed/defaults/contracts/11-critical-hit.md`

### Step 1: Criar o contrato YAML

Criar `internal/embed/defaults/contracts/critical-hit.yaml`:

```yaml
module: critical-hit
type: routing_decision
version: "0.1"

description: >
  Ponto de decisão entre fluxo completo (main_mission) e fluxo direto (direct_execute).
  Executado após intake, antes de qualquer slot provider.
  Quando condições de Critical Hit são satisfeitas, bypassa Ranger e Archivist,
  substituindo o approval_gate padrão por um gate inline de confirmação direta.

trigger_conditions:
  all_of:
    - task_type: [doc_edit, content_update, readme_update, comment_update]
    - risk_level: low
    - file_count: "<= 5"
    - no_cross_module_deps: true
    - no_adr_required: true
    - no_architectural_decision: true
  none_of:
    - keywords: [investigar, propor, auditar, projetar, redesenhar, arquitetura, refatorar]
    - task_type: [architecture_analysis, refactor, security_audit, new_feature]

route_assigned: direct_execute

pipeline:
  omitted_phases: [discovery, refinement, opportunity_attack]
  artifacts_written: none
  gate_style: inline

inline_gate:
  display: |
    Critical Hit detectado.
    Tarefa: <task_description>
    Arquivos: <file_list>
    Confirma execução direta? (sim/nao)
  stop: true
  wait_for_response: true
  responses:
    sim: proceed to sniper
    yes: proceed to sniper
    nao: resolve as plan_only (no artifacts written)
    no:  resolve as plan_only (no artifacts written)

emit:
  on_trigger:
    format: "[Strategist] phase=critical_hit status=triggered task_type={task_type} file_count={file_count}"
    level: INFO
  on_gate_approved:
    format: "[Strategist] phase=critical_hit status=gate_approved"
    level: TRACE
  on_gate_declined:
    format: "[Strategist] phase=critical_hit status=gate_declined"
    level: TRACE

write_scope: read-only (gate phase only; Sniper write scope applies after approval)
owner: internal (orchestrator)

invariants:
  - Critical Hit NEVER fires when risk_level != low
  - Critical Hit NEVER fires for tasks with cross-module dependencies
  - Critical Hit NEVER fires when any none_of keyword is present in the prompt
  - If in doubt, route to main_mission — conservatism is the safe default
  - Approval gate cannot be bypassed even in Critical Hit route
```

### Step 2: Criar o contrato narrativo

Criar `internal/embed/defaults/contracts/11-critical-hit.md`:

```markdown
# Strategist — Contract 11: Critical Hit

## Purpose

Decision point between **fluxo completo** (main_mission) and **fluxo direto** (direct_execute).
Evaluated after Intake, before any slot provider is invoked.

## When to Apply

Critical Hit fires when **all** conditions are true:

- `task_type` is one of: `doc_edit`, `content_update`, `readme_update`, `comment_update`
- `risk_level` is `low`
- Number of affected files ≤ 5
- No cross-module dependencies
- No ADR required
- No architectural decision involved

And **none** of these are true:

- Prompt contains keywords: `investigar`, `propor`, `auditar`, `projetar`, `redesenhar`, `arquitetura`, `refatorar`
- `task_type` is: `architecture_analysis`, `refactor`, `security_audit`, `new_feature`

**When in doubt → main_mission. Conservatism is the safe default.**

## Pipeline Difference

| Phase | main_mission | direct_execute (Critical Hit) |
|-------|-------------|-------------------------------|
| Ranger discovery | ✅ | ❌ skipped |
| Archivist refinement | ✅ | ❌ skipped |
| Opportunity attack | ✅ | ❌ skipped |
| Approval gate | Full gate (reads tasks.md) | Inline gate (single message) |
| Sniper execution | ✅ | ✅ |
| Artifacts written | analysis.md, proposal.md, design.md, tasks.md | none |

## Inline Gate

```
Critical Hit detectado.
Tarefa: <descrição da tarefa>
Arquivos: <lista de arquivos>
Confirma execução direta? (sim/nao)
```

- `sim/yes` → proceed to Sniper
- `nao/no` → resolve as `plan_only` (nothing written)

## Emit Events

- `critical_hit_triggered` — when conditions match and Critical Hit route is selected
- `critical_hit_gate_approved` — user confirmed
- `critical_hit_gate_declined` — user declined

## Invariants

- Cannot fire when `risk_level != low`
- Cannot fire for cross-module or architectural tasks
- Approval is still required (inline gate is not auto-approve)
- If any condition is ambiguous, fall back to `main_mission`
```

### Step 3: Verificar que os contratos são YAML válido

```bash
python3 -c "import yaml,sys; yaml.safe_load(open('internal/embed/defaults/contracts/critical-hit.yaml'))" && echo "YAML OK"
```

Expected: `YAML OK`

### Step 4: Commit

```bash
git add internal/embed/defaults/contracts/critical-hit.yaml internal/embed/defaults/contracts/11-critical-hit.md
git commit -m "feat(contracts): add Critical Hit routing decision contract"
```

---

## Task 5: Atualizar `00-routing.md` e `intake.yaml` com routing_matrix

**Files:**
- Modify: `internal/embed/defaults/contracts/00-routing.md`
- Modify: `internal/embed/defaults/contracts/intake.yaml`
- Modify: `internal/embed/defaults/contracts/02-intake.md`

### Step 1: Atualizar `00-routing.md`

Substituir a seção `## Routes` por:

```markdown
## Routes

- **Quick Draw** — only for explicit quick capture / note append requests
- **Critical Hit** — fast path for low-risk doc/content edits (see `11-critical-hit.md`)
- **Main Mission** — every other request

## Route Selection Order

1. Quick Draw keywords detected → Quick Draw
2. Critical Hit conditions satisfied (see `critical-hit.yaml`) → Critical Hit
3. Default → Main Mission

## Main Mission Sequence

`bootstrap → preflight → intake → discovery → refinement → approval_gate → execution? → adr? → learning`

## Critical Hit Sequence

`bootstrap → preflight → intake → critical_hit_gate → execution → learning`

## Contract Lookup

When operating inside the main mission, consult contracts in this order:

1. `01-bootstrap.md`
2. `02-intake.md`
3. `03-discovery.md`
4. `04-refinement.md`
5. `05-approval-gate.md`
6. `06-execution.md`
7. `07-adr.md`
8. `08-learning.md`
9. `09-response.md`
10. `10-telemetry.md`
11. `11-critical-hit.md` ← only when evaluating Critical Hit eligibility

## Invariants

- No direct repository mutation without canonical pipeline evidence
- No execution without explicit approval (applies to ALL routes including Critical Hit)
- No slot work performed by Strategist itself
```

### Step 2: Adicionar `routing_matrix` e `critical_hit_triggers` em `intake.yaml`

Após a seção `quick_draw_behavior`, adicionar:

```yaml
  routing_matrix:
    evaluation_order:
      1: quick_draw   # checked first (keyword-based, explicit user signal)
      2: critical_hit # checked second (automatic classifier)
      3: main_mission # default (always safe)

    critical_hit_conditions:
      task_type_allow: [doc_edit, content_update, readme_update, comment_update]
      risk_level_require: low
      file_count_max: 5
      keyword_deny: [investigar, propor, auditar, projetar, redesenhar, arquitetura, refatorar]
      task_type_deny: [architecture_analysis, refactor, security_audit, new_feature]
      tie_break: main_mission

    emit_on_route_selected:
      format: "[Strategist] phase=intake route_selected={route} task_type={task_type} risk_level={risk_level}"
      level: INFO
```

### Step 3: Atualizar `02-intake.md`

Adicionar ao final da seção `## Required Behavior`:

```markdown
- evaluate `routing_matrix` after classification:
  - if Quick Draw triggers match → route `quick_draw`
  - else if Critical Hit conditions all satisfied → route `direct_execute` (emit `critical_hit_triggered`)
  - else → route `standard`
- emit `route_selected` event with the resolved route name
```

### Step 4: Verificar YAML válido

```bash
python3 -c "import yaml,sys; yaml.safe_load(open('internal/embed/defaults/contracts/intake.yaml'))" && echo "YAML OK"
```

Expected: `YAML OK`

### Step 5: Commit

```bash
git add internal/embed/defaults/contracts/00-routing.md \
        internal/embed/defaults/contracts/intake.yaml \
        internal/embed/defaults/contracts/02-intake.md
git commit -m "feat(contracts): add Critical Hit routing_matrix to intake and update routing contract"
```

---

## Task 6: Timing no Dojo — `DojoTimingCriteria` e `CheckTiming`

**Files:**
- Modify: `internal/domain/types.go` — novo struct + campo em `DojoCriteria`
- Modify: `internal/dojo/checker.go` — `CheckTiming`, chamada em `Run`
- Test: `internal/dojo/checker_test.go`

### Step 1: Escrever testes que falham

Adicionar em `checker_test.go`:

```go
func TestCheckTiming_Pass(t *testing.T) {
    logDir := t.TempDir()
    logPath := filepath.Join(logDir, "emit.log")
    require.NoError(t, os.WriteFile(logPath,
        []byte("ranger_start\ntotal_wall_time_ms=1200\napproval_prompt\n"), 0o644))

    criteria := domain.DojoCriteria{
        TimingCriteria: &domain.DojoTimingCriteria{
            MaxWallTimeMs: 30000,
        },
    }
    items := dojo.CheckTiming(criteria, logPath)
    require.Len(t, items, 1)
    assert.True(t, items[0].Passed, "expected pass: %s", items[0].Detail)
}

func TestCheckTiming_Fail_ExceedsMax(t *testing.T) {
    logDir := t.TempDir()
    logPath := filepath.Join(logDir, "emit.log")
    require.NoError(t, os.WriteFile(logPath,
        []byte("total_wall_time_ms=45000\n"), 0o644))

    criteria := domain.DojoCriteria{
        TimingCriteria: &domain.DojoTimingCriteria{
            MaxWallTimeMs: 30000,
        },
    }
    items := dojo.CheckTiming(criteria, logPath)
    require.Len(t, items, 1)
    assert.False(t, items[0].Passed)
    assert.Contains(t, items[0].Detail, "45000")
}

func TestCheckTiming_NilCriteria_Skip(t *testing.T) {
    criteria := domain.DojoCriteria{}
    items := dojo.CheckTiming(criteria, t.TempDir()+"/emit.log")
    assert.Empty(t, items)
}

func TestCheckTiming_LogMissing(t *testing.T) {
    criteria := domain.DojoCriteria{
        TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 30000},
    }
    items := dojo.CheckTiming(criteria, filepath.Join(t.TempDir(), "nonexistent.log"))
    require.Len(t, items, 1)
    assert.False(t, items[0].Passed)
    assert.Contains(t, items[0].Detail, "emit.log not found")
}

func TestCheckTiming_FieldMissing(t *testing.T) {
    logDir := t.TempDir()
    logPath := filepath.Join(logDir, "emit.log")
    require.NoError(t, os.WriteFile(logPath, []byte("ranger_start\nranger_done\n"), 0o644))

    criteria := domain.DojoCriteria{
        TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 30000},
    }
    items := dojo.CheckTiming(criteria, logPath)
    require.Len(t, items, 1)
    assert.False(t, items[0].Passed)
    assert.Contains(t, items[0].Detail, "total_wall_time_ms not found")
}
```

### Step 2: Rodar para confirmar que falham

```bash
go test ./internal/dojo/... -run TestCheckTiming -v
```

Expected: FAIL — `DojoTimingCriteria` e `CheckTiming` não existem.

### Step 3: Adicionar `DojoTimingCriteria` em `types.go`

No final do bloco de tipos Dojo em `types.go`, adicionar:

```go
// DojoTimingCriteria specifies wall-time performance constraints for a scenario.
// MaxWallTimeMs is extracted from the total_wall_time_ms= field in emit.log.
type DojoTimingCriteria struct {
    MaxWallTimeMs int `yaml:"max_wall_time_ms"`
}
```

E adicionar campo em `DojoCriteria`:

```go
TimingCriteria *DojoTimingCriteria `yaml:"timing_criteria,omitempty"`
```

### Step 4: Implementar `CheckTiming` em `checker.go`

Adicionar imports necessários no topo de `checker.go`:
```go
import (
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"

    "github.com/SergioLacerda/strategist-skill/internal/domain"
    "gopkg.in/yaml.v3"
)
```

Adicionar a função `CheckTiming`:

```go
// CheckTiming validates timing constraints from a timing_criteria block.
// It reads total_wall_time_ms=<value> from the emit log.
// If timing_criteria is nil, returns empty (no check performed).
func CheckTiming(criteria domain.DojoCriteria, logPath string) []domain.DojoCheckItem {
    if criteria.TimingCriteria == nil {
        return nil
    }
    tc := criteria.TimingCriteria

    if !fileExists(logPath) {
        return []domain.DojoCheckItem{{
            Label:  "timing total_wall_time_ms",
            Passed: false,
            Detail: "emit.log not found — run the LLM scenario first",
        }}
    }

    raw, err := os.ReadFile(logPath)
    if err != nil {
        return []domain.DojoCheckItem{{
            Label:  "timing total_wall_time_ms",
            Passed: false,
            Detail: err.Error(),
        }}
    }

    log := string(raw)
    const field = "total_wall_time_ms="
    idx := strings.Index(log, field)
    if idx < 0 {
        return []domain.DojoCheckItem{{
            Label:  "timing total_wall_time_ms",
            Passed: false,
            Detail: "total_wall_time_ms not found in emit.log",
        }}
    }

    rest := log[idx+len(field):]
    end := strings.IndexAny(rest, " \t\n\r")
    if end < 0 {
        end = len(rest)
    }
    valStr := rest[:end]
    val, err := strconv.Atoi(valStr)
    if err != nil {
        return []domain.DojoCheckItem{{
            Label:  "timing total_wall_time_ms",
            Passed: false,
            Detail: fmt.Sprintf("cannot parse total_wall_time_ms=%q: %v", valStr, err),
        }}
    }

    passed := val <= tc.MaxWallTimeMs
    return []domain.DojoCheckItem{{
        Label:  "timing total_wall_time_ms",
        Passed: passed,
        Detail: ifFail(passed, fmt.Sprintf("wall time %d ms exceeds max %d ms", val, tc.MaxWallTimeMs)),
    }}
}
```

### Step 5: Chamar `CheckTiming` em `Run`

Na função `Run` de `checker.go`, adicionar após a chamada a `CheckManifests`:

```go
result.Items = append(result.Items, CheckTiming(criteria, emitLogPath)...)
```

### Step 6: Rodar todos os testes do dojo

```bash
go test ./internal/dojo/... -v -count=1
```

Expected: todos PASS incluindo os 5 novos TestCheckTiming_*.

### Step 7: Commit

```bash
git add internal/domain/types.go internal/dojo/checker.go internal/dojo/checker_test.go
git commit -m "feat(dojo): add timing_criteria support with CheckTiming and DojoTimingCriteria"
```

---

## Task 7: Cenário de dojo `critical-hit`

**Files:**
- Create: `.analysis/dojo/critical-hit/input.yaml`
- Create: `.analysis/dojo/critical-hit/criteria.yaml`

### Step 1: Criar `input.yaml`

Criar `.analysis/dojo/critical-hit/input.yaml`:

```yaml
scenario: critical-hit
description: >
  [dojo-fixture] Direct edit of a documentation file via the Critical Hit fast path.
  Validates that Ranger and Archivist are NOT invoked and Sniper fires directly after gate.

prompt: >
  [dojo-fixture] edite docs/dojo/critical-hit-fixture.md adicionando uma linha:
  "CRITICAL_HIT_CANARY: validação de rota direta"

task_type: doc_edit
risk_level: low
file_count: 1
target_path: docs/dojo/critical-hit-fixture.md

gate_response: sim
```

### Step 2: Criar `criteria.yaml`

Criar `.analysis/dojo/critical-hit/criteria.yaml`:

```yaml
scenario: critical-hit
description: >
  Critical Hit: doc edit via fast path — Ranger e Archivist não invocados,
  gate inline apresentado, Sniper escreve apenas o arquivo alvo.
run_dir: "dojo/run"
auto_stop_at_gate: false

files_created:
  - path: "docs/dojo/critical-hit-fixture.md"
    must_contain:
      - "CRITICAL_HIT_CANARY"

pipeline:
  must_stop_at: ""
  slots_invoked: [execution]
  slots_not_invoked: [discovery, refinement]

emit_log:
  must_contain:
    - critical_hit_triggered
    - critical_hit_gate_approved
    - sniper_start
    - sniper_done
  must_not_contain:
    - ranger_start
    - archivist_start

timing_criteria:
  max_wall_time_ms: 60000
```

### Step 3: Validar YAML válido

```bash
python3 -c "
import yaml
for f in ['.analysis/dojo/critical-hit/input.yaml', '.analysis/dojo/critical-hit/criteria.yaml']:
    yaml.safe_load(open(f))
    print(f'OK: {f}')
"
```

Expected:
```
OK: .analysis/dojo/critical-hit/input.yaml
OK: .analysis/dojo/critical-hit/criteria.yaml
```

### Step 4: Verificar que o dojo lista o novo cenário

```bash
strategist dojo list
```

Expected: `critical-hit` aparece na lista.

### Step 5: Commit

```bash
git add .analysis/dojo/critical-hit/
git commit -m "feat(dojo): add critical-hit scenario with timing_criteria"
```

---

## Task 8: Suite completa de testes + verificação final

### Step 1: Rodar todos os testes do projeto

```bash
go test ./... -v -count=1 2>&1 | tail -30
```

Expected: sem FAIL.

### Step 2: Verificar que os novos símbolos estão acessíveis

```bash
go vet ./...
```

Expected: sem erros.

### Step 3: Verificar que os contratos YAML são todos válidos

```bash
python3 -c "
import yaml, glob, sys
errors = []
for f in glob.glob('internal/embed/defaults/contracts/**/*.yaml', recursive=True):
    try:
        yaml.safe_load(open(f))
    except yaml.YAMLError as e:
        errors.append(f'{f}: {e}')
if errors:
    print('ERRORS:', errors); sys.exit(1)
print(f'All contracts OK ({len(list(glob.glob(\"internal/embed/defaults/contracts/**/*.yaml\", recursive=True)))} files)')
"
```

Expected: `All contracts OK (N files)`

### Step 4: Commit final de verificação (se houver ajustes menores)

```bash
git add -p  # revisar qualquer ajuste residual
git commit -m "chore: finalize Critical Hit implementation — all tests green"
```

---

## Resumo do que foi implementado

| Gap | Implementado em |
|-----|----------------|
| Gap 1 — Rota `direct_execute` | Tasks 1, 3, 4 |
| Gap 2 — `routing_matrix` em intake | Task 5 |
| Gap 3 — Exceção em `pipeline_bypass` | Task 2 |
| Gap 4 — Timing no dojo | Task 6 |
| Gap 5 — Classificação automática | Task 5 (routing_matrix automático) |
| Contrato "Critical Hit" | Task 4 |
| Cenário dojo `critical-hit` | Task 7 |
