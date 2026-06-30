# CLI Reference — strategist

**Status:** Accepted
**Last Updated:** 2026-06-26

The `strategist` binary is built in Go with [cobra](https://github.com/spf13/cobra). All commands follow the pattern:

```
strategist <command> [flags]
```

---

## install

Installs the Strategist skill in a target repository.

```
strategist install [--target=<dir>] [--wizard] [--silent]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--target` | `.` (current directory) | Repository root where `.strategist/` will be created |
| `--wizard` | `false` | Interactive mode: collects mode, base_path, and provider via prompts |
| `--silent` | `false` | Installation without prompts using pragmatic defaults (default behavior when no flag is passed) |

**What it does:**

1. Extracts embedded defaults to `<target>/.strategist/`
2. Generates `active.yaml` (wizard or pragmatic template)
3. Adds `.strategist/.compiled/` to `.gitignore`
4. Installs the shim at `~/.claude/skills/strategist/SKILL.md`
5. Compiles all artifacts to `.strategist/.compiled/`

**Rollback:** if any step fails, created files are removed and the workspace is restored to its previous state.

**Examples:**

```bash
# Install with wizard in the current directory
strategist install --wizard

# Silently install in another repository
strategist install --target=/path/to/project

# Via bootstrap (recommended for first installation)
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash
```

**Success output:**
```
[Strategist] install complete → .
```

---

## compile

Compiles all skill YAML artifacts to gzip+JSON.

```
strategist compile [--root=<dir>]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--root` | `.strategist` | Path to the `.strategist/` root |

**Artifacts generated in `<root>/.compiled/`:**

| File | Content |
|------|---------|
| `.index.gz` | Compiled `knowledge.index.yaml` |
| `.domain.gz` | Compiled domain templates (`templates/domain/`) |
| `.config.gz` | Compiled `active.yaml` + `personas/` + `roles/` |
| `.manifest.gz` | SHA256 of the 3 artifacts above |

Run after any manual edits to YAML configuration files.

**Success output:**
```
[Strategist] compile complete → .strategist/.compiled/
```

---

## check-stale

Checks whether a compiled artifact is stale relative to its YAML sources.

```
strategist check-stale <artifact.gz>
```

**Argument:** path to a `.gz` file in `.strategist/.compiled/`.

**Exit codes:**

| Code | Meaning |
|------|---------|
| `0` | Artifact is fresh — sources were not modified |
| `1` | Artifact is stale — at least one source was modified, or the artifact/manifest does not exist |

**Designed for use in CI/scripts:**

```bash
if ! strategist check-stale .strategist/.compiled/.config.gz; then
  strategist compile
fi
```

An artifact is considered stale when:
- The `.gz` file does not exist
- `.manifest.gz` does not exist in the same directory
- Any source listed in `artifact.sources` was modified after compilation
- Any listed source no longer exists on disk

---

## validate

Validates the `.strategist/` configuration tree.

```
strategist validate [--root=<dir>]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--root` | `.strategist` | Path to the `.strategist/` root |

**Checks performed:**

| File | What is checked |
|------|----------------|
| `active.yaml` | Exists, valid YAML, `mode` and `roles_config` fields present, `mode` is `pragmatic` or `epic` |
| `personas/*.yaml` | Each file has `tone_directive` and `phase_labels` |
| `roles/*.yaml` | Each file has the `discovery`, `refinement`, and `execution` slots |
| `knowledge.index.yaml` | If present, valid YAML |

**Success output:**
```
[Strategist] validate OK — 7 check(s) passed (.strategist)
```

**Failure output:**
```
  ✗ active.yaml: invalid mode "custom" (must be pragmatic or epic)
  ✗ roles/custom.yaml: missing slot: execution
validate: 2 error(s) in .strategist
```

Useful in CI to ensure that manual configuration edits have not introduced schema errors.

---

## version

Displays the binary version.

```
strategist version
```

The version is injected at build time via `-ldflags "-X main.Version=x.y.z"`. In local builds without ldflags, displays `strategist dev`.

**Output:**
```
strategist v1.0.0
```

---

## check

Validates operational readiness of the Strategist runtime — confirms the skill is installed, providers are configured, and persona is valid. Use before starting a mission.

`check` does **not** test whether the environment can invoke external agents. It confirms the runtime is installed and configured. If a slot provider fails to be invoked during a mission, Strategist reports `role_invocation_failed` as an internal skill error — not a `check` failure.

```
strategist check [--root=<dir>]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--root` | `.strategist` | Path to the `.strategist/` root |

**Checks performed:**

- `active.yaml` present and parseable
- For each slot (`discovery`, `refinement`, `execution`):
  - `skills/<provider>/skill.yaml` exists (provider skill), **or** `roles/<provider>.yaml` exists with the slot field (native role)
  - Provider skills must declare the correct `risk_score`: `discovery`/`refinement` → `write_analysis`; `execution` → `controlled`
  - Native roles are accepted by field match; no `risk_score` verification
- Active persona file exists and contains required fields
- Normative runtime files match embedded defaults (detects stale installs)

**Success output:**
```
STATUS   
  ok   root=.strategist

SLOTS   
  discovery    brainstorming
  refinement   openspec-explore
  execution    sniper

PERSONA   
  mode   epic
```

**Failure output:**
```
  ✗ slot execution: provider "sniper" not installed (missing .strategist/skills/sniper/skill.yaml)
[Strategist] check=failed errors=1 root=.strategist
```

---

## initiative

Displays the configured slot providers and the current workspace state. Immediate read with no LLM call.

```
strategist initiative [--root=<dir>]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--root` | `.strategist` (auto-discovered) | Path to the `.strategist/` root |

**Output:**

```
SLOTS                                                  
discovery      brainstorming      Ranger rankeado      ✓ manifest OK
refinement     openspec-explore   Archivist rankeado   ✓ manifest OK
execution      sniper             Sniper (base)        ✓ manifest OK
                                                       
WORKSPACE                                              
mode           epic                                    
base_path      .analysis                               
pending        0 cards                                 
done           49 missions                             
last mission   —                                       
```

The **SLOTS** section displays, for each slot: configured provider, canonical role, class (`rankeado` or `base`), and status of the local manifest at `.strategist/skills/<provider>/skill.yaml`.

The **WORKSPACE** section displays: `mode` and `base_path` from `active.yaml`, counts of pending cards and completed missions, and the ID of the last mission recorded in `memory/outcomes.jsonl` (if present).

---

## dojo

Health-check system for the Strategist skill — validates that the skill is installed, configured, and operating correctly.

```
strategist dojo check <scenario> [--root=<dir>] [--files-only]
strategist dojo list
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `dojo check <scenario>` | Runs offline checks for a scenario |
| `dojo list` | Lists available scenarios |

**Flags for `dojo check`:**

| Flag | Default | Description |
|------|---------|-------------|
| `--root` | `.strategist` | Path to the `.strategist/` root |
| `--files-only` | `false` | Skips `emit_log` validation; checks files only |

**Available scenarios** (via `strategist dojo list`):

| Scenario | What it validates |
|----------|------------------|
| `critical-hit` | Doc edit via fast path — Ranger and Archivist not invoked, inline gate presented, Sniper writes only the target file |
| `quick-draw` | Raw idea converted to a pending todo item, gate presented, execution not invoked |
| `ranger-weapons` | Lists available providers for the discovery slot and validates manifests |
| `treasure-chest` | Treasure chest found and content incorporated in the analysis |

**Example:**

```bash
# Validate a scenario offline
strategist dojo check quick-draw

# Check files only (no emit log)
strategist dojo check quick-draw --files-only

# List available scenarios
strategist dojo list
```

For full pipeline execution with synthetic input, use the `/strategist dojo <scenario>` skill via Claude Agent. See `docs/strategist-concepts.md#dojo`.

---

## treasure-chest

Displays the status of configured treasure chests, governance policies, and compiled knowledge index health.

```
strategist treasure-chest [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--root` | `.strategist` (auto-discovered) | Path to the `.strategist/` root |
| `--scope` | `""` | Filters output by slot scope (`discovery`, `refinement`, `execution`) |
| `--index` | `false` | Rebuilds the compiled knowledge index from declared sources |
| `--include-historical` | `false` | Includes historical T2/T3 sources in the rebuild (requires `--index`) |
| `--format` | `table` | Output format: `table` or `json` |

**Sources consulted:**

- `.strategist/active.yaml` — configured chests and their scopes
- `.strategist/treasure-chests.yaml` — trust and routing policies
- `.strategist/knowledge.index.yaml` — indexed retrieval sources
- `.strategist/.compiled/.index.gz` — compiled artifact (fast-path)

**Example output:**

```
CHESTS                                             
ID       PATH          SCOPE   TRUST   FRESHNESS   DRIFT
source   .sdd/source   all     T1      unknown     none

INDEX                                                       
artifact      .strategist/.compiled/.index.gz               
health        ok                                            
compiled_at   2026-06-26 18:19:47 UTC                       
```

---

## sync-governance

Synchronizes `.strategist/skill.yaml` with the active SDD governance mandates.

```
strategist sync-governance [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--root` | `.strategist` | Path to the `.strategist/` root |
| `--sdd` | `.sdd` | Path to the `.sdd/` directory |
| `--dry-run` | `false` | Displays changes without writing |

**What it does:**

1. Reads `.sdd/metadata.json` to verify the governance fingerprint
2. Reads `.sdd/source/governance-core.json` to extract active mandates
3. Compares active mandates against `compliance.mandates` in `skill.yaml`
4. Applies missing governance fields (`validation_policy`, `budget_policy`, `telemetry_policy`)
5. Reports drift before applying changes

**Example:**

```bash
# Check drift without writing
strategist sync-governance --dry-run

# Apply synchronization
strategist sync-governance
```

Requires `.sdd/` to be present in the repository (SDD governance). Without `.sdd/`, the command returns an error.

---

## Observability (OpenTelemetry)

All commands emit OTel spans when a collector is configured. Without configuration, the binary uses a no-op provider — zero overhead and zero open network connections.

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `""` | Collector gRPC endpoint (e.g. `localhost:4317`). Empty → no-op. |
| `OTEL_SERVICE_NAME` | `strategist` | Service name in traces. |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true` | TLS disabled by default. In production: `false`. |

**Example with local collector:**

```bash
# Start Jaeger all-in-one (accepts gRPC on port 4317)
docker run -d -p 16686:16686 -p 4317:4317 jaegertracing/all-in-one

# Run with OTel enabled
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_SERVICE_NAME=strategist \
strategist install --target .

# View traces at http://localhost:16686
```

**Span attributes:**

| Span | Attributes |
|------|-----------|
| `strategist.install` | `strategist.target` |
| `strategist.compile` | `strategist.target` |
| `strategist.check_stale` | `strategist.artifact`, `strategist.cache.hit` |
| `strategist.sync_governance` | `strategist.mandates.count`, `strategist.mandates.missing` |
| `strategist.check` | `strategist.target` |
| `strategist.initiative` | `strategist.target` |

---

## Exit codes

All commands return a standardized exit code. Useful for CI/CD and scripts.

| Code | Meaning | Example cause |
|------|---------|--------------|
| `0` | Success | Command completed without errors |
| `1` | Generic / unknown error | Invalid YAML, file not found |
| `2` | Governance / policy violation | Pipeline bypass detected without approval |
| `3` | Stale artifact or config integrity error | `.compiled/` out of date, manifest missing |

**Example in script:**

```bash
strategist validate --root .strategist
code=$?

case $code in
  0) echo "OK";;
  2) echo "Governance violation — check pipeline state" >&2; exit 1;;
  3) echo "Config stale — run: strategist compile" >&2; exit 1;;
  *) echo "Error ($code)" >&2; exit 1;;
esac
```

**Example in CI (GitHub Actions):**

```yaml
- name: Validate strategist config
  run: strategist validate --root .strategist
  # Exits 2 if governance bypassed, 3 if compiled artifacts are stale
```

---

## Local installation (build from source)

```bash
# Clone and build
git clone https://github.com/SergioLacerda/strategist-skill
cd strategist-skill

# Build
make build          # → bin/strategist

# Install to PATH (~/.local/bin/)
make install-local  # equivalent to: install -m 755 bin/strategist ~/.local/bin/strategist

# Ensure ~/.local/bin is in PATH
export PATH="$HOME/.local/bin:$PATH"

# Verify
strategist version
```
