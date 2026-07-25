# Runbook: Delegated Execution Blocked States

Three related blocked states from local execution context / delegated invocation
resolution (see `.strategist/agent-protocol.md` § 6). Grouped because they share one
audience — an operator integrating or debugging a governance adapter — and one
resolution context: fixing how the adapter supplies (or the workspace declares)
execution provider information.

## Symptom: `error=local_execution_provider_missing`

```
error=local_execution_provider_missing
reason=delegated invocation did not provide execution_provider
```

### Root Cause

Delegated invocation mode was declared
(`local_execution_context.invocation_mode: delegated`) but no `execution_provider`
field was supplied in the forwarded context.

### Resolution Steps

1. Confirm the invoking adapter/orchestrator actually sets
   `governance_injection.execution_provider` before calling Strategist.
2. If direct (non-delegated) invocation was intended instead, remove/omit
   `invocation_mode: delegated` so Strategist falls back to `active.slots.execution`.

---

## Symptom: `error=execution_provider_unavailable`

```
error=execution_provider_unavailable
reason=resolved execution_provider cannot be invoked in this environment
```

### Root Cause

An execution provider was resolved (from either delegated context or
`active.slots.execution`) but cannot actually be invoked in the current runtime —
missing installation, invalid skill manifest, or an environment mismatch.

### Resolution Steps

1. Verify the resolved provider is installed and its skill manifest is valid — same
   diagnostic path as `role-invocation-failed.md`.
2. Confirm the execution environment (this session/runtime) actually has access to
   that provider — a provider valid in one environment is not guaranteed available
   in another.

---

## Symptom: `drift=local_execution_context_bypass`

```
drift=local_execution_context_bypass
reason=Strategist attempted direct execution instead of invoking the resolved provider
```

### Root Cause

This is a self-detected protocol violation, not an environment problem: Strategist's
own orchestration attempted to materialize documentation directly instead of
delegating to the resolved execution provider.

### Resolution Steps

1. This should stop the mission automatically — no user action is needed to "fix" a
   root cause outside their control.
2. If seen repeatedly, report it — it indicates a Strategist orchestration bug, not
   a workspace configuration issue.

## Reference

- `strategist/SKILL.md` § Blocked States
- `.strategist/agent-protocol.md` § 6 Local Execution Context and Approval Gates
