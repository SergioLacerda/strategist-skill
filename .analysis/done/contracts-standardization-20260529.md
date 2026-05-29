# Analysis: Strategist Module Contract Standardization
**Date:** 2026-05-29
**Status:** pending
**Origin:** perf-opt-20260529 Task 11

---

## Motivation

During the perf-opt mission, five new modules were introduced (compile scripts, compiled artifact schemas, LearningBuffer shell). Additionally, the existing Strategist modules (bootstrap, preflight, context-enrichment, learning-curator) have implicit contracts embedded in SKILL.md prose that have never been formally specified, making them difficult to maintain and test.

This analysis drives a future mission to formally specify input/output contracts for ALL Strategist modules.

---

## Scope

### Modules requiring formal contracts

| Module | Type | Status |
|--------|------|--------|
| `bootstrap` | agent phase | implicit in SKILL.md §1 |
| `preflight` | agent phase | implicit in SKILL.md §2 |
| `context-enrichment` | agent phase | implicit in SKILL.md §4 |
| `learning-curator` | agent phase | implicit in SKILL.md §8 |
| `compile-knowledge-index.sh` | shell script | partially defined in design.md §3.2 |
| `compile-domain.sh` | shell script | partially defined in design.md §3.3 |
| `compile-config.sh` | shell script | partially defined in design.md §3.4 |
| `compile-all.sh` | shell script | partially defined in design.md §3.5 |
| `check-stale.sh` | shell script | partially defined in design.md §3.1 |
| `LearningBuffer` (shell) | write path | partially defined in design.md §6 |

### Compiled artifact schemas (import from perf-opt design.md §2, do not duplicate)

- `.compiled/.index.gz` — `strategist-compiled-index/1.0`
- `.compiled/.domain.gz` — `strategist-compiled-domain/1.0`
- `.compiled/.config.gz` — `strategist-compiled-config/1.0`
- `.compiled/.manifest.gz` — `strategist-compiled-manifest/1.0`

---

## Contract Schema Definition (per module)

Each contract file in `.strategist/contracts/<module-name>.yaml` should specify:

```yaml
module: <name>
type: agent_phase | shell_script | write_path
contract:
  input:
    - name: <field>
      type: <type>
      required: true/false
      description: <what it is>
  output:
    - name: <field>
      type: <type>
      description: <what it produces>
  error_conditions:
    - code: <error_code>
      trigger: <when this happens>
      behavior: <what the module does>
  write_scope: <path pattern or "read-only">
  owner: <slot or internal>
```

---

## Expected Deliverables

1. **`contracts/` directory** in `.strategist/` with one YAML per module (10 files)
2. **Updated references** in `SKILL.md` — each phase section links to its contract file
3. **Validation rule** in preflight: if contracts dir exists, load active module's contract and validate input before invoking

---

## Starting Point

The perf-opt `design.md §3` already defines contracts for the four compile scripts and `check-stale.sh`. Those should be the first entries written in the contracts mission — import their definitions without re-deriving.

---

## Risk

| Risk | Severity |
|------|----------|
| Contracts become stale as SKILL.md evolves | Medium — mitigated by linking contracts from SKILL.md |
| Over-specification creates maintenance burden | Low — keep contracts declarative, not procedural |
