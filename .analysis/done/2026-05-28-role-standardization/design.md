# Design: Role Standardization — Ranger / Archivist / Sniper
**Date:** 2026-05-28

---

## Canonical contract (to be encoded in SKILL.md §1)

```
INTERNAL NAME   SLOT KEY     SLOT CONTRACT   PHASE LABEL (display)
────────────────────────────────────────────────────────────────────
Ranger          discovery    write_pending   discovery
Archivist       refinement   write_analysis  refinement
Sniper          execution    controlled      execution
```

- **Internal name**: what SKILL.md prose and agent-facing text use to refer to the role concept
- **Slot key**: the machine-readable identifier used in `roles/*.yaml` and `skill.yaml` — never changes
- **Phase label**: what appears in `[Strategist] phase=<label>` progress events — always the slot key

---

## Change 1 — SKILL.md: rename in prose + add contract table

**File:** `strategist/SKILL.md`

**1a. Add contract table to line 4 (intro paragraph), after "three pluggable slots:"**

Replace:
```
three pluggable slots: Scout (discovery) → Engineer (refinement) → Hunter (execution).
You do not perform discovery, refinement, or execution yourself — you delegate.
```
With:
```
three pluggable slots: Ranger (discovery) → Archivist (refinement) → Sniper (execution).
You do not perform discovery, refinement, or execution yourself — you delegate.

| Internal name | Slot key   | Contract       | Progress label |
|---------------|------------|----------------|----------------|
| Ranger        | discovery  | write_pending  | discovery      |
| Archivist     | refinement | write_analysis | refinement     |
| Sniper        | execution  | controlled     | execution      |
```

**1b. Bulk rename in prose (replace_all):**

| Find | Replace |
|------|---------|
| `Scout` | `Ranger` |
| `Engineer` | `Archivist` |
| `Hunter` | `Sniper` |
| `scout_label` | `ranger_label` |
| `engineer_label` | `archivist_label` |
| `hunter_label` | `sniper_label` |
| `scout_failed` | `ranger_failed` |

**Exceptions — do NOT rename these (external vocabulary):**
- `discovery slot` / `refinement slot` / `execution slot`
- `write_pending` / `write_analysis` / `controlled`
- pipeline stage keys (`discovery:`, `refinement:`, `execution:`)
- `execution_provider` in sdd_injection context

---

## Change 2 — skill.yaml: forbidden_behaviors + description prose

**File:** `strategist/skill.yaml`

**2a. Slot description prose** (lines 32–42):

| Find | Replace |
|------|---------|
| `description: Present side quest manifest. Require explicit approval before Hunter moves files.` | `description: Present side quest manifest. Require explicit approval before Sniper moves files.` |
| `# non-blocking: log and continue to Engineer` | `# non-blocking: log and continue to Archivist` |

**2b. forbidden_behaviors** (lines 110–122):

| Find | Replace |
|------|---------|
| `scout_writes_outside_pending` | `ranger_writes_outside_pending` |
| `engineer_writes_non_md` | `archivist_writes_non_md` |
| `invoke_side_quest_hunter_without_approval` | `invoke_side_quest_sniper_without_approval` |

---

## Change 3 — personas/epic.yaml: phase_labels + prose

**File:** `strategist/personas/epic.yaml`

**3a. phase_labels** (lines 5–7):

Replace:
```yaml
phase_labels:
  discovery: scout
  refinement: engineer
  execution: hunter
```
With:
```yaml
phase_labels:
  discovery: Ranger
  refinement: Archivist
  execution: Sniper
```

**3b. approval_prompt prose** (lines 24–29):

| Find | Replace |
|------|---------|
| `Engineer briefing complete.` | `Archivist briefing complete.` |
| `Authorize Hunter deployment?` | `Authorize Sniper deployment?` |
| `Mission halted at Hunter authorization.` | `Mission halted at Sniper authorization.` |
| `To deploy Hunter:` | `To deploy Sniper:` |

---

## Change 4 — personas/pragmatic.yaml: approval_prompt prose

**File:** `strategist/personas/pragmatic.yaml`

The pragmatic persona already uses external names (`discovery`/`refinement`/`execution`) as phase labels — no change needed there.

Only the approval_prompt prose references a role concept:

| Find | Replace |
|------|---------|
| `Proceed to execution?` | `Proceed to execution?` ← no change needed — "execution" is the external label |

No changes required in pragmatic.yaml.

---

## Change 5 — install.sh: variable rename + bug fix

**File:** `strategist/install.sh`

**5a. Variable rename** — find/replace in the wizard section:

| Find | Replace |
|------|---------|
| `read -r scout` | `read -r ranger` |
| `scout="${scout:-brainstorming}"` | `ranger="${ranger:-brainstorming}"` |
| `validate_provider "$scout"` | `validate_provider "$ranger"` |
| `read -r engineer` | `read -r archivist` |
| `engineer="${engineer:-openspec-explore}"` | `archivist="${archivist:-openspec-explore}"` |
| `validate_provider "$engineer"` | `validate_provider "$archivist"` |
| `read -r hunter` | `read -r sniper` |
| `hunter="${hunter:-sdd-ask}"` | `sniper="${sniper:-sdd-ask}"` |
| `validate_provider "$hunter"` | `validate_provider "$sniper"` |
| `[ -n "$hunter" ]` | `[ -n "$sniper" ]` |

