# Runbook: Slot provider failure — diagnosis and fallback policy

## Symptom

Any of:

```
error=slot_provider_not_found
slot=<discovery|refinement|execution>
```
```
error=role_provider_invalid
slot=<discovery|refinement|execution>
```
```
error=slot_risk_mismatch
slot=<discovery|refinement|execution>
```
```
error=role_invocation_failed
slot=<discovery|refinement|execution>
provider=<configured_provider>
```

## Root Cause

Each token names a different point of failure in slot provider resolution (see `.strategist/contracts/machine/errors.yaml`, the canonical source for this text):

- **`slot_provider_not_found`** — "configured slot provider has no capability descriptor under `.strategist/skills/`."
- **`role_provider_invalid`** — "resolved role/provider descriptor is present but invalid for the slot."
- **`slot_risk_mismatch`** — "provider `risk_score` is incompatible with the slot's declared contract."
- **`role_invocation_failed`** — "a configured role/provider cannot be invoked from the installed runtime." (See also `docs/runbooks/role-invocation-failed.md` for this token's own dedicated runbook, including a refinement-slot-specific escalation.)

A valid capability descriptor does not prove a provider is invocable in the active agent runtime — this gap is exactly what [ADR-0028](../adr/0028-native-role-resilient-baseline.md) addresses architecturally. This runbook is the operational companion to that decision: ADR-0028 records *why* native roles are the resilient baseline; this runbook covers *how* to diagnose and recover in the moment.

## Resolution Steps

1. Confirm the exact `slot` and `provider` from the blocked event.
2. Classify the failure using the Root Cause list above:
   - missing descriptor → `slot_provider_not_found`;
   - present but invalid descriptor/role → `role_provider_invalid`;
   - risk mismatch → `slot_risk_mismatch`;
   - valid metadata but unavailable invocation → `role_invocation_failed`.
3. Verify that a candidate native role exists for the affected slot (`roles/<slot-role>.yaml` + `internal_skills/<slot-role>/SKILL.md`) and declares the same slot.

## Fallback Policy Decision

Once a compatible native role is confirmed, resolution follows the fallback policy from `.strategist/active.yaml#provider_resolution_policy` (absent or empty defaults to `ask`) — see [ADR-0028](../adr/0028-native-role-resilient-baseline.md) for the rationale and `.strategist/contracts/narrative/00-routing.md` § Provider Resolution Policy (ADR-0028) for the full contractual procedure. Summary only (the linked sections are normative):

- **`block`** — stop; repair provider configuration or installation. No automatic fallback.
- **`ask`** (default) — present the concrete choice to the user: (a) use the native role for this mission, (b) reconfigure the slot to a different installed provider, or (c) accept the mission's current terminal state without this slot. Do not pick for the user.
- **`native`** — use the compatible native role automatically, but emit degradation evidence naming the configured provider, the effective provider, and the reason.

None of the three policies authorizes skipping the Strategist Approval Gate, changing write scope, or inventing a provider that isn't `roles/<id>.yaml`-backed.

## Decision Gates

- Stop if the native role is absent, invalid, or incompatible with the slot.
- Stop if fallback policy is `block`.
- Stop if policy is `ask` and confirmation is absent.
- Stop if changing providers would expand write scope or bypass an approval gate.
- Continue only after the runtime resolves the slot to an invocable, compatible role.

## Expected Evidence

- Before: blocked token with slot and configured provider.
- After: `strategist check` reports the resolved role with `kind=native_role` (or a corrected external provider, if that was the fix).
- The mission resumes at the next incomplete phase — it does not restart completed phases.

## Stop Conditions

- descriptor or role invalid;
- incompatible slot/risk/write scope;
- explicit user refusal;
- compile/check failure;
- attempted phase or Approval Gate bypass.

## Reference

- `.strategist/contracts/machine/errors.yaml` — canonical reason/action text for each token
- [ADR-0028](../adr/0028-native-role-resilient-baseline.md) — architectural decision (native roles as resilient baseline, block/ask/native policy)
- `.strategist/contracts/narrative/00-routing.md` § Provider Resolution Policy (ADR-0028) — full contractual procedure
- `docs/runbooks/role-invocation-failed.md` — dedicated runbook for `role_invocation_failed`, including the refinement-slot-specific escalation for when no alternative refinement provider exists at all
