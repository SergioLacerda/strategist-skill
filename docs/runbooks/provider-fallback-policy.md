# Runbook: Slot provider failure — diagnosis and retired fallback policy

> Status: retired. This document is retained for migration and historical
> evidence only. Active missions never apply block/ask/native provider fallback.

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

A valid capability descriptor does not prove a provider is invocable in the active agent runtime. Static readiness and live invocation evidence remain separate. ADR-0028 is historical context; it is not an active fallback authorization.

## Resolution Steps (migration only)

1. Confirm the exact `slot` and `provider` from the blocked event.
2. Classify the failure using the Root Cause list above:
   - missing descriptor → `slot_provider_not_found`;
   - present but invalid descriptor/role → `role_provider_invalid`;
   - risk mismatch → `slot_risk_mismatch`;
   - valid metadata but unavailable invocation → `role_invocation_failed`.
3. Do not select a native role as a substitute. Repair the selected binding or stop with `role_invocation_failed`.

There is no active `provider_resolution_policy` field. If an older
`.strategist/active.yaml` contains it, remove it and rerun installation/check.

## Decision Gates

- Stop if the native role is absent, invalid, or incompatible with the slot.
- Stop on every unavailable or incompatible selected provider.
- Do not change providers implicitly or bypass an approval gate.
- Continue only after the selected binding itself resolves to an invocable, compatible Weapon.

## Expected Evidence

- Before: blocked token with slot and configured provider.
- After: `strategist check` reports the selected binding and live invocation is separately evidenced.
- The mission resumes only after the selected binding is repaired; it does not substitute a native role.

## Stop Conditions

- descriptor or role invalid;
- incompatible slot/risk/write scope;
- explicit user refusal;
- compile/check failure;
- attempted phase or Approval Gate bypass.

## Reference

- `.strategist/contracts/machine/errors.yaml` — canonical reason/action text for each token
- [ADR-0028](../adr/0028-native-role-resilient-baseline.md) — historical fallback decision
- `docs/runbooks/role-invocation-failed.md` — dedicated runbook for `role_invocation_failed`, including the refinement-slot-specific escalation for when no alternative refinement provider exists at all
