# Docs Governance CI Gate Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a shell-based CI gate that enforces the four docs governance editorial policies automatically on every push.

**Architecture:** Three sequential tasks — write the shell script, wire it into `make`, add the CI step. The script follows the exact pattern of `scripts/check-refined-structure.sh`: accumulate all violations before exiting, exit 1 if any found. No new dependencies.

**Tech Stack:** POSIX sh, `find`, `grep`. Existing: `Makefile`, `.github/workflows/test.yml`.

---

## Task 1: Write `scripts/check-docs-governance.sh`

**Files:**
- Create: `scripts/check-docs-governance.sh`

Reference implementation to follow: `scripts/check-refined-structure.sh` — same `set -eu`, same heredoc-while pattern, same violation accumulator, same single `OK:` line on success.

### Step 1: Create the script file

Create `scripts/check-docs-governance.sh` with the following exact content:

```sh
#!/usr/bin/env sh
set -eu

DOCS="docs"
ADR_DIR="$DOCS/adr"
README="$DOCS/README.md"
violations=0

# --- Check 7: docs/README.md must exist ---
if [ ! -f "$README" ]; then
  echo "FAIL: $README does not exist"
  violations=1
fi

# --- Checks 1-2: every docs/adr/*.md must have Status and Date/Data ---
while IFS= read -r f; do
  [ -n "$f" ] || continue
  if ! grep -qE "^\*\*Status:\*\*" "$f" 2>/dev/null; then
    echo "FAIL: $f missing **Status:** field"
    violations=1
  fi
  if ! grep -qE "^\*\*Date:\*\*|^\*\*Data:\*\*" "$f" 2>/dev/null; then
    echo "FAIL: $f missing **Date:** or **Data:** field"
    violations=1
  fi
done <<EOF
$(find "$ADR_DIR" -maxdepth 1 -name '*.md' 2>/dev/null | sort)
EOF

# --- Checks 3-4: every docs/*.md (excl. plans/) must have Status and Date/Last Updated ---
while IFS= read -r f; do
  [ -n "$f" ] || continue
  if ! grep -qE "^\*\*Status:\*\*" "$f" 2>/dev/null; then
    echo "FAIL: $f missing **Status:** field"
    violations=1
  fi
  if ! grep -qE "^\*\*(Date|Last Updated):\*\*" "$f" 2>/dev/null; then
    echo "FAIL: $f missing **Date:** or **Last Updated:** field"
    violations=1
  fi
done <<EOF
$(find "$DOCS" -maxdepth 1 -name '*.md' 2>/dev/null | sort)
EOF

# --- Check 5: no ../ refs in docs/*.md (excl. plans/) or docs/adr/*.md ---
while IFS= read -r f; do
  [ -n "$f" ] || continue
  echo "FAIL: $f contains external cross-reference (../)"
  violations=1
done <<EOF
$(find "$DOCS" -maxdepth 1 -name '*.md' -exec grep -l "\.\.\/" {} \; 2>/dev/null | sort
find "$ADR_DIR" -maxdepth 1 -name '*.md' -exec grep -l "\.\.\/" {} \; 2>/dev/null | sort)
EOF

# --- Check 6: no placeholders in docs/*.md (excl. plans/) or docs/adr/*.md ---
while IFS= read -r line; do
  [ -n "$line" ] || continue
  echo "FAIL: placeholder found: $line"
  violations=1
done <<EOF
$(find "$DOCS" -maxdepth 1 -name '*.md' -exec grep -inE "\bTBD\b|\bWIP\b|\[TBD\]|\bTODO\b" {} /dev/null \; 2>/dev/null
find "$ADR_DIR" -maxdepth 1 -name '*.md' -exec grep -inE "\bTBD\b|\bWIP\b|\[TBD\]|\bTODO\b" {} /dev/null \; 2>/dev/null)
EOF

# --- Checks 8-10: README navigation and language policy ---
if [ -f "$README" ]; then
  # Check 9: docs/adr/ directory referenced in README
  if ! grep -q "adr/" "$README"; then
    echo "FAIL: $README does not reference docs/adr/"
    violations=1
  fi

  # Check 10: language policy declared in README
  if ! grep -qiE "language|idioma" "$README"; then
    echo "FAIL: $README does not declare language policy (missing 'language' or 'idioma')"
    violations=1
  fi

  # Check 8: every docs/*.md (excl. README.md) is referenced in README
  while IFS= read -r f; do
    [ -n "$f" ] || continue
    base=$(basename "$f")
    [ "$base" = "README.md" ] && continue
    if ! grep -q "$base" "$README" 2>/dev/null; then
      echo "FAIL: $f not referenced in $README"
      violations=1
    fi
  done <<EOF
$(find "$DOCS" -maxdepth 1 -name '*.md' 2>/dev/null | sort)
EOF
fi

if [ "$violations" -ne 0 ]; then
  exit 1
fi

echo "OK: docs governance valid"
```

