# Runbook: CLI Authorization Policy

## Boundary

`strategist plugins authorize` is a deterministic CLI-before-write decision.
It composes static runtime/binding evidence, local and Strategist gates, and
the connector's enforceable target permission. It does not invoke external
providers and cannot intercept a parent agent that writes without calling the
CLI.

## Required evidence

The JSON report separates:

- `runtime`: installed and parseable `.strategist` configuration;
- `binding`: persisted discovery/refinement Role→Weapon resolution;
- `gate`: local `execution_gate` and explicit `approval-gate` acceptance;
- `target`: path classification and positive connector enforcement;
- `live_provider`: host/provider invocation evidence, normally
  `unknown`/`unverified` for external providers.

All required dimensions must be `ready` or `allowed`. Unknown, unsupported,
missing, observed-only, or forbidden target permissions fail closed.

## Stop conditions

Stop and investigate when the report returns:

- `role_provider_binding_invalid` or `ranked_runtime_digest_mismatch`;
- `approval_gate_not_accepted` or `local_execution_gate_blocked`;
- `forbidden_planning_path`;
- `write_target_not_enforceable`;
- `provider_runtime_unavailable` or another runtime failure.

Do not silently substitute a provider or treat `strategist check --json` as
proof of live host invocation. Approval of a Strategist analysis authorizes
only declared analysis/documentation targets; source, hook, configuration, and
test changes remain separately authorized implementation handoffs.

## Example

```bash
strategist plugins authorize \
  --target=.analysis/refined/mission/tasks.md \
  --approval-gate=accepted --json
```

Use the report's `decision`, `reason_code`, and `exit_class` in CI. Preserve
the complete JSON as evidence when a check is denied or blocked.
