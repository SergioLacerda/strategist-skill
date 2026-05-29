# Wizard Provider Validation — Implementation Plan
**Date:** 2026-05-28
**Status:** pending implementation — apenas §4 restante
**Spec:** `.analysis/done/2026-05-28-slot-risk-contract-design.md`

---

## Contexto

| Seção | Descrição | Status |
|-------|-----------|--------|
| §1 — skill.yaml vocabulary | `controlled_write` → `controlled` | ✅ `99789a4` |
| §2 — templates/known-providers.yaml | Template para install copiar | ✅ `8db4a4c` |
| §3 — SKILL.md preflight resolution order | Ordem de resolução com known-providers.yaml | ✅ `8db4a4c` |
| §4 — install.sh wizard validation | `validate_provider()` no wizard | ❌ pendente |

**Apenas §4 permanece.** Este plano cobre exclusivamente a implementação do wizard.

---

## Tarefa — `strategist/install.sh` — `validate_provider()`

**Arquivo:** `strategist/install.sh`

Adicionar função helper `validate_provider()` antes de `run_wizard()` e chamá-la
nas três prompts de configuração de provider.

### Contratos esperados por slot (valores atuais)

| Slot | Contrato requerido | Provider padrão |
|------|--------------------|----------------|
| Scout | `write_pending` | brainstorming |
| Engineer | `write_analysis` | openspec-explore |
| Hunter | `controlled` | sdd-ask |

### Função `validate_provider()`

Adicionar antes de `run_wizard()`:

```bash
# validate_provider <name> <required_risk> <slot_label>
# Returns 0 if valid, 1 if mismatch or not found.
validate_provider() {
  local name="$1" required="$2" label="$3"
  local score=""

  # 1. Check ~/.claude/skills/<name>/skill.yaml
  local skill_yaml="${HOME}/.claude/skills/${name}/skill.yaml"
  if [ -f "$skill_yaml" ]; then
    score=$(grep -m1 '^risk_score:' "$skill_yaml" | awk '{print $2}')
  fi

  # 2. Fall back to templates/known-providers.yaml
  if [ -z "$score" ]; then
    local known="${SKILL_ROOT}/templates/known-providers.yaml"
    if [ -f "$known" ]; then
      score=$(grep -m1 "^  ${name}:" "$known" | awk '{print $2}')
    fi
  fi

  # 3. Not found — prompt user
  if [ -z "$score" ]; then
    echo "  ⚠ risk_score for '${name}' not declared."
    printf "  Enter value [write_pending/write_analysis/controlled/orchestrator]: "
    read -r score
  fi

  # 4. Compare
  if [ "$score" != "$required" ]; then
    echo "  ✗ Slot ${label} requires '${required}', but '${name}' declares '${score}'."
    return 1
  fi

  echo "  ✓ ${name} → ${score}"
  return 0
}
```

### Wizard prompts atualizados

Substituir os três blocos de leitura de provider em `run_wizard()` por:

```bash
# Scout
while true; do
  printf "Scout provider (discovery, write_pending) [brainstorming]: "
  read -r scout_provider
  scout_provider="${scout_provider:-brainstorming}"
  validate_provider "$scout_provider" "write_pending" "Scout" && break
  echo "  Enter a different provider name."
done

# Engineer
while true; do
  printf "Engineer provider (refinement, write_analysis) [openspec-explore]: "
  read -r engineer_provider
  engineer_provider="${engineer_provider:-openspec-explore}"
  validate_provider "$engineer_provider" "write_analysis" "Engineer" && break
  echo "  Enter a different provider name."
done

# Hunter
while true; do
  printf "Hunter provider (execution, controlled) [sdd-ask]: "
  read -r hunter_provider
  hunter_provider="${hunter_provider:-sdd-ask}"
  validate_provider "$hunter_provider" "controlled" "Hunter" && break
  echo "  Enter a different provider name."
done
```

---

## Localizar ponto de inserção no install.sh

```bash
# validate_provider() → inserir imediatamente antes da linha:
run_wizard() {
```

```bash
# Wizard prompts → localizar e substituir os três read -r *_provider dentro de run_wizard()
# Atualmente são leituras simples sem loop ou validação.
```

---

## Critério de done

- `bash strategist/install.sh --wizard` com provider de risco errado → mensagem de erro + re-prompt
- `bash strategist/install.sh --wizard` com provider válido → aceita sem aviso
- Provider desconhecido → prompt para informar o valor manualmente
- Defaults (`brainstorming`, `openspec-explore`, `sdd-ask`) passam sem prompt extra