### Step 2: Make it executable

```bash
chmod +x scripts/check-docs-governance.sh
```

### Step 3: Run the script manually to verify it passes on current state

```bash
bash scripts/check-docs-governance.sh
```

Expected output:
```
OK: docs governance valid
```

If it fails, investigate the violation messages and fix before continuing. The current `docs/` state (after Spec A implementation) should pass all checks.

### Step 4: Verify a deliberate violation is caught

Temporarily add a bad field to a docs file, confirm the gate catches it, then revert:

```bash
# Introduce a violation
echo "TBD" >> docs/architecture.md

# Should fail
bash scripts/check-docs-governance.sh
# Expected: FAIL: docs/architecture.md: contains placeholder

# Revert
git checkout docs/architecture.md

# Should pass again
bash scripts/check-docs-governance.sh
# Expected: OK: docs governance valid
```

---

## Task 2: Add `docs-governance-gate` to Makefile

**Files:**
- Modify: `Makefile:1` (`.PHONY` line)
- Modify: `Makefile:74-76` (after `analysis-structure-gate` target)

### Step 1: Add to `.PHONY` line

In [Makefile:1](Makefile), the `.PHONY` line currently ends with `... analysis-structure-gate install sync-embed release snapshot clean`.

Change:
```
.PHONY: build test test-all integration spec test-lite test-telemetry-lite test-compile-cache test-domain-architecture lint vuln bench cover cover-gate cover-html analysis-structure-gate install sync-embed release snapshot clean
```

To:
```
.PHONY: build test test-all integration spec test-lite test-telemetry-lite test-compile-cache test-domain-architecture lint vuln bench cover cover-gate cover-html analysis-structure-gate docs-governance-gate install sync-embed release snapshot clean
```

### Step 2: Add the target

In [Makefile:75-76](Makefile), after the `analysis-structure-gate` target block, insert:

```makefile
docs-governance-gate:
	bash scripts/check-docs-governance.sh
```

The result should look like:

```makefile
analysis-structure-gate:
	bash scripts/check-refined-structure.sh

docs-governance-gate:
	bash scripts/check-docs-governance.sh

install: build
```

> Note: the indentation before `bash` must be a TAB character, not spaces. Makefile requires tabs.

### Step 3: Verify the make target works

```bash
make docs-governance-gate
```

Expected output:
```
bash scripts/check-docs-governance.sh
OK: docs governance valid
```

---

## Task 3: Add CI step to `.github/workflows/test.yml`

**Files:**
- Modify: `.github/workflows/test.yml:78-79` (after `Analysis refined structure gate` step)

### Step 1: Add the step

In [.github/workflows/test.yml:79](`.github/workflows/test.yml`), after:

```yaml
      - name: Analysis refined structure gate
        run: make analysis-structure-gate
```

Insert:

```yaml
      - name: Docs governance gate
        run: make docs-governance-gate
```

The result should look like:

```yaml
      - name: Analysis refined structure gate
        run: make analysis-structure-gate

      - name: Docs governance gate
        run: make docs-governance-gate
```

### Step 2: Verify the YAML is valid

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/test.yml'))" && echo "YAML valid"
```

Expected: `YAML valid`

### Step 3: Run a full local simulation

```bash
make analysis-structure-gate && make docs-governance-gate
```

Expected: both targets print `OK:` and exit 0.

---

## Verification Sweep (post-all-tasks)

```bash
# Script exists and is executable
ls -la scripts/check-docs-governance.sh

# Make target works
make docs-governance-gate

# YAML valid
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/test.yml'))" && echo "YAML valid"

# Docs governance gate appears in PHONY
grep "docs-governance-gate" Makefile

# CI step present
grep "Docs governance gate" .github/workflows/test.yml
```

Expected: all commands succeed, no error output.
