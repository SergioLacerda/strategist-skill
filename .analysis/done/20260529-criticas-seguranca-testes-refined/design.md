# Design — Segurança e Testes (Críticos)
**Mission ID:** 20260529-criticas
**Analysis group:** seguranca-testes

---

## Modules Index

| Módulo | Arquivo | Papel no sistema |
|--------|---------|-----------------|
| Curl installer | `bootstrap.sh` | Entry point público — baixa e executa o tarball de instalação |
| Local installer | `strategist/install.sh` | Configura o workspace: copia runtime, roda wizard, instala shims |
| Release CI | `.github/workflows/release.yml` | Empacota assets e publica GitHub Release |
| Test CI | `.github/workflows/test.yml` | **NOVO** — roda o test harness no CI |
| Skill instructions | `.strategist/SKILL.md` | Define o comportamento do agente; contém a spec do preflight |
| Schemas existentes | `.strategist/schemas/intake.schema.yaml`, `.strategist/schemas/progress-contract.yaml` | Schemas de validação já presentes mas não usados no preflight |
| Schema de slot output | `.strategist/schemas/slot-output.schema.yaml` | **NOVO** — contrato de output dos slot providers |
| Test fixtures | `strategist/tests/fixtures/` | **NOVO** — estados de missão simulados para o test harness |

### Boundaries

- `bootstrap.sh` ↔ GitHub Releases: dependência externa. Boundary clara: `bootstrap.sh` não conhece a estrutura interna do tarball, apenas executa `install.sh` dentro dele.
- `install.sh` ↔ `.strategist/`: escrita local no workspace do usuário. **Ambiguidade identificada:** não há definição explícita de quais arquivos o wizard é responsável por criar vs. quais são responsabilidade de `copy_skill_runtime`. Isso complica o rollback (ver item 5).
- Strategist (agente) ↔ slot providers: interface definida em prosa. **Boundary ausente:** nenhum schema define o contrato de output que um slot provider deve respeitar.

---

## Item 1 — bootstrap.sh: Verificação de Integridade

### Decisão de design

Adicionar geração de `SHA256SUMS` no passo de empacotamento do `release.yml`, e verificação do checksum antes da extração no `bootstrap.sh`.

```
release.yml (package step)
  │
  ├─ cria strategist-skill-X.Y.Z.tar.gz
  ├─ cria strategist-skill-X.Y.Z.zip
  └─ NOVO: sha256sum *.tar.gz *.zip > SHA256SUMS
           → inclui SHA256SUMS nos assets do release

bootstrap.sh (download step)
  │
  ├─ baixa strategist-skill-X.Y.Z.tar.gz  (já existente)
  ├─ NOVO: baixa SHA256SUMS do mesmo release
  ├─ NOVO: sha256sum --check --ignore-missing SHA256SUMS
  │         → falha explícita se checksum não bater
  └─ extrai e executa install.sh  (só após verificação)
```

**Limitação do happy path (sem release):** quando `resolve_ref()` não encontra release e cai em `main`, não há arquivo `SHA256SUMS` para verificar. Decisão: emitir um aviso explícito para o usuário e exigir `--ref=vX.Y.Z` para o fluxo verificado. O fallback para `main` deve ser marcado como inseguro no output.

**Arquivos afetados:**
- `bootstrap.sh`: bloco de download + extração
- `.github/workflows/release.yml`: passo "Package release assets"
- `readme.md`: aviso explícito sobre o risco de piping curl sem `--ref`

### Alternativa considerada e descartada
Assinar o checksum com GPG (maior segurança). Descartada por adicionar complexidade de gestão de chave sem benefício significativo para o perfil de risco atual (HTTPS já mitiga MITM para a maioria dos usuários; SHA256 sem assinatura ainda detecta corrupção acidental e torna ataques de supply chain mais difíceis).

---

## Item 2 — Preflight: Validação YAML

### Decisão de design

O preflight atual (SKILL.md §2) carrega arquivos sem validar estrutura. Os schemas já existem em `.strategist/schemas/` mas não são referenciados no preflight.

Adicionar um step `2a.validate` antes de qualquer uso dos valores carregados:

```
Preflight
  2a. Load internal domain         (existente)
  2a.validate NOVO ─────────────────────────────
  │  Para cada arquivo carregado que tem schema:
  │    active.yaml         → validar campos obrigatórios (mode, base_path, roles_config)
  │    roles/<config>.yaml → validar campos (discovery, refinement, execution não-nulos)
  │  Se validação falhar:
  │    emit blocked reason=yaml_validation_failed file=<path> field=<field>
  │    STOP
  2b. Load identity files          (existente)
  2c. Resolve slot providers        (existente)
  2d. Validate slot risk contracts  (existente)
  2e. Emit preflight done           (existente)
```

O schema de `active.yaml` não existe atualmente — precisa ser criado em `.strategist/schemas/active.schema.yaml`. O schema de `roles/<config>.yaml` também não existe — criar `.strategist/schemas/roles.schema.yaml`.

**Arquivos afetados:**
- `.strategist/SKILL.md`: inserir step 2a.validate
- `.strategist/schemas/active.schema.yaml`: **NOVO**
- `.strategist/schemas/roles.schema.yaml`: **NOVO**

