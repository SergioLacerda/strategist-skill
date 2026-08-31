# ADR-0005 — Per-slot write contracts (read_only / write_analysis / controlled)

**Status:** Accepted  
**Date:** 2026-05-28  
**Context:** Slot write contracts design (2026-05-28-slot-write-contracts-design.md)

> 2026-06-26 note: this ADR records the slot write-contract model. The current
> Strategist execution slot is restricted to approved documentation,
> diagrams, analysis artifacts, and handoffs; source-code mutation is forbidden.

---

## Context

The discovery (Ranger) and refinement (Archivist) slots need to write artifacts under the configured runtime workspace as part of the normal mission flow. With the original model where all slots were `read_only` except Sniper (`controlled`), **any artifact write — even a local Markdown file in `<base_path>/pending/` — required passing through the approval gate**.

This made the flow excessively interactive: creating a discovery artifact under `<base_path>/pending/` required an explicit "yes" from the user, even for a low-risk operation with no impact on code.

The question: how to differentiate low-risk writes (local analysis artifacts) from high-risk writes (code, configs, system files)?

Alternatives considered:
- **Gate for everything** — keep all slots as `read_only` + universal gate
- **No gate** — trust providers not to write in the wrong places
- **Scope contracts** — declare allowed write scope and type per slot, validated in preflight

## Decision

Three write contract levels, declared in `skill.yaml` and validated by the Strategist in preflight:

| Contract | Write scope | Allowed types | Approval gate |
|----------|------------|--------------|--------------|
| `read_only` | none | — | not applicable |
| `write_analysis` | `<base_path>/` and derivatives (`pending/`, `refined/`, `archived/`, `todo/`) | `.md` | no |
| `controlled` | `<base_path>/archived/` and approved `.md` documentation | `.md` | **mandatory** |

The provider's `risk_score` (declared in the provider's `skill.yaml`) must match the contract required by the slot. Mismatch blocks in preflight with `slot_risk_mismatch`.

The `write_pending` contract was discontinued. Discovery now produces the canonical artifact
`<base_path>/pending/<mission_id>-analysis.md`, so the minimum compatible contract
for Ranger is also `write_analysis`.

Runtime violations:
- Non-`.md` type write by `write_analysis` → `slot_write_type_violation`
- Write outside the declared scope → `slot_write_scope_violation`

### Enforcement levels

The contract uses three enforcement levels:

| Level | Role in this ADR | Current enforcement |
|-------|------------------|---------------------|
| Declarative | Slot contracts, allowed scopes, and allowed types are declared in skill/provider metadata. | `skill.yaml`, role manifests, and contract documentation define the expected write boundary. |
| Detective | Preflight and runtime checks detect drift from declared contracts and report named violations. | Preflight rejects slot/provider mismatches; runtime violations are reported as `slot_write_type_violation` or `slot_write_scope_violation`. |
| Preventive | A slot cannot continue materializing outside its approved boundary once a blocking violation is detected. | The approval gate remains mandatory for `controlled` writes, and scope/type violations stop the governed flow instead of being silently accepted. |

## Consequences

**Positive:**
- Discovery and refinement write artifacts silently — natural flow without unnecessary interruptions
- Approval gate preserved only where it truly matters (Sniper / `controlled` slot)
- Contracts verified in preflight — configuration errors detected before the mission starts, not in the middle
- Extensible model: new contracts can be added without changing the main pipeline

**Negative:**
- Two scope verification points: preflight (risk_score) and runtime (actual write) — these can diverge if the provider does not honor its declared contract
- A malicious provider with `risk_score: write_analysis` could attempt to write outside scope; runtime validation is necessary but depends on the orchestrator detecting the violation
- `known-providers.yaml` must be kept up to date for providers that do not declare `risk_score` in their `skill.yaml` — otherwise preflight cannot validate

---

## Addendum (2026-08-30) — mapping onto the unified enforcement-tier vocabulary

The text above is left as originally written; this addendum does not change the
Decision or Consequences, it cross-references a newer, more general vocabulary
introduced elsewhere. `internal/embed/defaults/contracts/machine/errors.yaml`
previously tagged only error tokens with a binary `enforced_by: binary|agent`
field. `.analysis/refined/20260830-skill-gaps-triage/tasks.md` Task 4 (G03)
generalized that into a 3-tier vocabulary — `machine_enforced` /
`machine_observed` / `agent_only` — now defined canonically in `errors.yaml`
and applied across other machine/narrative contract files, not just error
tokens.

This ADR's "Enforcement levels" table (§ above) predates that generalization
and uses different names for a related but not identical distinction: it
describes *stages in a contract's enforcement lifecycle* (declare it, detect
drift from it, then prevent continuation past a detected violation), not a
per-clause classification of *whether a specific obligation is machine-checked
today*. The mapping is:

| ADR-0005 level | What it describes | Maps to (per concrete clause) |
|---|---|---|
| Declarative | The contract is declared in metadata (`skill.yaml`, role manifests) with no detection or prevention wired yet. | `agent_only` — nothing machine-side observes or blocks a declarative-only clause; the orchestrating agent is trusted to honor the declaration. |
| Detective (without a following Preventive step) | A Go-side check runs on a *live, reachable* path and reports a named violation, but nothing stops the flow because of it. | `machine_observed` — detected and reported, not blocking. |
| Detective + Preventive together, on a live path | A Go-side check detects the violation on a live, reachable path *and* that detection actually stops the governed flow from continuing. | `machine_enforced` — checked and blocking. |
| Detective + Preventive implemented, but only exercised by an eval/test harness | The decision logic exists as a real Go function with Detective+Preventive shape, but its only caller is a scenario/eval runner rather than a path a live mission write goes through. | `agent_only` — same "wired but not connected" pattern as `internal/domain/pipeline_bypass.go`'s `EvaluatePipelineBypass` (see `errors.yaml`'s vocabulary note); a function nothing on a live path calls does not make the obligation machine-checked, however correct the function is in isolation. |

Applying this mapping to this ADR's own two runtime violations, verified
against actual 2026-08-30 Go call sites:

- **`slot_risk_mismatch`** (preflight risk_score match) — `machine_enforced`.
  `internal/check/check_slots.go#resolveSkillProviderSlot` performs this
  comparison and its error is appended to `strategist check`'s error list,
  which fails the command (`check=failed`) with a non-zero exit — a live,
  reachable, blocking path. (This is a different code path from
  `errors.yaml`'s `slot_risk_mismatch` *error token*, which stays
  `agent_only` there because no code path emits that literal
  `error=slot_risk_mismatch` string — the two catalogs classify different
  things: this ADR's check-time violation vs. errors.yaml's specific
  emitted-token contract.)
- **`slot_write_type_violation` / `slot_write_scope_violation`** (runtime
  write-scope check) — `agent_only`, not `machine_enforced`, despite a
  complete, unit-tested implementation:
  `internal/domain/contract_validator.go#ValidateSlotWrite` correctly
  implements the Detective+Preventive shape (returns a named violation error
  that would block a caller), but its only non-test caller repo-wide is
  `internal/eval/harness_policy_scenarios.go#runSlotWriteScopeScenario` — the
  `strategist eval run` scenario harness, which checks the function's
  correctness against fixtures. No live Ranger/Archivist/Sniper write path
  calls it. This ADR's own "Negative" consequence above already flagged this
  gap ("runtime validation is necessary but depends on the orchestrator
  detecting the violation") — the vocabulary now makes that gap visible as a
  tag instead of prose alone.
