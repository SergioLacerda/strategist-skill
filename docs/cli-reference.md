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
strategist install [--target=<dir>] [--wizard] [--silent] [--force]
                    [--strict-compile] [--no-shim | --shim-path=<path>]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--target` | `.` (current directory) | Repository root where `.strategist/` will be created |
| `--wizard` | `false` | Interactive mode: collects mode, base_path, and provider via prompts |
| `--silent` | `false` (default behavior when no flag is passed) | Installation without prompts, using **epic** profile defaults |
| `--force` | `false` | Overwrite all files, including user-modified ones (default: preserve customizations that differ from the embedded default) |
| `--strict-compile` | `false` | Make a `CompileAll` failure after extraction fatal — the install rolls back instead of completing with a partial/uncompiled runtime. Default is warning-only (install still completes) |
| `--no-shim` | `false` | Skip writing the SKILL.md shim entirely — no write to `~/.claude/skills` at all. Useful for CI/containers without a writable home directory. Mutually exclusive with `--shim-path` |
| `--shim-path` | `` (default: `~/.claude/skills/strategist/SKILL.md`) | Write the shim to this path instead of the default home-relative location. Mutually exclusive with `--no-shim` |

**What it does:**

1. Extracts embedded defaults to `<target>/.strategist/`
2. Generates `active.yaml` (wizard or epic template)
3. Adds `.strategist/.compiled/` to `.gitignore`
4. Installs the shim at `~/.claude/skills/strategist/SKILL.md`, unless `--no-shim` or `--shim-path` is set
5. Compiles all artifacts to `.strategist/.compiled/` — a failure here is warning-only by default, fatal with `--strict-compile`

**Rollback:** if any step fails, the install is rolled back. On a **fresh** install (no pre-existing `.strategist/`), Strategist owns the entire tree and removes it wholesale (`rm -rf .strategist/`), plus any `.gitignore`/shim entries it added. On a **re-install over an existing tree** (`--force`), pre-existing content is never deleted — only the specific entries added during that run (e.g. a new `.gitignore` line or shim file) are rolled back.

Only `env/profile`-based shim precedence is out of scope for now — CLI flags (`--no-shim`/`--shim-path`) are the only supported override today.

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
| `personas/*.yaml` | Each file satisfies the same runtime contract `check` enforces: `id`, `tone_directive`, `phase_labels.{discovery,refinement,execution}`, `diagnostics.pipeline_header`, `diagnostics.bootstrap_origin` |
| `roles/*.yaml` | A native role definition (has a `role` key) must have `role` and a `slot` that is one of `discovery`/`refinement`/`execution`. A slot map (e.g. `roles/default.yaml`, shaped like `active.yaml`'s `slots:`) must have all three slots present and non-empty |
| `knowledge.index.yaml` | If present, valid YAML |

Go structs/validators (`domain.PersonaConfig`, `domain.RoleConfig`, `domain.RoleSlotMap`) are the authoritative source of truth for these rules — the `.schema.yaml` files under `schemas/` are descriptive documentation only and are not loaded or enforced by the CLI.

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
strategist check [--root=<dir>] [--strict] [--simulate]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--root` | `.strategist` | Path to the `.strategist/` root |
| `--strict` | `false` | Additionally require `.compiled/{.index,.domain,.config,.manifest}.gz` to exist and match the recorded manifest SHA256 hashes. Composes on top of the base checks below — never weakens them |
| `--simulate` | `false` | Print a `READINESS` report (per-slot/persona status, plus `pipeline_route`/`decision_reason`) instead of the terse pass/fail banner. Performs no provider invocation and no workspace mutation — same read-only guarantee as plain `check` |

**Checks performed:**

- `active.yaml` present and parseable
- For each slot (`discovery`, `refinement`, `execution`):
  - `skills/<provider>/skill.yaml` exists (provider skill), **or** `roles/<provider>.yaml` exists with the slot field (native role)
  - Provider skills must declare the correct `risk_score`: `discovery`/`refinement` → `write_analysis`; `execution` → `controlled`
  - Native roles are validated against `domain.RoleConfig` (required `role` + valid `slot`), then accepted by slot match; no `risk_score` verification
- Active persona file exists and contains required fields
- Normative runtime files match embedded defaults (detects stale installs)
- With `--strict`: compiled artifacts exist and match the recorded manifest hashes (see `compile`)

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

**`--simulate` output:**
```
READINESS
  root             .strategist
  pipeline_route   main
  decision_reason  all_slots_ready

SLOTS
  discovery    provider=brainstorming       status=ready
  refinement   provider=openspec-explore    status=ready
  execution    provider=sniper              status=ready

PERSONA
  mode   epic
```

Note: `--simulate` reports readiness for the CLI-known `main` pipeline route only. Quick-draw and critical-hit routing — and Scout's route classification generally — are prompt-time decisions made by the LLM runtime from `contracts/narrative/00-routing.md` and `contracts/machine/scout-routing.yaml`, and are not simulated here — `check` intentionally does not take over mission routing.

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

### `treasure-chest add` / `treasure-chest remove`

Registry mutation commands (Track T-I, implemented in `SQ-006`). Update `active.yaml`,
`treasure-chests.yaml`, and `knowledge.index.yaml` together via `yaml.Node` round-tripping, so
existing comments and formatting in those files are preserved rather than overwritten.

```
strategist treasure-chest add "path" [--id <id>] [--scope all|discovery|refinement|execution] [--trust-tier T0|T1|T2|T3] [--reviewed-by human|auto] [--tags <tag>,...] [--index]
strategist treasure-chest remove ["path"] [--id <id>]
```

**`add`:**

- `--id` defaults to the last non-empty path segment; explicit `--id` overrides it.
- `--scope` defaults to `all`.
- `--trust-tier` defaults to `T1`; `--reviewed-by` defaults to `human`.
- `--tags` is optional, comma-separated; defaults to `[all]`.
- Fails if `--id` (or the derived id) is already registered in `active.yaml` — use a different
  `--id` or `remove` the existing entry first.
- Updates `active.yaml` (`treasure_chests[]`), `treasure-chests.yaml` (`chests[]`), and
  `knowledge.index.yaml` (`sources[]`) together — if a write fails partway through, the error
  reports exactly which files were already written so state can be reconciled manually.
- `.compiled/.index.gz` is left stale with an explicit warning unless `--index` is also passed.

**`remove`:**

- Resolves by positional `path` or `--id`. If both are given and disagree (same id different
  path, or vice versa), or if a path matches multiple ids, the command rejects with an explicit
  ambiguity error rather than guessing.
- Tombstones rather than hard-deletes: removes the entry from `active.yaml`'s
  `treasure_chests[]` (the "active" declaration), and sets `status: inactive` on the matching
  entry in `treasure-chests.yaml` and `knowledge.index.yaml` — the entries stay for audit
  history instead of being deleted.
- Reports a stale-index warning identical in spirit to `add`'s.

**Registry layers touched:** see [Registry Layers](configuration.md#registry-layers) in the
configuration reference — `add`/`remove` keep configured, governed, and indexed layers
consistent, and only ever touch the compiled layer through explicit `--index` or an explicit
stale warning.

### `treasure-chest index` / `treasure-chest mine`

The offline organization plane exposes exactly two public, steady-state commands (Track:
`treasure-chest-index-mine-pipeline`). `scan` is folded into `index` as an internal phase —
it remains callable directly (`strategist treasure-chest scan`, hidden from `--help`) for
debugging/dry-run inspection, but is no longer documented UX. There is no `gaps` or `pack`
command; Evidence Packs already exist as mission artifacts (Track T-A), generated
automatically by `dossier-builder`, and open gaps surface through `index`'s internal scan
phase into `.strategist/treasure/gaps/` and `status:proposed` jewels.

```
strategist treasure-chest index [--include-historical]
strategist treasure-chest mine --list [--format table|json]
strategist treasure-chest mine --accept <jewel-id>[,<jewel-id>...]
strategist treasure-chest mine --verify <jewel-id>[,<jewel-id>...] --evidence <ref>
strategist treasure-chest mine --deprecate <jewel-id>[,<jewel-id>...]
strategist treasure-chest mine --migrate-status
strategist treasure-chest jewel list [--status all|proposed|accepted|verified|deprecated] [--chest <chest-id>] [--format table|json]
strategist treasure-chest jewel show <jewel-id> [--format table|json]
strategist treasure-chest jewel accept <jewel-id>...
strategist treasure-chest jewel verify <jewel-id>... --evidence <ref>
strategist treasure-chest jewel deprecate <jewel-id>...
```

**`index`** rebuilds the offline knowledge substrate:

1. Runs the internal scan phase over `<base_path>/refined/**/tasks.md` and
   `<base_path>/done/**` (lexical/tag matching only — no embeddings, no vector index),
   writing `.strategist/treasure/clusters/` and `.strategist/treasure/gaps/` exactly as the
   former standalone `scan` command did.
2. Polishes each cluster/gap into a deduplicated `status: proposed` jewel candidate
   (`kind: pattern` for clusters, `kind: gap` for gaps), scored 0-100 as a ranking/economy
   hint only — never a promotion authority. Candidates use the virtual
   `chest_id: mission-history` since they are derived from mission history, not a specific
   treasure chest's own content.
3. Deduplicates against existing monolithic `jewels.yaml` and partitioned
   `jewels/<chest-id>.yaml` entries by `id` — a rerun never overwrites or duplicates a
   jewel, curated or not.
4. Rebuilds the compiled knowledge index (`.strategist/.compiled/.index.gz`), same as the
   legacy `--index` flag on the base `treasure-chest` command.
5. Reports: missions scanned, candidates found, proposed jewels written, duplicates
   skipped, compiled artifact refreshed.

Score generation is configurable via optional `scoring_policy` in `treasure-chests.yaml`.
Defaults preserve the original formula:

- cluster score = `40 + missions * 10 + tags * 5`
- gap score = `30 + missions * 15`
- both are capped at `100`

**`mine`** is the human curation command over `status: proposed` jewels — exactly one action
flag is required per invocation:

- `--list [--format table|json]` — lists only `status: proposed` jewels (the curation
  queue), grouped/sorted by chest then id, with kind/trust/score/statement.
- `--accept <jewel-id>[,<jewel-id>...]` — promotes one or more jewels to `status: accepted`;
  sets `reviewed_by: human` and `last_reviewed` to today.
- `--verify <jewel-id>[,<jewel-id>...] --evidence <ref>` — promotes one or more jewels to
  `status: verified`; requires `--evidence`, appended to `verification.evidence_refs`.
- `--deprecate <jewel-id>[,<jewel-id>...]` — marks one or more jewels as `status: deprecated`.
  Deprecation is terminal: a
  deprecated jewel can never be promoted back to `accepted`/`verified`.
- `--migrate-status` — see Migration below.

**`jewel`** is the read-only inspection surface over all jewels regardless of status —
unlike `mine --list` (scoped to the `status: proposed` curation queue only):

- `list [--status all|proposed|accepted|verified|deprecated] [--chest <chest-id>] [--format table|json]`
  — without `--status`, shows `proposed` + `accepted` + `verified` (excludes
  `deprecated`); `--status all` includes `deprecated`; `--chest` filters by chest id,
  combinable with `--status`. Sorted by `(chest_id, id)`, same as `mine --list`.
- `show <jewel-id> [--format table|json]` — prints every field of a single jewel
  (`statement`, `source_refs`, `trust`, `score`, `applicability`, `verification`,
  etc.). Unknown id: error, non-zero exit.
- `accept <jewel-id>...`, `verify <jewel-id>... --evidence <ref>`, and
  `deprecate <jewel-id>...` — curation commands equivalent to the legacy `mine` flags,
  accepting either repeated positional ids or comma-separated ids.

**`treasure-chest scan` contract** (internal phase, folded into `index`; originally Track
T-F / `SQ-003`, defined in mission `bau-tesouro-sq003-004-007`, implemented in
`bau-tesouro-sq010-scan-runtime`):

- **Input scope**: `.analysis/refined/**/tasks.md` and `.analysis/done/**` only —
  Archivist-reviewed or closure-validated content. Excludes `.analysis/pending/scraps`
  (unreviewed capture) and raw `.analysis/archived/*-report.md` files (Sniper completion
  reports, not analysis content).
- **Method**: lexical/tag matching only — no embeddings, no vector index (semantic/vector
  retrieval remains explicitly forbidden per the parent mission's scope).
- **Trust default**: `T2` (`examples` tier), matching `treasure-chests.yaml`'s existing
  semantics for "previous missions."
- **Output**: `.strategist/treasure/clusters/<cluster-id>.md` and
  `.strategist/treasure/gaps/<gap-id>.md` — see
  [Storage Domain](configuration.md#storage-domain-track-t-h--sq-004--contract-only-not-implemented)
  in the configuration reference.

### Jewels

Jewels (Track T-J / `SQ-007`, schema defined in mission `bau-tesouro-sq003-004-007`,
implemented in `SQ-009`; lifecycle statuses revised in the
`treasure-chest-index-mine-pipeline` mission, see [ADR-0012](adr/0012-jewel-lifecycle-statuses.md))
are compact, source-linked knowledge units — children of the specific chest they were
extracted from (or, for `index`-generated candidates, the virtual `mission-history` chest).

**Gate revision note:** the original design required human pre-approval before a jewel could
be usable (`candidate → reviewed → active`). At the originating mission's Approval Gate, that
was revised: the agent analyzing a chest generates a jewel immediately (`reviewed_by: agent`),
with a **trust-tier ceiling** as the safeguard instead of a pre-approval step — a jewel's
`trust` can never exceed its parent chest's `trust.tier`. This relaxes the
`20260711-bau-tesouro-upgrade` mission's `forbidden` clause "Any generated jewel promotion
without explicit review/approval," scoped narrowly to this mechanism only — see
[ADR-0011](adr/0011-jewel-promotion-trust-ceiling-exception.md) for the full scope
statement. Enforced at runtime by `ValidateJewelTrust` (`internal/domain/jewel_grade.go`).

```yaml
# .strategist/jewels/<chest-id>.yaml — created and populated at runtime
schema_version: "1"
jewels:
  - id: <identifier>
    chest_id: <parent chest id>          # mandatory — jewel is a child of exactly one chest
    kind: decision | pattern | anti_pattern | gap | risk | constraint | example | heuristic | template | question
    statement: <compact fact/pattern/decision, source-linked>
    source_refs:                         # mandatory — at least one
      - <chest-id>#<section-slug>
    trust: T0 | T1 | T2 | T3             # mandatory; MUST be <= parent chest's trust.tier
    status: proposed | accepted | verified | deprecated
    reviewed_by: agent | human
    last_reviewed: YYYY-MM-DD | null
    score: { value: 0-100, reasons: [<short reason>, ...] }
    applicability: { scope: [...], applies_when: [...], avoid_when: [...] }
    verification: { evidence_refs: [...] }
    history:
      - status: proposed | accepted | verified | deprecated
        at: YYYY-MM-DD
        by: agent | human
        evidence_ref: <ref>              # present when status: verified was evidence-backed
```

**Lifecycle:** agent- or `index`-generated jewels always start at `status: proposed` — an
agent must never write `accepted` or `verified` directly. Only `treasure-chest mine` (human
curation) promotes a jewel to `accepted` or `verified` (the latter requires an evidence ref).
`deprecated` is reached manually via `mine --deprecate`, or automatically when the parent
chest is tombstoned via `treasure-chest remove` (the chest removal is already an explicit
human action). Deprecation is terminal.

**Lifecycle history:** newly proposed jewels and every later status mutation append a
`history` entry. Older jewels without `history` remain valid and gain history on their next
curation or deprecation event.

**Migration (`active` → `accepted`):** the pre-`treasure-chest-index-mine-pipeline` schema
used a two-state `active | deprecated` model. `active` is a **removed legacy status** —
`loadJewels` fails loudly (not a silent fallback) on any remaining `active` entry, per
`ValidateJewelStatus` (`internal/domain/jewel_grade.go`). Run
`strategist treasure-chest mine --migrate-status` once to rewrite every `status: active`
entry to `status: accepted` in place across monolithic and partitioned manifests; the
command is idempotent and reports how many entries it migrated (0 is a valid, non-error
outcome).

**Implemented**: `loadJewels` accepts legacy `.strategist/jewels.yaml` and partitioned
`.strategist/jewels/<chest-id>.yaml`; new `index` candidates are written to partitioned
manifests. Non-deprecated jewel counts are shown in the `treasure-chest` list's `JEWELS`
column and JSON output (`cmd/strategist/treasure_chest.go`); removing a chest cascades to
mark its jewels `deprecated` across both layouts (`markJewelsDeprecatedForChest` in
`internal/treasure/yaml_node.go`); the `jewel_generation` and `jewel_retrieval` contract blocks
govern LLM-facing generation/retrieval behavior
(`internal/embed/defaults/contracts/machine/context-enrichment.yaml`), including
status-precedence retrieval (`verified` preferred, then `accepted`, `proposed` as hint only,
`deprecated` excluded); `treasure-chest jewel list`/`jewel show`
(`cmd/strategist/treasure_chest_jewel.go`) expose all jewels regardless of status for
inspection, independent of `mine`'s curation queue.

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
