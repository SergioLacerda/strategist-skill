# Runbook: `error=role_invocation_failed`

## Symptom

```
error=role_invocation_failed
slot=<discovery|refinement|execution>
provider=<configured_provider>
```

## Root Cause

The configured slot provider's `skill.yaml` is missing, schema-invalid, or not
callable in the current runtime — the provider is not installed, is installed at
the wrong path, or the installed runtime doesn't expose it as a skill. This is
specifically about the provider being uninvokable, not about what it does once
invoked: standalone `SKILL.md` style quirks are not this condition either —
preflight validates only that the manifest exists and matches the slot's risk
contract. (Discovery no longer invokes an external weapon at all — all
discovery subtypes resolve to `internal_skills/ranger` natively — so
discovery-subtype coverage is not a runtime condition of any kind anymore;
see `.analysis/refined/20260728-ranger-drift-eval/`.)

## Resolution Steps

1. Run `strategist check` — confirms the current slot → provider mapping and
   whether it reports `ok`.
2. Confirm the provider is actually installed where Strategist expects a skill
   (e.g. `~/.claude/skills/<provider>/`).
3. Confirm `.strategist/active.yaml`'s `slots.<phase>` value matches an installed
   provider's id exactly — typos are the most common cause.
4. If the provider's `skill.yaml` exists but fails schema validation, fix or
   reinstall it.
5. Rerun `strategist check` until STATUS reports `ok` before retrying the mission.

## Refinement-Specific Escalation

The general Resolution Steps above assume a fix exists: install the provider correctly,
or point `active.slots.<phase>` at one that is installed. On the **refinement** slot
specifically, that assumption can fail — no alternative refinement provider may be
installed at all (verified twice: `.analysis/pending/drift_skill.txt` and mission
`20260819-portable-light-client-eval`, see [ADR-0027](../adr/0027-refinement-native-role-for-light-client.md)).

When `slot=refinement` and steps 1–4 above don't resolve it:

1. Confirm `roles/archivist.yaml` and `internal_skills/archivist/SKILL.md` exist (present
   by default install) — Archivist has a native-role path structurally identical to
   Ranger's and Sniper's, even though `00-routing.md` has, as of this writing, no formal
   override statement for refinement the way it does for discovery.
2. Do **not** substitute the parent agent for Archivist silently — per
   `agent-protocol.md` §1b, that is `direct_execution` drift even if the output would be
   correct.
3. Escalate to the user with the concrete choice instead of hard-stopping silently: (a)
   authorize treating Archivist as native for this mission, (b) reconfigure
   `active.slots.refinement` to a different, actually-installed provider, or (c) accept
   analysis-only as the mission's terminal outcome. Do not pick for the user.
4. If the user authorizes (a), record it as a mission-scoped decision (an ADR, per the
   Opportunity Attack routine, is the natural place) and proceed with the native-role
   mechanism per `contracts/narrative/03-discovery.md`'s Ranger contract shape, applied to
   Archivist.

## Reference

- `.strategist/SKILL.md` § Role Invocation Failures
- `.strategist/contracts/machine/preflight.yaml` → `error_conditions.role_invocation_failed`
- [ADR-0027](../adr/0027-refinement-native-role-for-light-client.md) — refinement-specific precedent and open follow-up work