**5b. Prompt labels** (cosmetic context hints):

| Find | Replace |
|------|---------|
| `# Scout provider` | `# Ranger (discovery) provider` |
| `echo "  Scout: descobre o espaço do problema → escreve discovery em pending/"` | `echo "  Ranger: descobre o espaço do problema → escreve discovery em pending/"` |
| `printf "Scout provider (write_pending)` | `printf "Ranger provider (discovery, write_pending)` |
| `# Engineer provider` | `# Archivist (refinement) provider` |
| `echo "  Engineer: refina a descoberta → escreve proposal/design/tasks em refined/"` | `echo "  Archivist: refina a descoberta → escreve proposal/design/tasks em refined/"` |
| `printf "Engineer provider (refinement, write_analysis)` | `printf "Archivist provider (refinement, write_analysis)` |
| `# Hunter provider` | `# Sniper (execution) provider` |
| `echo "  Hunter: executa o plano refinado → requer approval gate"` | `echo "  Sniper: executa o plano refinado → requer approval gate"` |
| `printf "Hunter provider (execution, controlled)` | `printf "Sniper provider (execution, controlled)` |
| `echo "  Error: Hunter provider is required."` | `echo "  Error: Sniper provider is required."` |

**5c. Bug fix — roles YAML write** (lines 231–233):

Find:
```bash
scout: ${scout}
engineer: ${engineer}
hunter: ${hunter}
```
Replace with:
```bash
discovery: ${ranger}
refinement: ${archivist}
execution: ${sniper}
```

---

## Change 6 — skills/engineer/ → skills/archivist/

**6a.** Create directory `strategist/skills/archivist/`

**6b.** Copy `strategist/skills/engineer/skill.yaml` → `strategist/skills/archivist/skill.yaml`
  Update:
  - `id: engineer` → `id: archivist`
  - `# path to Scout's output` → `# path to Ranger's output`
  - `handoff to Hunter` → `handoff to Sniper`
  - `output.reviewed_plan_path` description: "Scout's output" → "Ranger's output"

**6c.** Copy `strategist/skills/engineer/SKILL.md` → `strategist/skills/archivist/SKILL.md`
  Update:
  - `# engineer — Agent Instructions` → `# archivist — Agent Instructions`
  - All occurrences of `Hunter` → `Sniper`
  - `Scout` → `Ranger`

**6d.** Delete `strategist/skills/engineer/` directory (both files).

---

## Change 7 — roles/default.yaml: update refinement binding

**File:** `strategist/roles/default.yaml`

Find:
```yaml
refinement: engineer
```
Replace:
```yaml
refinement: archivist
```

Update comment:
```yaml
# Slot keys: discovery, refinement, execution (external names — map to internal: Ranger, Archivist, Sniper).
```

---

## Change 8 — schemas/progress-contract.yaml: update examples + note

**File:** `strategist/schemas/progress-contract.yaml`

**8a. Note** (lines 15–16):

Replace:
```
    epic: scout / engineer / hunter
    Internal slots use generic names (discovery/refinement/execution) in code;
```
With:
```
    epic: Ranger / Archivist / Sniper
    Internal role names (Ranger/Archivist/Sniper) appear in prose and persona labels;
    external slot keys (discovery/refinement/execution) appear in roles YAML and events.
```

**8b. Event examples** (lines 35–38):

| Find | Replace |
|------|---------|
| `phase=scout status=running skill=sdd-diagnose` | `phase=discovery status=running skill=sdd-diagnose` |
| `phase=scout status=done` | `phase=discovery status=done` |
| `phase=engineer status=running skill=engineer` | `phase=refinement status=running skill=archivist` |
| `phase=hunter status=done` | `phase=execution status=done` |

---

## Change 9 — templates/domain/identity/what-i-am.yaml

**File:** `strategist/templates/domain/identity/what-i-am.yaml`

| Find | Replace |
|------|---------|
| `I do not perform root-cause analysis; Scout does.` | `I do not perform root-cause analysis; Ranger does.` |
| `I do not produce plans; Engineer does.` | `I do not produce plans; Archivist does.` |
| `I do not apply changes; Hunter does.` | `I do not apply changes; Sniper does.` |
| `I never invoke Hunter without explicit user approval` | `I never invoke Sniper without explicit user approval` |
| `before every Hunter invocation.` | `before every Sniper invocation.` |

---

## What does NOT change

- Slot keys in roles YAML: `discovery` / `refinement` / `execution` — these are external identifiers
- `protocol.md` — already uses external names throughout, no changes needed
- `roles/mission.yaml` + `roles/spec-driven.yaml` — use `_injected_by_sdd` for execution, no `engineer` reference
- Slot contracts: `write_pending` / `write_analysis` / `controlled`
- All pipeline stage keys in `skill.yaml`
- `sdd_injection.execution_provider` field name
- housekeeping_scan logic, side quest pipeline, learning phase, bootstrap scripts