**Boundary ambígua identificada:** o preflight é definido em SKILL.md (instrução de agente), mas a validação referencia schemas em `.strategist/schemas/`. O agente que executa o preflight precisa ser capaz de validar estrutura YAML. Esta validação é estrutural (campos obrigatórios, tipos, valores permitidos) — não requer um validador externo pesado; o agente pode verificar presença e não-nulidade dos campos durante o carregamento.

---

## Item 3 — Test Harness para Contratos Críticos

### Decisão de design

Criar um test harness baseado em fixtures YAML que simulam estados de missão e verificam que o agente emite os eventos corretos (ou para corretamente).

```
strategist/tests/
  fixtures/
    approval-bypass.yaml        ─ simula invocação do Sniper sem approval
    slot-risk-mismatch.yaml     ─ simula provider com risk_score errado
    discovery-failed.yaml       ─ simula falha do slot de discovery
    yaml-null-field.yaml        ─ simula active.yaml com field nulo
    side-quest-bypass.yaml      ─ simula move de arquivo sem mini approval gate
  run-tests.sh                  ─ executa fixtures e verifica saída esperada
```

Cada fixture define:
- `scenario`: nome do cenário
- `input`: estado de configuração simulado (active.yaml, roles config)
- `trigger`: ação que deve ser bloqueada
- `expected_event`: evento bloqueado esperado (ex: `[Strategist] phase=preflight status=blocked reason=slot_risk_mismatch`)

O `run-tests.sh` usa golden-file comparison: compara output do agente com o `expected_event` da fixture.

**Integração CI:**
```yaml
# .github/workflows/test.yml (novo)
on: [push, pull_request]
jobs:
  test:
    steps:
      - shellcheck bootstrap.sh strategist/install.sh   (já existe via release.yml, mover para cá)
      - bash strategist/tests/run-tests.sh
      - validate YAML schemas em .strategist/schemas/
```

**Arquivos afetados:**
- `strategist/tests/` (diretório NOVO)
- `.github/workflows/test.yml` (NOVO)

---

## Item 4 — Contrato de Interface entre Slots

### Decisão de design

Criar um schema `slot-output.schema.yaml` que define o que um slot provider DEVE retornar para que Strategist possa processar corretamente.

```yaml
# .strategist/schemas/slot-output.schema.yaml
discovery_slot:
  required_fields:
    - artifact_path   # path do arquivo escrito
    - status          # success | failed
  optional_fields:
    - partial_artifact_path   # presente quando status=failed mas escreveu algo

refinement_slot:
  required_fields:
    - artifact_dir    # path do diretório refined/<mission_id>/
    - status
    - files:
        - proposal.md
        - design.md
        - tasks.md    # pode ser vazio; se ausente → Sniper não invocado
```

Adicionar verificação deste schema em Strategist após cada slot retornar — antes de avançar para a próxima fase.

**Arquivos afetados:**
- `.strategist/schemas/slot-output.schema.yaml` (NOVO)
- `.strategist/SKILL.md`: adicionar step de validação de output após 5a (Ranger) e 5e (Archivist)
- `.strategist/schemas/` → incluir `slot-output.schema.yaml` em `index.yaml` sob `load_always`

---

## Item 5 — install.sh Wizard: Rollback

### Decisão de design

Introduzir um `INSTALL_MANIFEST` que rastreia cada arquivo e diretório criado. Em caso de falha (via `trap ERR`), um `rollback()` desfaz as escritas em ordem reversa.

```
install.sh
  ├─ INSTALL_MANIFEST=()          ─ array de paths criados
  ├─ trap 'rollback' ERR          ─ ativa rollback em qualquer erro
  │
  ├─ copy_skill_runtime()
  │    → registra cada arquivo copiado no MANIFEST
  │
  ├─ run_wizard() / run_silent()
  │    → registra cada escrita no MANIFEST
  │
  └─ rollback()
       → itera MANIFEST em ordem reversa
       → remove arquivos, rmdir dirs criados (apenas se vazios)
       → emite: [Strategist] WARN: install rolled back due to error
```

**Boundary ambígua identificada (confirmada):** `copy_skill_runtime` e `run_wizard` escrevem em dois destinos distintos — `${TARGET_REPO}/.strategist/` e `${HOME}/.claude/skills/strategist/`. O rollback precisa rastrear ambos. A função `install_agent_shims` escreve nos shims de agente — incluir no manifest.

**Arquivo afetado:**
- `strategist/install.sh`: adicionar MANIFEST, trap, e rollback function

---

## Dependency Map

```
Item 1 (checksum)
  → release.yml + bootstrap.sh
  → independente dos demais

Item 2 (YAML validation)
  → SKILL.md preflight + 2 schemas novos
  → independente dos demais

Item 3 (test harness)
  → strategist/tests/ + .github/workflows/test.yml
  → beneficia de Item 2 (fixture de null field testa o step novo)

Item 4 (slot output schema)
  → .strategist/schemas/slot-output.schema.yaml + SKILL.md
  → independente; pode ser incluído nas fixtures do Item 3

Item 5 (wizard rollback)
  → strategist/install.sh
  → independente dos demais
```

Itens podem ser implementados em qualquer ordem. Ordem recomendada: 1 → 5 → 2 → 4 → 3 (risco decrescente, testes por último para cobrir o que foi corrigido).
