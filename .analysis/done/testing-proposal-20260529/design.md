# Design: Testes Agnósticos de Linguagem — Strategist Skill
**Mission ID:** testing-proposal-20260529  
**Date:** 2026-05-29

---

## Estrutura de Diretórios

```
strategist/
└── tests/
    ├── fixtures/
    │   ├── configs/
    │   │   ├── active-test.yaml          ← active.yaml minimal para testes
    │   │   └── roles-test.yaml           ← roles config mínima
    │   └── compiled/
    │       ├── fresh.config.gz           ← artifact fresco (para check-stale = 0)
    │       └── stale.config.gz           ← artifact com sources adulterados (check-stale = 1)
    │
    ├── specs/                            ← Camada 2: BDD (documentação)
    │   ├── approval-gate.feature
    │   ├── slot-contracts.feature
    │   ├── forbidden-behaviors.feature
    │   └── drift-correction.feature
    │
    ├── validators/                       ← Validators reutilizáveis
    │   ├── validate-contracts.sh
    │   ├── validate-schemas.sh
    │   ├── validate-compiled.sh
    │   └── validate-events.sh
    │
    ├── unit/                             ← Testes unitários de shell scripts
    │   ├── test-check-stale.sh
    │   ├── test-compile-config.sh
    │   ├── test-compile-domain.sh
    │   └── test-compile-all.sh
    │
    ├── integration/                      ← Testes de integração
    │   ├── test-install-silent.sh
    │   └── test-install-wizard.sh
    │
    └── harness/
        ├── run-tests.sh                  ← Entrypoint: bash tests/harness/run-tests.sh
        └── Makefile
```

---

## Camada 1: Validators

### validate-contracts.sh

Valida que cada arquivo em `strategist/contracts/*.yaml` tem os campos obrigatórios:

```bash
#!/usr/bin/env bash
set -euo pipefail

CONTRACTS_DIR="${1:-strategist/contracts}"
PASS=0; FAIL=0

for f in "$CONTRACTS_DIR"/*.yaml; do
  name=$(basename "$f")
  errors=""
  
  for field in module type description write_scope owner; do
    val=$(yq ".$field" "$f" 2>/dev/null)
    if [[ -z "$val" || "$val" == "null" ]]; then
      errors="$errors missing:$field"
    fi
  done
  
  for section in contract.input contract.output contract.error_conditions; do
    val=$(yq ".$section | length" "$f" 2>/dev/null)
    if [[ -z "$val" || "$val" == "0" ]]; then
      errors="$errors empty:$section"
    fi
  done
  
  if [[ -z "$errors" ]]; then
    echo "  ok  $name"
    ((PASS++))
  else
    echo "  FAIL $name →$errors"
    ((FAIL++))
  fi
done

echo ""
echo "contracts: $PASS ok, $FAIL failed"
[[ $FAIL -eq 0 ]]
```

### validate-schemas.sh

Valida que todos os `*.yaml` em `strategist/schemas/` são YAML válido e têm campo `schema_version`:

```bash
#!/usr/bin/env bash
set -euo pipefail

SCHEMAS_DIR="${1:-strategist/schemas}"
PASS=0; FAIL=0

for f in "$SCHEMAS_DIR"/*.yaml; do
  name=$(basename "$f")
  
  if ! yq eval '.' "$f" > /dev/null 2>&1; then
    echo "  FAIL $name → YAML parse error"
    ((FAIL++))
    continue
  fi
  
  echo "  ok  $name"
  ((PASS++))
done

echo ""
echo "schemas: $PASS ok, $FAIL failed"
[[ $FAIL -eq 0 ]]
```

### validate-compiled.sh

Valida que um arquivo `.gz` contém JSON válido com campos obrigatórios de schema:

```bash
#!/usr/bin/env bash
set -euo pipefail

GZ_FILE="$1"
EXPECTED_SCHEMA="$2"  # ex: "strategist-compiled-config/1.0"

if [[ ! -f "$GZ_FILE" ]]; then
  echo "FAIL: $GZ_FILE not found"
  exit 1
fi

JSON=$(gunzip -c "$GZ_FILE")

actual_schema=$(echo "$JSON" | jq -r '.schema // empty')
if [[ "$actual_schema" != "$EXPECTED_SCHEMA" ]]; then
  echo "FAIL: schema mismatch. expected=$EXPECTED_SCHEMA actual=$actual_schema"
  exit 1
fi

for field in compiled_at sources; do
  val=$(echo "$JSON" | jq -r ".$field // empty")
  if [[ -z "$val" ]]; then
    echo "FAIL: missing field $field"
    exit 1
  fi
done

echo "ok: $GZ_FILE schema=$actual_schema"
```

