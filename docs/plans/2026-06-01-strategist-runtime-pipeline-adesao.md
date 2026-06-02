# Strategist Runtime Pipeline Adherence Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fazer o `/strategist` refletir no chat a execução real do pipeline local, com origem de perfil explícita, `mission_checkpoint` e `mission_metrics`, sem fallback silencioso e sem sucesso quando a telemetria obrigatória faltar.

**Architecture:** Manter a resolução local deterministica de perfil, ligar o tracker de missao ao runtime real do CLI, e centralizar a emissao dos eventos obrigatorios de pipeline/fase/papel/gate/oportunidade/tesouro/compliance. A renderizacao por persona continua variando na apresentacao, mas nao no contrato semantico.

**Tech Stack:** Go (`cmd/strategist`, `internal/telemetry`), YAML de contratos/personas em `.strategist` e `internal/embed/defaults`, testes existentes em `strategist/tests` e `cmd/strategist/*_test.go`.

## What Is Already Covered

These pieces are already present and should be treated as baseline, not reopened as design work:

- `pipeline=starting` is emitted by the CLI root.
- `mission_checkpoint` exists in the skill docs, personas, and intake/progress contracts.
- `mission_metrics` exists in the contracts, docs, personas, telemetry schema, and tracker helpers.
- `internal/telemetry/mission_run.go` tracks `start`, `intake`, `ranger`, `lines_emitted`, and token counters.
- `internal/telemetry/mission_metrics.go` formats the metrics payload.
- `cmd/strategist/root.go` already creates and closes a mission tracker.
- Existing commands already mark visible output and first-action timings.
- Baseline benchmarks and performance notes already exist in the analysis/docs layer.
- Contract alignment tests already assert the presence of `profile`, `mode`, `speed`, and `mission_metrics` keys.

## What Is Still Missing In Runtime

These are the runtime gaps the implementation still has to close:

- The pipeline still does not have a single, explicit mission orchestrator that emits every required event in order.
- The current telemetry is command-level; it is not yet the full semantic pipeline contract for a mission run.
- `tokens_in` and `tokens_out` are tracked structurally, but there is no real source wired from the executor/model layer yet.
- Role events, gate events, opportunity events, treasure events, and compliance footer events still need a canonical runtime emission path.
- Missing required telemetry does not yet behave like a hard runtime blocker everywhere it should.
- The runtime does not yet guarantee that `epic` and `pragmatic` render the same semantics with different presentation.

## Implementation Plan

### Task 1: Freeze the runtime contract against the current covered surface

**Files:**
- Update: `.analysis/pending/2026-06-01-strategist-pipeline-adesao-runtime-design.md`
- Update: `docs/skill-internals.md`
- Update: `strategist/SKILL.md`
- Update: `internal/embed/defaults/SKILL.md`
- Update: `strategist/tests/spec_alignment_test.go`

**Checklist**
- [ ] Record the current baseline explicitly.
  - Document that `pipeline=starting`, `mission_checkpoint`, and `mission_metrics` are already covered.
  - Document that the remaining work is runtime orchestration, not schema invention.
- [ ] Tighten the acceptance language.
  - Make the design and docs say that missing required telemetry must block the mission.
  - Keep the local-only profile rule explicit.
- [ ] Expand alignment tests.
  - Assert the contract still exposes the already covered signals.
  - Add a guard that fails if the runtime docs regress to a fallback/global assumption.

**Acceptance Criteria**
- The plan/docs now describe the current baseline as covered, not pending.
- The local-only rule remains explicit.
- Tests fail if fallback/global assumptions reappear in the runtime contract text.

### Task 2: Introduce a canonical runtime mission emitter

**Files:**
- Modify: `internal/telemetry/mission_run.go`
- Modify: `internal/telemetry/mission_metrics.go`
- Modify: `cmd/strategist/root.go`
- Modify: `cmd/strategist/install.go`
- Modify: `cmd/strategist/compile.go`
- Modify: `cmd/strategist/check_stale.go`
- Modify: `cmd/strategist/validate.go`
- Modify: `cmd/strategist/sync_governance.go`
- Add/modify tests under `internal/telemetry` and `cmd/strategist`

**Checklist**
- [ ] Define the mission event order.
  - Emit a stable header first.
  - Emit phase/role/gate/opportunity/treasure/compliance events in a single canonical path.
- [ ] Centralize emission helpers.
  - Reuse the tracker for timestamps and counts.
  - Add helpers so commands do not each invent their own event format.
