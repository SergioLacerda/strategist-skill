# ADR-0005 — Per-slot write contracts (read_only / write_analysis / controlled)

**Status:** Accepted  
**Date:** 2026-05-28  
**Context:** Slot write contracts design (2026-05-28-slot-write-contracts-design.md)

> 2026-06-26 note: this ADR records the slot write-contract model. The current
> Strategist execution slot is restricted to approved documentation,
> diagrams, analysis artifacts, and handoffs; source-code mutation is forbidden.

---

## Context

The discovery (Ranger) and refinement (Archivist) slots need to write artifacts to `.analysis/` as part of the normal mission flow. With the original model where all slots were `read_only` except Sniper (`controlled`), **any artifact write — even a local `.md` in `pending/` — required passing through the approval gate**.

This made the flow excessively interactive: creating `.analysis/pending/discovery.md` required an explicit "yes" from the user, even for a low-risk operation with no impact on code.

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
