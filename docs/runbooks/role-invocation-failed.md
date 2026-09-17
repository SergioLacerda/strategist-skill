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

## Ranked Runtime Escalation

When the configured provider is a certified Ranked binding, `strategist check`
being structurally ready is not sufficient to prove live invocability. Inspect
the role/provider runtime contract and its prepared state before retrying:

1. Confirm the selected role/provider pair in `.strategist/active.yaml`,
   `.strategist/plugins.lock`, and `.strategist/plugins/catalog.yaml`.
2. For `archivist -> openspec-propose`, confirm the runtime contract points to
   `.strategist/openspec` and that `config.yaml` is directly under that root.
   A repository-root or nested `openspec/config.yaml` is not a valid substitute.
3. Confirm `.strategist/ranked-runtimes.yaml` contains the selected slot,
   provider, and matching certification digest.
4. Run `openspec context --json` with the working directory set to
   `.strategist/openspec`; do not run it from the repository root and do not
   initialize the runtime lazily.
5. If runtime state, digest, root, or healthcheck fails, treat the binding as
   unavailable and reinstall/repair through the governed installation path.
   Do not fall back silently to native Archivist, another provider, the
   repository root, or `.analysis`.

For `ranger -> brainstorming` with `runtime.kind: none`, no OpenSpec runtime is
required or created. Static certification, project-contract readiness, and live
provider invocation must be reported as separate evidence dimensions.

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
- [`docs/runbooks/provider-fallback-policy.md`](provider-fallback-policy.md) — general runbook for the other three slot-provider failure tokens (`slot_provider_not_found`, `role_provider_invalid`, `slot_risk_mismatch`) and the block/ask/native fallback policy
