# Tasks: Role Standardization — Ranger / Archivist / Sniper
**Date:** 2026-05-28
**Scope:** external — writes to strategist/ outside base_path
**Apply before:** `.analysis/refined/2026-05-28-engineer-openspec-output/` (overlapping SKILL.md §5e/§6)

---

## T1 — SKILL.md: add contract table to intro + bulk rename

**File:** `strategist/SKILL.md`

### T1a — Replace intro line (line 4)

Find:
```
three pluggable slots: Scout (discovery) → Engineer (refinement) → Hunter (execution).
You do not perform discovery, refinement, or execution yourself — you delegate.
```

Replace:
```
three pluggable slots: Ranger (discovery) → Archivist (refinement) → Sniper (execution).
You do not perform discovery, refinement, or execution yourself — you delegate.

| Internal name | Slot key   | Contract       | Progress label |
|---------------|------------|----------------|----------------|
| Ranger        | discovery  | write_pending  | discovery      |
| Archivist     | refinement | write_analysis | refinement     |
| Sniper        | execution  | controlled     | execution      |
```

### T1b — Bulk rename (replace_all across entire file)

Apply in this order to avoid partial matches:

1. `scout_label` → `ranger_label`
2. `engineer_label` → `archivist_label`
3. `hunter_label` → `sniper_label`
4. `scout_failed` → `ranger_failed`
5. `Scout (discovery slot)` → `Ranger (discovery slot)`
6. `Engineer (refinement slot)` → `Archivist (refinement slot)`
7. `Hunter (execution slot)` → `Sniper (execution slot)`
8. `Scout` → `Ranger` (replace_all remaining)
9. `Engineer` → `Archivist` (replace_all remaining)
10. `Hunter` → `Sniper` (replace_all remaining)

**Do NOT rename:**
- `discovery slot` / `refinement slot` / `execution slot` (external vocabulary)
- `execution_provider` (sdd_injection field name)
- `write_pending` / `write_analysis` / `controlled`

---

## T2 — skill.yaml: forbidden_behaviors + inline prose

**File:** `strategist/skill.yaml`

### T2a — Description prose (replace_all)

| Find | Replace |
|------|---------|
| `before Hunter moves files.` | `before Sniper moves files.` |
| `log and continue to Engineer` | `log and continue to Archivist` |

### T2b — forbidden_behaviors (replace_all)

| Find | Replace |
|------|---------|
| `scout_writes_outside_pending` | `ranger_writes_outside_pending` |
| `engineer_writes_non_md` | `archivist_writes_non_md` |
| `invoke_side_quest_hunter_without_approval` | `invoke_side_quest_sniper_without_approval` |

---

## T3 — personas/epic.yaml: phase_labels + approval prose

**File:** `strategist/personas/epic.yaml`

Find:
```yaml
phase_labels:
  discovery: scout
  refinement: engineer
  execution: hunter
```
Replace:
```yaml
phase_labels:
  discovery: Ranger
  refinement: Archivist
  execution: Sniper
```

Then replace_all in prose:

| Find | Replace |
|------|---------|
| `Engineer briefing complete.` | `Archivist briefing complete.` |
| `Authorize Hunter deployment?` | `Authorize Sniper deployment?` |
| `Mission halted at Hunter authorization.` | `Mission halted at Sniper authorization.` |
| `To deploy Hunter:` | `To deploy Sniper:` |

---

## T4 — install.sh: variable rename + bug fix

**File:** `strategist/install.sh`

### T4a — Variable and validate_provider calls (replace_all in wizard section)

| Find | Replace |
|------|---------|
| `read -r scout` | `read -r ranger` |
| `scout="${scout:-brainstorming}"` | `ranger="${ranger:-brainstorming}"` |
| `validate_provider "$scout" "write_pending" "Scout"` | `validate_provider "$ranger" "write_pending" "Ranger"` |
| `read -r engineer` | `read -r archivist` |
| `engineer="${engineer:-openspec-explore}"` | `archivist="${archivist:-openspec-explore}"` |
| `validate_provider "$engineer" "write_analysis" "Engineer"` | `validate_provider "$archivist" "write_analysis" "Archivist"` |
| `read -r hunter` | `read -r sniper` |
| `hunter="${hunter:-sdd-ask}"` | `sniper="${sniper:-sdd-ask}"` |
| `validate_provider "$hunter" "controlled" "Hunter"` | `validate_provider "$sniper" "controlled" "Sniper"` |
| `[ -n "$hunter" ]` | `[ -n "$sniper" ]` |
| `"Error: Hunter provider is required."` | `"Error: Sniper provider is required."` |

### T4b — Prompt labels (replace_all)

