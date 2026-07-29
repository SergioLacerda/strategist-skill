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

## Reference

- `.strategist/SKILL.md` § Role Invocation Failures
- `.strategist/contracts/machine/preflight.yaml` → `error_conditions.role_invocation_failed`