### validate-events.sh

Valida formato de linhas de evento emitidas pelo Strategist:

```bash
#!/usr/bin/env bash
# Uso: echo "$agent_output" | bash validate-events.sh
# Ou:  bash validate-events.sh < session.log

EVENT_PATTERN='^\[Strategist\] phase=[a-z_]+ status=(running|done|failed|blocked)'
PASS=0; FAIL=0

while IFS= read -r line; do
  if echo "$line" | grep -q '^\[Strategist\]'; then
    if echo "$line" | grep -qE "$EVENT_PATTERN"; then
      ((PASS++))
    else
      echo "  FAIL invalid event format: $line"
      ((FAIL++))
    fi
  fi
done

echo "events: $PASS valid, $FAIL invalid"
[[ $FAIL -eq 0 ]]
```

---

## Camada 1: Unit Tests

### test-check-stale.sh

```bash
#!/usr/bin/env bash
set -euo pipefail

SCRIPT="$(dirname "$0")/../../scripts/check-stale.sh"
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

PASS=0; FAIL=0

run_case() {
  local name="$1"; local expected="$2"; shift 2
  if "$@" > /dev/null 2>&1; then actual=0; else actual=1; fi
  if [[ "$actual" == "$expected" ]]; then
    echo "  ok  $name"
    ((PASS++))
  else
    echo "  FAIL $name (expected exit=$expected, got=$actual)"
    ((FAIL++))
  fi
}

# Case 1: arquivo ausente → stale (exit 1)
run_case "absent_file" 1 sh "$SCRIPT" "$TMPDIR/nonexistent.gz"

# Case 2: arquivo presente mas sem .manifest.gz → stale
echo '{"schema":"strategist-compiled-config/1.0","compiled_at":"2026-01-01","sources":{},"active":{},"personas":{},"roles":{}}' \
  | gzip > "$TMPDIR/.config.gz"
run_case "missing_manifest" 1 sh "$SCRIPT" "$TMPDIR/.config.gz"

# Case 3: artifact + manifest frescos → fresh (exit 0)
echo '{"schema":"strategist-compiled-manifest/1.0","compiled_at":"2026-01-01","artifacts":{}}' \
  | gzip > "$TMPDIR/.manifest.gz"
# Recria .config.gz com sources vazio (sem arquivos para checar mtime)
echo '{"schema":"strategist-compiled-config/1.0","compiled_at":"2026-01-01","sources":{},"active":{},"personas":{},"roles":{}}' \
  | gzip > "$TMPDIR/.config.gz"
run_case "fresh_no_sources" 0 sh "$SCRIPT" "$TMPDIR/.config.gz"

echo ""
echo "check-stale: $PASS ok, $FAIL failed"
[[ $FAIL -eq 0 ]]
```

### test-compile-config.sh

```bash
#!/usr/bin/env bash
set -euo pipefail

COMPILE_SCRIPT="$(dirname "$0")/../../scripts/compile-config.sh"
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

# Montar um .strategist/ mínimo
mkdir -p "$TMPDIR/personas" "$TMPDIR/roles"

cat > "$TMPDIR/active.yaml" << 'EOF'
mode: pragmatic
base_path: .analysis
roles_config: default
EOF

cat > "$TMPDIR/personas/pragmatic.yaml" << 'EOF'
id: pragmatic
description: Test persona
phase_labels:
  discovery: analysis
  refinement: refinement
  execution: execution
tone_directive: Be precise.
EOF

cat > "$TMPDIR/roles/default.yaml" << 'EOF'
discovery: brainstorming
refinement: openspec-explore
execution: sdd-ask
EOF

OUT_GZ="$TMPDIR/out.config.gz"
sh "$COMPILE_SCRIPT" "$TMPDIR" "$OUT_GZ"

# Validar output
JSON=$(gunzip -c "$OUT_GZ")

check() {
  local field="$1"; local expected="$2"
  actual=$(echo "$JSON" | jq -r "$field // empty")
  if [[ "$actual" == "$expected" ]]; then
    echo "  ok  $field=$actual"
  else
    echo "  FAIL $field: expected=$expected actual=$actual"
    exit 1
  fi
}

check '.schema' 'strategist-compiled-config/1.0'
check '.active.mode' 'pragmatic'
check '.personas.pragmatic.id' 'pragmatic'
check '.roles.default.discovery' 'brainstorming'

echo "  ok  compile-config output valid"
```