| Find | Replace |
|------|---------|
| `# Scout provider` | `# Ranger (discovery) provider` |
| `Scout: descobre o espaço do problema` | `Ranger: descobre o espaço do problema` |
| `"Scout provider (write_pending)` | `"Ranger provider (discovery, write_pending)` |
| `# Engineer provider` | `# Archivist (refinement) provider` |
| `Engineer: refina a descoberta` | `Archivist: refina a descoberta` |
| `"Engineer provider (refinement, write_analysis)` | `"Archivist provider (refinement, write_analysis)` |
| `# Hunter provider` | `# Sniper (execution) provider` |
| `Hunter: executa o plano refinado` | `Sniper: executa o plano refinado` |
| `"Hunter provider (execution, controlled)` | `"Sniper provider (execution, controlled)` |

### T4c — Bug fix: roles YAML write (lines ~231–233)

Find:
```bash
scout: ${scout}
engineer: ${engineer}
hunter: ${hunter}
```
Replace:
```bash
discovery: ${ranger}
refinement: ${archivist}
execution: ${sniper}
```

---

## T5 — skills/engineer/ → skills/archivist/ (full rename)

### T5a — Create new directory and copy files

```bash
mkdir strategist/skills/archivist
cp strategist/skills/engineer/skill.yaml strategist/skills/archivist/skill.yaml
cp strategist/skills/engineer/SKILL.md strategist/skills/archivist/SKILL.md
```

### T5b — Update strategist/skills/archivist/skill.yaml

| Find | Replace |
|------|---------|
| `id: engineer` | `id: archivist` |
| `path to Scout's output` | `path to Ranger's output` |
| `handoff to Hunter` | `handoff to Sniper` |

### T5c — Update strategist/skills/archivist/SKILL.md

| Find | Replace |
|------|---------|
| `# engineer — Agent Instructions` | `# archivist — Agent Instructions` |
| `You are engineer,` | `You are archivist,` |
| `Hunter` | `Sniper` (replace_all) |
| `Scout` | `Ranger` (replace_all) |

### T5d — Delete old directory

```bash
rm strategist/skills/engineer/skill.yaml
rm strategist/skills/engineer/SKILL.md
rmdir strategist/skills/engineer
```

---

## T6 — roles/default.yaml: update binding + comment

**File:** `strategist/roles/default.yaml`

Find:
```yaml
# Slot keys: discovery, refinement, execution (internal names).
```
Replace:
```yaml
# Slot keys: discovery, refinement, execution (external names — Ranger, Archivist, Sniper internally).
```

Find:
```yaml
refinement: engineer
```
Replace:
```yaml
refinement: archivist
```

---

## T7 — schemas/progress-contract.yaml: note + examples

**File:** `strategist/schemas/progress-contract.yaml`

Find:
```
    epic: scout / engineer / hunter
    Internal slots use generic names (discovery/refinement/execution) in code;
    all user-facing events use the persona label.
```
Replace:
```
    epic: Ranger / Archivist / Sniper
    Internal role names (Ranger/Archivist/Sniper) appear in SKILL.md prose and persona labels;
    external slot keys (discovery/refinement/execution) appear in roles YAML and progress events.
```

Then update event examples (replace_all):

| Find | Replace |
|------|---------|
| `phase=scout status=running skill=sdd-diagnose checklist=0/3` | `phase=discovery status=running skill=sdd-diagnose checklist=0/3` |
| `phase=scout status=done artifact=` | `phase=discovery status=done artifact=` |
| `phase=engineer status=running skill=engineer checklist=1/3` | `phase=refinement status=running skill=archivist checklist=1/3` |
| `phase=hunter status=done artifact=` | `phase=execution status=done artifact=` |

---

## T8 — templates/domain/identity/what-i-am.yaml: prose

**File:** `strategist/templates/domain/identity/what-i-am.yaml`

Replace_all:

| Find | Replace |
|------|---------|
| `before every Hunter invocation.` | `before every Sniper invocation.` |
| `root-cause analysis; Scout does.` | `root-cause analysis; Ranger does.` |
| `produce plans; Engineer does.` | `produce plans; Archivist does.` |
| `apply changes; Hunter does.` | `apply changes; Sniper does.` |
| `invoke Hunter without` | `invoke Sniper without` |

---

## Verification

After all tasks complete:

```bash
# 1. No old internal names remain in prose files
grep -r "Scout\|Engineer\|Hunter" strategist/SKILL.md strategist/skill.yaml \
  strategist/personas/ strategist/install.sh \
  strategist/templates/domain/identity/what-i-am.yaml

# 2. skills/archivist exists and skills/engineer is gone
ls strategist/skills/

# 3. roles/default.yaml uses archivist
grep "refinement" strategist/roles/default.yaml

# 4. install.sh writes correct keys
grep -A3 "discovery:\|refinement:\|execution:" strategist/install.sh | head -10

# 5. progress-contract examples use external labels
grep "phase=discovery\|phase=refinement\|phase=execution" \
  strategist/schemas/progress-contract.yaml
```

Expected: all greps return matches; `ls strategist/skills/` shows `archivist` but not `engineer`.
