# Runbook: `error=role_invocation_failed`

## Symptom

```
error=role_invocation_failed
slot=<discovery|refinement|execution>
provider=<configured_provider>
```

## Root Cause

The selected Role/Weapon binding is not executable in the declared runtime.
For Custom bindings this can still mean a missing or invalid external
connector. For Ranked bindings it means the persisted `plugins.lock` record
does not match the compiled catalog, the Embedded dispatch is unavailable, or
the explicitly registered prompt bridge could not execute the embedded
payload. Ranked execution never repairs itself by reading a host skill
directory or substituting a native Role.

## Resolution Steps

1. Run `strategist check` — confirms the current slot → provider mapping and
   whether it reports `ok`.
2. Identify the binding mode in `.strategist/plugins.lock`. For `mode: ranked`,
   compare Role, slot, Weapon, source digest, execution mode, connector,
   entrypoint, and binding digest with `.strategist/plugins/catalog.yaml`; for
   `mode: custom`, continue with the external connector checks below.
3. Confirm `.strategist/active.yaml`'s `slots.<phase>` value matches the lock
   binding exactly. If a Ranked record differs, rerun the governed install or
   wizard so the lock is materialized from the compiled catalog.
4. For a Ranked `prompt_bridge` Weapon, register the host-agent bridge at the
   adapter boundary and retry. The bridge must consume Strategist's embedded
   payload; it must not resolve another skill path. Without that bridge, the
   correct result is `role_invocation_failed`.
5. For a Custom binding, confirm the provider is installed where its explicit
   connector expects it (for example, `~/.claude/skills/<provider>/`) and that
   its manifest is valid.
6. Rerun `strategist check` until static binding integrity is ready before
   retrying the mission. Static readiness does not by itself certify a live
   prompt-bridge invocation.

### CODEX bootstrap-specific checks

If the client is CODEX and the project contains `.codex/`, inspect the
`strategist check --json` warnings for one of these non-blocking reasons:

- `codex_bootstrap_missing` — create or refresh the seed with `strategist compile`
  or reinstall;
- `codex_bootstrap_stale` — the generated Strategist Runtime Discovery section
  does not match the source writer;
- `codex_bootstrap_unreadable` — the project-local seed cannot be read.

These diagnostics cover `.codex/commands.md` in the project. Separately verify
that the existing global CODEX shim at
`~/.codex/skills/strategist/SKILL.md` has a `skill_root` pointing to the current
`.strategist` directory and that the configured provider exists in the effective
CODEX skill root. A static `check` result still does not prove live provider
invocation; only an authorized `strategist-live-evidence/v1` probe can certify
that client row.

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

For `ranger -> brainstorming`, the compiled binding selects the Embedded
`prompt_bridge` mode and the payload is read from the CLI's embedded defaults.
No OpenSpec runtime or host skill loader is required. Static certification,
binding integrity, and live bridge invocation remain separate evidence
dimensions.

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