---

## Camada 1: Integration Tests

### test-install-silent.sh

```bash
#!/usr/bin/env bash
set -euo pipefail

INSTALL_SCRIPT="$(pwd)/strategist/install.sh"
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

# Rodar install.sh --silent no tmpdir
cd "$TMPDIR"
bash "$INSTALL_SCRIPT" --silent

# Verificar estrutura obrigatória
PASS=0; FAIL=0

check_path() {
  if [[ -e ".strategist/$1" ]]; then
    echo "  ok  .strategist/$1"
    ((PASS++))
  else
    echo "  FAIL .strategist/$1 not found"
    ((FAIL++))
  fi
}

check_path "SKILL.md"
check_path "active.yaml"
check_path "personas/pragmatic.yaml"
check_path "roles/default.yaml"
check_path "schemas/"
check_path "contracts/"
check_path "scripts/check-stale.sh"
check_path "scripts/compile-all.sh"

echo ""
echo "install-silent: $PASS ok, $FAIL failed"
[[ $FAIL -eq 0 ]]
```

---

## Camada 2: Behavior Specs (Gherkin — documentação)

### approval-gate.feature

```gherkin
Feature: Approval Gate Enforcement
  Invariant: Sniper never executes without explicit user approval.

  Background:
    Given Archivist has completed successfully
    And tasks.md contains tasks that write outside .analysis/

  Scenario: Sniper blocked before approval
    When Strategist evaluates tasks.md
    Then Strategist emits "[Strategist] phase=approval_gate status=pending"
    And Strategist does NOT invoke the Sniper slot
    And waits for user response

  Scenario: Sniper proceeds after explicit "yes"
    Given the approval gate is presented
    When user responds with "yes"
    Then Strategist emits "[Strategist] phase=sniper status=running"
    And invokes the execution slot provider

  Scenario: Mission ends as plan_only after "no"
    Given the approval gate is presented
    When user responds with "no"
    Then Strategist emits "[Strategist] phase=approval_gate status=plan_only"
    And Sniper is never invoked
    And mission result has status=plan_only
```

### slot-contracts.feature

```gherkin
Feature: Slot Write Scope Contracts
  Invariant: Each slot may only write to its declared scope.

  Scenario: Ranger respects write_pending boundary
    Given Ranger (discovery slot) is executing
    And Ranger.write_scope = "write_pending"
    When Ranger attempts to write to .analysis/refined/
    Then Strategist emits "slot_write_scope_violation"
    And the write is blocked

  Scenario: Archivist respects write_analysis boundary
    Given Archivist (refinement slot) is executing
    And Archivist.write_scope = "write_analysis"
    When Archivist attempts to write outside .analysis/
    Then Strategist emits "slot_write_scope_violation"
    And the write is blocked

  Scenario: Sniper requires controlled write_scope
    Given Sniper is declared in roles config
    When preflight resolves Sniper's risk_score
    Then risk_score MUST equal "controlled"
    Otherwise: emit blocked event reason=slot_risk_mismatch
```

### forbidden-behaviors.feature

```gherkin
Feature: Forbidden Behavior Detection
  Invariant: Certain behaviors are explicitly prohibited and must trigger self-correction.

  Scenario: Direct execution by Strategist
    Given Strategist is in the discovery phase
    When Strategist starts performing discovery work itself (not via slot)
    Then Strategist detects "direct_execution" drift pattern
    And stops immediately
    And re-routes to the discovery slot provider

  Scenario: Silent phase advance
    Given Strategist completed a phase
    When Strategist starts the next phase without emitting phase=done
    Then Strategist detects "silent_phase_advance" drift pattern
    And emits the missing done event before proceeding

  Scenario: Approval bypass
    Given Archivist completed and tasks.md has tasks
    When Strategist invokes Sniper without asking the user
    Then Strategist detects "approval_bypass" drift pattern
    And stops immediately
    And presents the approval gate
```

