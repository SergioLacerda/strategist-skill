# Strategist Runtime Pipeline Adherence Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Garantir que o `/strategist` reflita no chat a execução real do pipeline, com resolução de perfil determinística (local/global) e telemetria obrigatória por fase/papel.

**Architecture:** Introduzir um resolvedor de perfil com modo `auto|local|global`, unificar emissão semântica de eventos obrigatórios em runtime e aplicar renderização por persona (`pragmatic` técnico, `epic` com emojis) sem alterar o contrato semântico. A validação final bloqueia missões com eventos obrigatórios ausentes.

**Tech Stack:** Go (cmd/internal), YAML contracts (`.strategist` e defaults embed), testes Go existentes em `internal/install` e `strategist/tests`.

---

### Task 1: Mapear pontos de entrada e emissão atuais

**Files:**
- Read: `cmd/strategist/install.go`
- Read: `internal/install/installer.go`
- Read: `internal/install/shim.go`
- Read: `.strategist/SKILL.md`
- Read: `.strategist/skill.yaml`

**Step 1: Listar onde profile/path é resolvido hoje**
- Identificar funções e structs responsáveis pela carga de config.

**Step 2: Listar onde eventos são emitidos hoje**
- Inventariar eventos atuais de pipeline, gate, opportunity e persona.

**Step 3: Registrar gaps contra o design aprovado**
- Gerar checklist local (working notes) com faltas: profile_source, role_events, compliance_footer etc.

**Step 4: Validar baseline sem alterações**
Run: `go test ./internal/install ./strategist/tests`
Expected: suíte baseline passa (ou falhas conhecidas documentadas no output).

### Task 2: Implementar resolução de perfil `--profile=auto|local|global`

**Files:**
- Modify: `cmd/strategist/install.go`
- Modify: `internal/install/installer.go`
- Modify: `internal/install/active_yaml.go`
- Test: `internal/install/active_yaml_test.go`
- Test: `internal/install/install_test.go`

**Step 1: Escrever testes que falham para seleção de perfil**
- Cobrir: `auto(local ok)`, `auto(fallback global)`, `local inválido => blocked`, `global forçado`.

**Step 2: Rodar testes para garantir falha inicial**
Run: `go test ./internal/install -run Profile -v`
Expected: FAIL por comportamento ainda não implementado.

**Step 3: Implementar resolvedor determinístico**
- Adicionar parsing de flag `--profile`.
- Implementar ordem: shim skill_root -> `./.strategist` -> `~/.strategist` em `auto`.
- Implementar erro bloqueante para `local` inválido.

**Step 4: Expor metadados de origem para emissão**
- Garantir disponibilidade de `profile_source`, `profile_path`, `active_yaml`, `roles_config` no runtime.

**Step 5: Re-rodar testes de profile**
Run: `go test ./internal/install -run Profile -v`
Expected: PASS.

### Task 3: Definir contrato de eventos obrigatórios no runtime

**Files:**
- Modify: `.strategist/SKILL.md`
- Modify: `.strategist/skill.yaml`
- Modify: `internal/embed/defaults/SKILL.md`
- Modify: `internal/embed/defaults/skill.yaml` (se existir; se não, ajustar origem equivalente embed)
- Test: `strategist/tests/specs/approval-gate.feature`
- Test: `strategist/tests/specs/slot-contracts.feature`

**Step 1: Escrever cenários de contrato para eventos obrigatórios**
- Cenários: presença de pipeline_header, phase_events, role_events, gate_events, opportunity_events, treasure_events, compliance_footer.

**Step 2: Rodar cenários para falhar inicialmente**
Run: `go test ./strategist/tests -run SpecAlignment -v`
Expected: FAIL em missing required telemetry.

**Step 3: Atualizar contrato canônico**
- Inserir regra explícita de `blocked reason=missing_required_telemetry` quando faltarem eventos.

**Step 4: Ajustar camada runtime para completar emissão mínima**
- Garantir evento explícito para itens não aplicáveis (`none`/`skipped` + `reason`).

**Step 5: Revalidar testes de contrato**
Run: `go test ./strategist/tests -v`
Expected: PASS para os novos cenários.

### Task 4: Implementar renderização por persona sem alterar semântica

**Files:**
- Modify: `.strategist/personas/pragmatic.yaml`
- Modify: `.strategist/personas/epic.yaml`
- Modify: `internal/embed/defaults/personas/pragmatic.yaml`
- Modify: `internal/embed/defaults/personas/epic.yaml`
- Test: `strategist/tests/spec_alignment_test.go`

**Step 1: Criar teste de equivalência semântica entre personas**
- Mesmo conjunto de eventos exigidos, mensagens com estilo distinto.

**Step 2: Rodar teste para falhar inicialmente**
Run: `go test ./strategist/tests -run Persona -v`
Expected: FAIL por ausência de campos/eventos em uma das personas.

**Step 3: Atualizar templates de mensagem**
- `pragmatic`: bloco técnico explícito.
- `epic`: mensagens com emojis mantendo campos semânticos equivalentes.

**Step 4: Re-rodar teste de persona**
Run: `go test ./strategist/tests -run Persona -v`
Expected: PASS.

### Task 5: Integrar Ataque de Oportunidade e Baú do Tesouro na trilha de evidência

**Files:**
- Modify: `.strategist/SKILL.md`
- Modify: `.strategist/skill.yaml`
- Modify: `strategist/tests/fixtures/*.yaml` (casos existentes + novos)
- Modify: `strategist/tests/run-tests.sh` (se necessário)

**Step 1: Criar fixtures para casos aplicável/não aplicável**
- `opportunity_attack items>0`
- `opportunity_attack items=0`
- `treasure_chest_loaded ids=...`
- `treasure_chest_loaded none`

**Step 2: Rodar harness para falhar inicialmente**
Run: `bash strategist/tests/run-tests.sh`
Expected: FAIL em expected_event não emitido.

**Step 3: Implementar emissão explícita em todos os ramos**
- Sem ramos silenciosos para esses mecanismos.

**Step 4: Re-rodar harness**
Run: `bash strategist/tests/run-tests.sh`
Expected: PASS.

### Task 6: End-to-end de regressão e critérios de aceite

**Files:**
- Modify: `.analysis/refined/2026-06-01-analise-pendencias-em-aberto.md` (apenas status/evidência, se desejado)
- Optional docs: `readme.md` (se CLI flag `--profile` for exposta ao usuário)

**Step 1: Rodar suíte combinada**
Run: `go test ./internal/install ./strategist/tests ./internal/embed ./internal/domain`
Expected: PASS.

**Step 2: Validar critérios de aceite manualmente**
- Invocar `/strategist` com profile local/global e verificar presença de bloco de evidência.
- Validar diferenças de apresentação entre `pragmatic` e `epic`.

**Step 3: Registrar evidências de execução**
- Anotar comandos e resultados no artefato de análise/refined.

**Step 4: Encerrar sem git commit (preferência do usuário)**
- Não executar comandos `git commit`.

---

## Definition of Done

- `--profile=auto|local|global` funcional com comportamento conforme design.
- `/strategist` sempre emite evidência mínima obrigatória no chat.
- Ausência de evento obrigatório bloqueia missão com `missing_required_telemetry`.
- `epic` usa mensagens com emoji sem quebrar contrato semântico.
- Ataque de Oportunidade e Baú do Tesouro aparecem sempre como evento (ou `none`).