- [ ] Mark the runtime boundary.
  - Separate "command output" from "mission evidence" so the final footer can validate completeness.
- [ ] Add tests for the canonical emitter.
  - Cover happy path, skipped path, and blocked path.
  - Assert the emitted order and required fields.

**Acceptance Criteria**
- A single helper path can emit the mission evidence set in order.
- The footer can validate completeness from emitted evidence, not from ad hoc command output.
- Tests cover the canonical emitter for success, skip, and block cases.

### Task 3: Wire required semantic events into the live pipeline

**Files:**
- Modify: the runtime path that composes the Strategist mission flow
- Modify: persona templates if a field is missing from the rendered output
- Add tests/fixtures for the runtime flow

**Checklist**
- [ ] Emit pipeline header and profile origin.
  - Keep `profile_mode`, `profile_path`, `active_yaml`, `persona_resolved`, `reason`, and `output_profile` visible.
- [ ] Emit role events.
  - Make `ranger`, `archivist`, and `sniper` transitions explicit when they exist.
- [ ] Emit gate events.
  - Show gate prompts and outcomes.
  - Record `approved`, `declined`, or `review` explicitly.
- [ ] Emit opportunity and treasure events.
  - Emit explicit `items=0` or `none` when nothing is applicable.
  - Do not omit these sections silently.
- [ ] Emit compliance footer.
  - Report expected, executed, and missing counts at the end of the mission.

**Acceptance Criteria**
- Every mission emits the profile/origin header.
- Role, gate, opportunity, treasure, and compliance evidence are visible when applicable, and explicit `none`/`0` when not.
- The live runtime path does not silently skip required evidence sections.

### Task 4: Enforce hard failure on missing telemetry

**Files:**
- Modify: `cmd/strategist/root.go`
- Modify: runtime validation code near mission completion
- Add tests for blocked completion cases

**Checklist**
- [ ] Make telemetry completeness part of completion criteria.
  - If a required block is missing, the mission ends as blocked.
- [ ] Preserve partial diagnostics.
  - The runtime should still print what happened before the block.
  - Do not collapse failures into a silent success path.
- [ ] Add regression tests.
  - Cover missing `mission_metrics`.
  - Cover missing phase/role/gate sections.
  - Cover the explicit blocked footer.

**Acceptance Criteria**
- Missing required telemetry always results in blocked completion.
- Partial output remains visible for diagnosis.
- Regression tests fail if the runtime can still report success without the required evidence.

### Task 5: Source token counts from a real runtime boundary

**Files:**
- Modify: `internal/telemetry/mission_run.go`
- Modify: the executor/model integration point when available
- Add tests for token accounting behavior

**Checklist**
- [ ] Define the token source.
  - Identify the authoritative boundary for input and output tokens.
- [ ] Feed the tracker from that boundary.
  - Keep the helper fallback-safe until the real source is wired.
  - Avoid fabricating values in the runtime path.
- [ ] Verify metrics remain stable.
  - Ensure token counts are present when data exists and stay zero only when the source is genuinely unavailable.

**Acceptance Criteria**
- Token accounting has a named authoritative source.
- The runtime never invents token counts.
- Zero values are explainable by genuine source absence, not by missing plumbing.

### Task 6: Close the loop with end-to-end tests and docs

**Files:**
- Modify: `strategist/tests/spec_alignment_test.go`
- Add/update runtime fixtures
- Update `docs/skill-internals.md` if the emitted contract changes

**Checklist**
- [ ] Add end-to-end coverage.
  - Validate the final chat output contains the mandatory evidence block.
- [ ] Add persona coverage.
  - Confirm `pragmatic` and `epic` preserve the same semantics.
- [ ] Validate the blocked path.
  - Ensure missing telemetry never reports success.
- [ ] Re-run governance validation.
  - Confirm the repo stays aligned with SDD and the skill contracts.

**Acceptance Criteria**
- End-to-end tests validate the mandatory evidence block.
- Persona-specific rendering remains semantically equivalent.
- Governance validation passes after the runtime changes.

## Definition Of Done

- The runtime emits a complete mission evidence block in the chat.
- `mission_checkpoint` and `mission_metrics` remain present and validated.
- Missing required telemetry blocks mission completion.
- The current local-only profile behavior stays explicit and deterministic.
- `epic` and `pragmatic` differ in presentation only, not in meaning.
- Token metrics are wired to a real source, or clearly remain unavailable without being faked.
