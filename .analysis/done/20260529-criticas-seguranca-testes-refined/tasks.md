# Tasks — Segurança e Testes (Críticos)
**Mission ID:** 20260529-criticas
**Analysis group:** seguranca-testes

---

## Checklist de Implementação

### Bloco A — bootstrap.sh: Verificação de Integridade

- [ ] **A1.** Em `.github/workflows/release.yml`, adicionar ao passo "Package release assets": gerar `SHA256SUMS` com `sha256sum strategist-skill-*.tar.gz strategist-skill-*.zip > SHA256SUMS` e incluir `SHA256SUMS` nos assets do release (na lista `files:` do step `softprops/action-gh-release`).

- [ ] **A2.** Em `bootstrap.sh`, após o download do tarball, adicionar: download de `SHA256SUMS` do mesmo release URL (`${ARCHIVE_URL%.tar.gz}` → URL do asset `SHA256SUMS`), seguido de `sha256sum --check --ignore-missing SHA256SUMS`. Se a verificação falhar, emitir erro e sair com código 1.

- [ ] **A3.** Em `bootstrap.sh`, no bloco `resolve_ref()` (fallback para `main`), substituir a mensagem informativa por um aviso de segurança explícito: `[Strategist] AVISO DE SEGURANÇA: instalando de branch '${DEFAULT_REF}' sem verificação de integridade. Use --ref=vX.Y.Z para instalação verificada.`

- [ ] **A4.** Em `readme.md`, adicionar aviso no bloco de instalação: descrever o risco de piping curl direto e instruir uso de `--ref=vX.Y.Z` para ambientes de produção.

### Bloco B — install.sh: Rollback do Wizard

- [ ] **B1.** Em `strategist/install.sh`, adicionar array `INSTALL_MANIFEST=()` no início do script e uma função `manifest_add <path>` que appenda o path ao array.

- [ ] **B2.** Adicionar função `rollback()` que itera `INSTALL_MANIFEST` em ordem reversa: remove arquivos com `rm -f`, remove diretórios criados com `rmdir --ignore-fail-on-non-empty`. Ao final, emite `[Strategist] WARN: install rolled back — workspace restaurado ao estado anterior.`

- [ ] **B3.** Adicionar `trap 'rollback' ERR` logo após a declaração de `INSTALL_MANIFEST`. Garantir que o trap seja removido ao final de uma instalação bem-sucedida com `trap - ERR`.

- [ ] **B4.** Instrumentar `copy_skill_runtime()`, `write_active_yaml()`, `install_agent_shims()` e a seção de escrita do wizard com chamadas a `manifest_add <path>` para cada arquivo e diretório criado.

### Bloco C — Preflight: Validação YAML

- [ ] **C1.** Criar `.strategist/schemas/active.schema.yaml` definindo campos obrigatórios de `active.yaml`: `mode` (string, não-nulo), `base_path` (string, não-nulo), `roles_config` (string, não-nulo).

- [ ] **C2.** Criar `.strategist/schemas/roles.schema.yaml` definindo campos obrigatórios de `roles/<config>.yaml`: `discovery` (string, não-nulo), `refinement` (string, não-nulo), `execution` (string, não-nulo).

- [ ] **C3.** Em `.strategist/SKILL.md`, inserir step `2a.validate` imediatamente após o carregamento dos arquivos de configuração (entre os steps 2a e 2b atuais): para cada campo obrigatório de `active.yaml` e do roles config carregado, verificar presença e não-nulidade. Se falhar: emitir `[Strategist] phase=preflight status=blocked reason=yaml_validation_failed file=<path> field=<field>` e STOP.

- [ ] **C4.** Em `.strategist/index.yaml`, adicionar `active.schema.yaml` e `roles.schema.yaml` sob `load_always`.

### Bloco D — Contrato de Interface entre Slots

- [ ] **D1.** Criar `.strategist/schemas/slot-output.schema.yaml` com os contratos de output para discovery slot (campos: `artifact_path`, `status`) e refinement slot (campos: `artifact_dir`, `status`, presença dos três arquivos `proposal.md`, `design.md`, `tasks.md`).

- [ ] **D2.** Em `.strategist/SKILL.md`, no passo 5a (Ranger), adicionar: após o slot retornar, verificar output contra `slot-output.schema.yaml#discovery_slot`. Se inválido: emitir `[Strategist] phase=analysis status=blocked reason=slot_output_invalid` e STOP.

- [ ] **D3.** Em `.strategist/SKILL.md`, no passo 5e (Archivist), adicionar: após o slot retornar, verificar output contra `slot-output.schema.yaml#refinement_slot`. Se inválido: emitir `[Strategist] phase=refinement status=blocked reason=slot_output_invalid` e STOP.

- [ ] **D4.** Em `.strategist/index.yaml`, adicionar `slot-output.schema.yaml` sob `load_always`.

### Bloco E — Test Harness

- [ ] **E1.** Criar diretório `strategist/tests/fixtures/` com os seguintes arquivos de fixture YAML:
  - `approval-bypass.yaml` — cenário: Sniper invocado sem approval gate response; `expected_event`: `phase=execution status=blocked reason=approval_bypass`
  - `slot-risk-mismatch.yaml` — cenário: provider com `risk_score` incorreto; `expected_event`: `phase=preflight status=blocked reason=slot_risk_mismatch`
  - `discovery-failed.yaml` — cenário: slot de discovery retorna `status=failed`; `expected_event`: `phase=analysis status=blocked reason=ranger_failed`
  - `yaml-null-field.yaml` — cenário: `active.yaml` com `base_path: null`; `expected_event`: `phase=preflight status=blocked reason=yaml_validation_failed field=base_path`
  - `side-quest-bypass.yaml` — cenário: file move executado sem mini approval gate; `expected_event`: `phase=side_quest_execution status=blocked reason=side_quest_approval_bypass`

- [ ] **E2.** Criar `strategist/tests/run-tests.sh`: script que itera os fixtures, executa cada cenário e compara o output do agente com `expected_event` usando golden-file comparison. Retorna exit code 1 se qualquer fixture falhar.

- [ ] **E3.** Criar `.github/workflows/test.yml` com trigger em `push` e `pull_request`. Steps: checkout, `shellcheck bootstrap.sh strategist/install.sh`, `bash strategist/tests/run-tests.sh`, validação estrutural dos schemas YAML em `.strategist/schemas/` (ex: verificar que são YAML válido com `python3 -c "import yaml; yaml.safe_load(open(f))"` para cada arquivo em `schemas/`).

---

## Ordem de Implementação Recomendada

1. **Bloco A** (bootstrap checksum) — menor risco, maior impacto de segurança, independente
2. **Bloco B** (wizard rollback) — script local, independente
3. **Bloco C** (YAML validation) — modifica SKILL.md, requer novos schemas
4. **Bloco D** (slot output schema) — modifica SKILL.md, reutiliza padrão do Bloco C
5. **Bloco E** (test harness) — cobre todos os blocos anteriores; implementar por último para testar as correções

---

## Critérios de Conclusão

- CI verde em `.github/workflows/test.yml` com todos os 5 fixtures passando
- `bootstrap.sh` com `--ref=v1.0.0` falha explicitamente se o tarball for adulterado
- `bootstrap.sh` sem `--ref` emite aviso de segurança visível
- Preflight bloqueia com código de erro legível quando `active.yaml` tem campo nulo
- `install.sh --wizard` que falha no meio não deixa arquivos em `~/.claude/skills/` ou `.strategist/`
