# Slot Risk Contract Fix — Design Spec
**Date:** 2026-05-28  
**Status:** pending implementation  
**Topic:** Align Strategist slot risk vocabulary with SDD ecosystem + known-providers registry + wizard validation

---

## Problem Statement

The Strategist's preflight blocks on three distinct failures:

| Failure | Root Cause |
|---------|-----------|
| Hunter `sdd-ask` fails contract | Strategist requires `controlled_write`, SDD uses `controlled` — vocabulary mismatch |
| Scout/Engineer unverifiable | `brainstorming`, `openspec-explore` don't declare `risk_score` in their SKILL.md |
| Misconfiguration detected too late | Wizard accepts any provider name without validating the risk contract |

---

## Goals

1. `sdd-ask` works as Hunter without changing `sdd-ask` itself
2. Scout/Engineer resolution succeeds when risk_score is not in SKILL.md
3. Wizard catches mismatches at configuration time, not at runtime
4. No edits to files outside this repo (`~/.claude/skills/` untouched)

---

## Architecture

Three changes, each independent:

```
strategist/
├── skill.yaml                          ← (1) Hunter contract: controlled_write → controlled
├── SKILL.md                            ← (2) Preflight resolution order updated
├── templates/
│   └── known-providers.yaml            ← (3) New template for provider registry
└── install.sh                          ← (4) wizard validates providers + generates known-providers.yaml
```

`.strategist/` (installed runtime):
```
.strategist/
└── known-providers.yaml                ← generated at install time
```

---

## Section 1 — `known-providers.yaml` Template

**New file:** `strategist/templates/known-providers.yaml`  
**Installed to:** `.strategist/known-providers.yaml` by `copy_skill_runtime()`

```yaml
# Strategist provider risk registry
# Consulted during preflight when a provider's SKILL.md or skill.yaml
# does not declare risk_score.
# Values: read_only | controlled | orchestrator
# Add entries here for any custom providers you use.

providers:
  brainstorming: read_only
  openspec-explore: read_only
  openspec-propose: read_only
  openspec-apply-change: controlled
  openspec-archive-change: read_only
  sdd-ask: controlled
  sdd-ask-full: controlled
  sdd-diagnose: read_only
  sdd-converge: high
  sdd-correct: medium
  sdd-stabilize: medium
  sdd-validate-governance: medium
  sdd-organize: read_only
  sdd-review-architecture: read_only
  engineer: read_only
```

**Idempotency:** `copy_skill_runtime()` only copies this file if `.strategist/known-providers.yaml` does not already exist. User additions are preserved on reinstall.

---

## Section 2 — `skill.yaml` Vocabulary Fix

**File:** `strategist/skill.yaml`

**Change:** Hunter slot contract `controlled_write` → `controlled`

```yaml
slots:
  discovery:
    contract: read_only
  refinement:
    contract: read_only
  execution:
    contract: controlled      # was: controlled_write
```

**Rationale:** `controlled_write` was invented by the Strategist but does not exist in the SDD risk vocabulary. The correct SDD term for an executor that performs governed writes is `controlled`.

**Risk vocabulary (canonical):**

| Value | Meaning | Allowed slots |
|-------|---------|--------------|
| `read_only` | Reads only, no side effects | Scout, Engineer |
| `controlled` | Writes with governance guardrails | Hunter |
| `orchestrator` | Coordinates other slots, no direct execution | Strategist |

---

## Section 3 — SKILL.md Preflight Update

**File:** `strategist/SKILL.md` — Section 2c–2d

**Updated risk_score resolution order** (for each slot provider):

```
1. <skill_root>/<provider>/skill.yaml     → field: risk_score
2. ~/.claude/skills/<provider>/SKILL.md   → frontmatter field: risk_score
3. .sdd/skills/registry.json              → field: risk_score
4. .strategist/known-providers.yaml       → key: providers.<provider>
5. None found → BLOCK: slot_provider_unknown_risk
6. Found → compare against slot contract
7. Mismatch → BLOCK: slot_risk_mismatch
```

**New stop condition:** `slot_provider_unknown_risk` — added alongside the existing `slot_risk_mismatch`.

---

## Section 4 — Wizard Validation in `install.sh`

**File:** `strategist/install.sh` — `run_wizard()` function

**New helper:** `validate_provider(name, required_risk, slot_label)`

```
validate_provider(provider, required_risk, slot):
  1. Resolve provider: check ~/.claude/skills/<provider>/, SDD registry, local skills/
     → Not found: print error, ask for new name
  2. Read risk_score from: SKILL.md frontmatter, skill.yaml, SDD registry,
     templates/known-providers.yaml (source template, not installed copy)
     → Not found: print warning
                  prompt: "risk_score for '<provider>' not declared. Enter value [read_only/controlled]: "
                  read user input, validate against [read_only, controlled, orchestrator]
                  offer to add to known-providers.yaml
  3. Compare risk_score to required_risk
     → Mismatch: print "Slot <slot> requires <required_risk>, but <provider> declares <actual_risk>"
                 ask for new provider name, loop
  4. Match: accept provider
```

**Applied to wizard prompts:**

| Wizard step | Required risk | Validation |
|-------------|--------------|-----------|
| Scout provider | `read_only` | validate_provider name read_only "Scout" |
| Engineer provider | `read_only` | validate_provider name read_only "Engineer" |
| Hunter provider | `controlled` | validate_provider name controlled "Hunter" |

**Wizard completion:** after all providers validated, write `roles/default.yaml` to `.strategist/`.

---

## What Does NOT Change

- `sdd-ask` skill itself (no edits to its skill.yaml or SKILL.md)
- `brainstorming`, `openspec-explore` in `~/.claude/skills/` (untouched)
- `.strategist/roles/default.yaml` format
- The approval gate and execution flow
- Bootstrap scripts and release workflow

---

## Fix Immediate (pre-implementation)

While the spec is implemented, unblock by editing `.strategist/roles/default.yaml`:

```yaml
scout: brainstorming
engineer: openspec-explore
hunter: sdd-ask         # will work once skill.yaml contract is changed to 'controlled'
```

Add to `.strategist/known-providers.yaml` (create if absent):

```yaml
providers:
  brainstorming: read_only
  openspec-explore: read_only
  sdd-ask: controlled
```

This allows `/strategist` to run without waiting for the full implementation.