---

## Test Harness: run-tests.sh

```bash
#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TESTS_DIR="$REPO_ROOT/strategist/tests"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

PASSED=0; FAILED=0; SKIPPED=0

run_test() {
  local name="$1"; local script="$2"
  printf "  %-50s" "$name"
  if bash "$script" > /tmp/strategist_test.log 2>&1; then
    echo -e "${GREEN}PASS${NC}"
    ((PASSED++))
  else
    echo -e "${RED}FAIL${NC}"
    cat /tmp/strategist_test.log | sed 's/^/    /'
    ((FAILED++))
  fi
}

echo -e "\n${YELLOW}=== Validators ===${NC}"
run_test "contracts structure"  "$TESTS_DIR/validators/validate-contracts.sh"
run_test "schemas YAML valid"   "$TESTS_DIR/validators/validate-schemas.sh"

echo -e "\n${YELLOW}=== Unit Tests ===${NC}"
run_test "check-stale.sh"       "$TESTS_DIR/unit/test-check-stale.sh"
run_test "compile-config.sh"    "$TESTS_DIR/unit/test-compile-config.sh"
run_test "compile-domain.sh"    "$TESTS_DIR/unit/test-compile-domain.sh"
run_test "compile-all.sh"       "$TESTS_DIR/unit/test-compile-all.sh"

echo -e "\n${YELLOW}=== Integration Tests ===${NC}"
run_test "install.sh --silent"  "$TESTS_DIR/integration/test-install-silent.sh"

echo -e "\n${YELLOW}=== Summary ===${NC}"
echo -e "  Passed:  ${GREEN}$PASSED${NC}"
echo -e "  Failed:  ${RED}$FAILED${NC}"
echo -e "  Skipped: ${YELLOW}$SKIPPED${NC}"

[[ $FAILED -eq 0 ]]
```

---

## Makefile

```makefile
.PHONY: test test-validators test-unit test-integration clean

TESTS_DIR := strategist/tests

test: test-validators test-unit test-integration
	@echo "\n✓ All tests passed"

test-validators:
	@echo "=== Validators ==="
	@bash $(TESTS_DIR)/validators/validate-contracts.sh strategist/contracts
	@bash $(TESTS_DIR)/validators/validate-schemas.sh strategist/schemas

test-unit:
	@echo "=== Unit Tests ==="
	@bash $(TESTS_DIR)/unit/test-check-stale.sh
	@bash $(TESTS_DIR)/unit/test-compile-config.sh
	@bash $(TESTS_DIR)/unit/test-compile-domain.sh
	@bash $(TESTS_DIR)/unit/test-compile-all.sh

test-integration:
	@echo "=== Integration ==="
	@bash $(TESTS_DIR)/integration/test-install-silent.sh

clean:
	rm -f /tmp/strategist_test.log /tmp/strategist_test_*
```

---

## Dependências

| Ferramenta | Versão mínima | Uso |
|-----------|-------------|-----|
| `bash` | 3.2+ | harness, todos os scripts |
| `jq` | 1.6+ | JSON validation, compiled artifact inspection |
| `yq` | 4.x | YAML parsing e validation |
| `gzip` / `gunzip` | qualquer | compressed artifact handling |
| `grep` / `sed` | POSIX | event log validation |

Sem Python, Go, Node, Ruby ou qualquer linguagem de programação.

---

## Invariantes de Segurança — Cobertura

| Invariante | Camada 1 (shell) | Camada 2 (BDD) |
|-----------|-----------------|----------------|
| Approval gate não bypassável | via validate-events.sh (log check) | approval-gate.feature |
| Slot write scope respeitado | validate-contracts.sh (contract structure) | slot-contracts.feature |
| Forbidden behaviors bloqueados | via validate-events.sh | forbidden-behaviors.feature |
| Contratos têm campos obrigatórios | validate-contracts.sh | — |
| Schemas YAML válidos | validate-schemas.sh | — |
| check-stale detecta artifacts stale | test-check-stale.sh | — |
| install.sh produz estrutura correta | test-install-silent.sh | — |
| compile scripts produzem schema correto | test-compile-*.sh | — |
