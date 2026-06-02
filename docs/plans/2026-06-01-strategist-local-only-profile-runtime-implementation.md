# Strategist Local-Only Profile Runtime Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fazer o `/strategist` carregar perfil local por padrão (sem fallback global), permitir global só por flag explícita e provar no chat qual persona/perfil foi resolvido.

**Architecture:** Ajustar resolução de perfil no runtime e install, tornar diagnóstico de profile/persona obrigatório no início da execução, e bloquear em mismatch de renderização (`persona_resolved` vs estilo emitido). Cobrir com testes de contrato e integração.

**Tech Stack:** Go (`cmd/strategist`, `internal/install`), YAML (`.strategist/*`, `internal/embed/defaults/*`), testes Go (`internal/install`, `strategist/tests`, `internal/embed`).

---

### Task 1: Baseline e mapeamento dos pontos de resolução

**Files:**
- Read: `cmd/strategist/install.go`
- Read: `internal/install/installer.go`
- Read: `internal/install/shim.go`
- Read: `.strategist/skill.yaml`
- Read: `.strategist/SKILL.md`

**Step 1: Confirmar baseline dos testes**
Run: `go test ./internal/install ./strategist/tests ./internal/embed`
Expected: PASS baseline.

**Step 2: Mapear pontos de fallback global atuais**
- Identificar chamadas que leem `~/.strategist` por padrão.

**Step 3: Mapear pontos de emissão de eventos de perfil/persona**
- Identificar onde incluir `profile_mode`, `profile_source_path`, `persona_resolved`.

### Task 2: Resolução local-only no runtime

**Files:**
- Modify: `.strategist/skill.yaml`
- Modify: `.strategist/SKILL.md`
- Modify: `internal/embed/defaults/skill.yaml`
- Modify: `internal/embed/defaults/SKILL.md`
- Test: `strategist/tests/spec_alignment_test.go`

**Step 1: Escrever teste que falha para contrato local-only**
- Assert de ausência de fallback automático global no modo default.

**Step 2: Rodar teste para falhar**
Run: `go test ./strategist/tests -run Profile -v`
Expected: FAIL inicial.

**Step 3: Implementar contrato local-only**
- Resolver local por default.
- Permitir global só com flag explícita.
- Definir bloqueio em local inválido.

**Step 4: Sincronizar para defaults embed**
- Copiar mudanças de `.strategist/*` para `internal/embed/defaults/*` equivalentes.

**Step 5: Re-rodar testes de contrato**
Run: `go test ./strategist/tests -v`
Expected: PASS.

### Task 3: Instalação padrão local-only e global forçado explícito

**Files:**
- Modify: `cmd/strategist/install.go`
- Modify: `internal/install/installer.go`
- Modify: `internal/install/install_test.go`
- Modify: `internal/install/installer_whitebox_test.go`

**Step 1: Escrever testes que falham para install local-only**
- Padrão não cria/atualiza `~/.strategist`.
- Modo forçado global cria/atualiza global.

**Step 2: Rodar testes de install para falhar**
Run: `go test ./internal/install -run Install -v`
Expected: FAIL inicial nos novos cenários.

**Step 3: Implementar comportamento de instalação**
- Default: runtime local apenas.
- Global somente com flag/comando explícito já existente.
- Log explícito quando global forçado.

**Step 4: Ajustar shim conforme origem**
- Default: `skill_root=<target>/.strategist`.
- Global forçado: aponta para global.

**Step 5: Re-rodar testes de install**
Run: `go test ./internal/install -v`
Expected: PASS.

### Task 4: Diagnóstico obrigatório no chat

**Files:**
- Modify: `.strategist/personas/pragmatic.yaml`
- Modify: `.strategist/personas/epic.yaml`
- Modify: `.strategist/output-profiles/emit-taxonomy.yaml`
- Modify: `internal/embed/defaults/personas/pragmatic.yaml`
- Modify: `internal/embed/defaults/personas/epic.yaml`
- Modify: `internal/embed/defaults/output-profiles/emit-taxonomy.yaml`
- Test: `strategist/tests/spec_alignment_test.go`

**Step 1: Escrever teste que falha exigindo bloco diagnóstico**
- Campos obrigatórios: `profile_mode`, `profile_source_path`, `active_yaml_path`, `persona_resolved`, `reason`.

**Step 2: Rodar teste para falhar**
Run: `go test ./strategist/tests -run Diagnostic -v`
Expected: FAIL inicial.

**Step 3: Implementar mensagens obrigatórias nas personas**
- Incluir chaves e formato nos dois idiomas para `pragmatic` e `epic`.

**Step 4: Tornar emissões visíveis por taxonomy**
- Garantir nível INFO para diagnóstico/compliance críticos.

**Step 5: Re-rodar testes de alinhamento**
Run: `go test ./strategist/tests -v`
Expected: PASS.

### Task 5: Bloqueio por mismatch de persona/renderização

**Files:**
- Modify: `.strategist/SKILL.md`
- Modify: `.strategist/skill.yaml`
- Modify: `.strategist/contracts/tests/emit-taxonomy.test.yaml`
- Modify: `.strategist/contracts/tests/compliance-summary.test.yaml`
- Test: `strategist/tests/spec_alignment_test.go`

**Step 1: Escrever testes que falham para mismatch**
- `persona_resolved=epic` sem estilo épico => blocked.
- ausência de diagnóstico => blocked.

**Step 2: Rodar testes para falhar**
Run: `go test ./strategist/tests -run Persona -v`
Expected: FAIL inicial.

**Step 3: Implementar contratos de bloqueio**
- `missing_profile_diagnostics`
- `persona_render_mismatch`

**Step 4: Re-rodar testes de persona/compliance**
Run: `go test ./strategist/tests -v`
Expected: PASS.

### Task 6: Regressão completa e validação de governança

**Files:**
- Optional update evidence: `.analysis/refined/2026-06-01-analise-pendencias-em-aberto.md`

**Step 1: Rodar suíte técnica principal**
Run: `go test ./internal/install ./strategist/tests ./internal/embed ./internal/domain`
Expected: PASS.

**Step 2: Rodar validação SDD**
Run: `uv run sdd governance validate`
Expected: todos checks PASS.

**Step 3: Validar cenário manual mínimo**
- `/strategist` default: diagnóstico mostra local.
- `/strategist` com global forçado: diagnóstico mostra forced_global.
- `mode: epic`: primeiras mensagens de fase em estilo épico.

**Step 4: Encerramento sem commit**
- Não executar `git commit`.

---

## Definition of Done

- Runtime default sem fallback automático global.
- Global só em modo explícito forçado.
- Bloco de diagnóstico sempre visível no chat.
- Bloqueio automático para `missing_profile_diagnostics` e `persona_render_mismatch`.
- Testes e governança (`uv run sdd governance validate`) verdes.
