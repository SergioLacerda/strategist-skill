# Strategist — Observability Contract

**Status:** Accepted
**Last Updated:** 2026-06-22

This document defines the canonical set of attributes, event sequences, and
integration patterns for consuming Strategist telemetry in CI/CD pipelines and
observability platforms.

## Transport

Strategist emits structured logs via `log/slog` and OpenTelemetry spans.

| Signal | Format | Default |
|--------|--------|---------|
| Structured logs | `log/slog` JSON to stdout | Always active |
| OTel spans | gRPC OTLP | Disabled unless `OTEL_EXPORTER_OTLP_ENDPOINT` is set |

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `""` | gRPC collector endpoint (e.g. `localhost:4317`). Empty → no-op exporter. |
| `OTEL_SERVICE_NAME` | `strategist` | Service name in traces. |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true` | TLS disabled by default. Set `false` in production. |

---

## Span attribute keys

All attributes are namespaced under `strategist.*`. Defined in `internal/telemetry/schema.go`.

### Identity

| Key | Type | Description |
|-----|------|-------------|
| `strategist.mission_id` | string | Unique mission identifier (e.g. `20260620-feature-xyz`) |
| `strategist.correlation_id` | string | Correlation ID across phases of the same mission |
| `strategist.component` | string | Emitting component: `root`, `install`, `compile`, `validate` |

### Runtime

| Key | Type | Description |
|-----|------|-------------|
| `strategist.runtime_mode` | string | Always `cli` for CLI invocations |
| `strategist.output_profile` | string | Active output profile (e.g. `default`, `epic`, `pragmatic`) |
| `strategist.target` | string | Target directory (sanitized — absolute paths are redacted) |

### Pipeline phase

| Key | Type | Description |
|-----|------|-------------|
| `strategist.phase` | string | Current phase: `intake`, `discovery`, `refinement`, `gate`, `execution` |
| `strategist.status` | string | Phase status: `running`, `done`, `blocked`, `policy_blocked` |
| `strategist.skill` | string | Skill name assigned to the current slot |
| `strategist.selected_skill` | string | Resolved skill after provider lookup |
| `strategist.artifact` | string | Artifact type being produced |
| `strategist.artifact.path` | string | Artifact path (sanitized) |

### Gate

| Key | Type | Description |
|-----|------|-------------|
| `strategist.gate.type` | string | Gate type: `approval`, `opportunity`, `adr` |
| `strategist.gate.status` | string | `pending`, `approved`, `declined`, `plan_only` |
| `strategist.gate.response` | string | Raw user response at the gate |
| `strategist.approval_policy` | string | Policy that governed this gate decision |
| `strategist.transition_group` | string | Transition group key (e.g. `finalize_analysis`) |

### Metrics (timing)

| Key | Type | Unit | Description |
|-----|------|------|-------------|
| `strategist.metrics.t_start_to_intake_ms` | int | ms | Time from pipeline start to intake complete |
| `strategist.metrics.t_intake_to_ranger_ms` | int | ms | Time from intake to Ranger start |
| `strategist.metrics.total_wall_time_ms` | int | ms | Total wall-clock time for the mission |
| `strategist.metrics.tokens_in` | int | tokens | Input tokens consumed |
| `strategist.metrics.tokens_out` | int | tokens | Output tokens produced |
| `strategist.metrics.lines_emitted` | int | lines | Total output lines emitted to the user |

### Diagnostics

| Key | Type | Description |
|-----|------|-------------|
| `strategist.reason` | string | Reason for a blocking decision or warning |
| `strategist.cache.hit` | bool | Whether a compiled artifact cache hit occurred |
| `strategist.mandates.count` | int | Number of active governance mandates |
| `strategist.checkpoint.path` | string | Path of the mission checkpoint file |

---

## Canonical event: pipeline start

Every command emits a `[Strategist] pipeline=starting` line at root `PersistentPreRunE`.

Format:
```
[Strategist] pipeline=starting mission_id=<id> profile_mode=<mode> profile_path=<path> active_yaml_path=<path> persona_resolved=<persona> reason=<reason> output=<profile>
```

Example:
```
[Strategist] pipeline=starting mission_id=install-1750000000 profile_mode=local profile_path=.strategist active_yaml_path=.strategist/active.yaml persona_resolved=unknown reason=local_default output=default
```

This line is emitted to **stdout** unconditionally, before any slog output.

---

## Event sequence per command

### `strategist install`

```
[Strategist] pipeline=starting mission_id=...
[Strategist] install starting  target=<dir>
[Strategist] install extracting-defaults  target=<dir>
[Strategist] install applying-config  wizard=<bool>
[Strategist] install writing-manifests  (wizard mode only)
[Strategist] install shim-step  target=<dir>
[Strategist] SKILL.md read from embedded FS  (when local SKILL.md absent)
[Strategist] compile warning  error=...  (non-fatal, only if compile fails)
[Strategist] install rolled back  (only on failure)
[Strategist] install complete  target=<dir>
```

### `strategist compile`

```
[Strategist] pipeline=starting mission_id=...
[Strategist] compile running  root=<dir>
[Strategist] compile complete  root=<dir> duration_ms=<n>
```

### `strategist validate`

```
[Strategist] pipeline=starting mission_id=...
[Strategist] validate complete  root=<dir>
```

---

## Mission checkpoint schema

The checkpoint file (`MissionCheckpoint`) is written atomically after each Sniper task.
Path: `.analysis/archived/<mission_id>.checkpoint.json` (managed by the CLI).

```json
{
  "mission_id": "20260620-feature-xyz",
  "tasks_total": 5,
  "tasks_completed": [1, 2, 3],
  "last_updated": "2026-06-20T18:30:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `mission_id` | string | Must match the analysis frontmatter `mission_id` |
| `tasks_total` | int | Total number of tasks in `tasks.md` |
| `tasks_completed` | []int | 1-indexed task numbers completed so far |
| `last_updated` | RFC3339 | Timestamp of last update (UTC) |

A missing checkpoint file means no tasks have been completed — not an error.

---

## Structured log event examples

### Mission metrics event (slog JSON)

Emitted at the end of each mission phase transition. Contains timing and token volume.

```json
{
  "time": "2026-06-21T14:35:02Z",
  "level": "INFO",
  "msg": "[Strategist] key=mission_metrics",
  "strategist.mission_id": "20260621-critical-hit-readme",
  "strategist.metrics.t_start_to_intake_ms": 820,
  "strategist.metrics.t_intake_to_ranger_ms": 0,
  "strategist.metrics.total_wall_time_ms": 4300,
  "strategist.metrics.tokens_in": 1840,
  "strategist.metrics.tokens_out": 620,
  "strategist.metrics.lines_emitted": 12
}
```

> `t_intake_to_ranger_ms: 0` indicates the Critical Hit route was used — Ranger was skipped.

### Gate approval event (slog JSON)

```json
{
  "time": "2026-06-21T14:35:01Z",
  "level": "INFO",
  "msg": "[Strategist] phase=approval_gate status=approved",
  "strategist.phase": "gate",
  "strategist.gate.type": "approval",
  "strategist.gate.status": "approved",
  "strategist.gate.response": "sim",
  "strategist.approval_policy": "any",
  "strategist.transition_group": "execution",
  "strategist.mission_id": "20260621-refactor-auth"
}
```

### Pipeline bypass event (slog JSON, exit code 2)

```json
{
  "time": "2026-06-21T14:35:05Z",
  "level": "ERROR",
  "msg": "[Strategist] pipeline_bypass_detected",
  "strategist.reason": "pipeline_bypass_detected",
  "strategist.phase": "approval_gate",
  "strategist.mission_id": "20260621-refactor-auth",
  "strategist.status": "blocked"
}
```

---

## CI/CD integration

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Generic error (unknown category) |
| `2` | Governance / policy violation (e.g. pipeline bypass detected) |
| `3` | Stale artifact or config integrity error |

### Health check pattern

```bash
strategist validate --root .strategist
echo "exit=$?"
```

Expected output on a valid installation:
```
[Strategist] pipeline=starting ...
[Strategist] validate complete root=.strategist
exit=0
```

### Structured log consumption

Set `OTEL_EXPORTER_OTLP_ENDPOINT` to route spans to your collector.
Filter on `strategist.phase=gate strategist.gate.status=approved` to audit approval decisions.

---

## Platform integration examples

### Datadog

```bash
# Install Datadog Agent with OTLP receiver enabled (datadog.yaml):
# otlp_config:
#   receiver:
#     protocols:
#       grpc:
#         endpoint: 0.0.0.0:4317

export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export OTEL_SERVICE_NAME=strategist
export DD_ENV=production

strategist install --wizard
```

Traces appear in **Datadog APM** under `service:strategist`. To create a monitor on governance violations:

```
@strategist.status:blocked @strategist.reason:pipeline_bypass_detected
```

### Grafana + Loki (structured logs)

```bash
# promtail config: scrape stdout and push to Loki
# job_name: strategist
#   pipeline_stages:
#     - json:
#         expressions:
#           mission_id: '"strategist.mission_id"'
#           phase: '"strategist.phase"'
#           wall_time: '"strategist.metrics.total_wall_time_ms"'

strategist compile --root .strategist 2>&1 | promtail --stdin
```

**LogQL query to detect slow missions (> 60s):**
```logql
{job="strategist"} | json | strategist_metrics_total_wall_time_ms > 60000
```

**LogQL query to audit gate approvals:**
```logql
{job="strategist"} | json | strategist_phase="gate" | strategist_gate_status="approved"
```

### Prometheus + Alertmanager

Strategist does not expose a metrics endpoint — it emits via OTLP spans. Use the [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) to bridge spans to Prometheus metrics:

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [prometheus]
```

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
strategist validate --root .strategist
```

**Example Alertmanager rule for stale config (exit code 3):**
```yaml
- alert: StrategistConfigStale
  expr: strategist_check_stale_total{cache_hit="false"} > 0
  for: 5m
  annotations:
    summary: "Strategist compiled artifacts are stale — run strategist compile"
```

---

## What ADR-0010 guarantees

ADR-0010 (`docs/adr/0010-ordered-contracts-and-mission-observability.md`) establishes:

- Contracts are read in a canonical numbered order (00–10)
- Each phase emits `phase=<name> status=running` before delegating and `status=done` on completion
- The response contract (`09-response.md`) defines the input/output shape for `mission_envelope.close`
- Checkpoint events are OTEL-compatible (attribute keys match this document)

ADR-0010 does **not** guarantee:
- Specific token counts (model-dependent)
- Timing attributes in non-CLI modes
- Checkpoint file presence when a mission is in `plan_only` mode
