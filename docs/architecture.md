# Architecture — Strategist Skill

**Status:** Accepted
**Last Updated:** 2026-08-03

## Overview

The project is composed of two independent layers:

| Layer | Location | Responsibility |
|-------|----------|----------------|
| **Go Binary** | `cmd/` + `internal/` | Install, compile, and validate skill artifacts |
| **Runtime source** | `internal/embed/defaults/` | Single authoring tree embedded into the binary (`go:embed`); generates the runtime package |
| **Runtime instance** | `.strategist/` | Operational instructions read by the agent: pipeline, slots, personas, contracts |

The binary **does not execute missions**. It prepares the environment so the agent can run the skill correctly. During a mission, the agent reads `.strategist/`; the retired root `strategist/` authoring mirror is not a current build, documentation, or runtime source.

---

## Go Package Map

```
cmd/strategist/          CLI commands (cobra)
  main.go                Entrypoint; initializes OTel + calls execute()
  root.go                Registers all subcommands; injects context via PersistentPreRunE
  install.go             strategist install
  compile.go             strategist compile
  check_stale.go         strategist check-stale
  validate.go            strategist validate
  sync_governance.go     strategist sync-governance
  version.go             strategist version
  check.go               strategist check
  dojo.go                strategist dojo
  treasure_chest.go      strategist treasure-chest
  root_discovery.go      root-level .strategist/ discovery (CWD walk)
  runtime_profile.go     output profile resolution (default/epic/pragmatic)

internal/
  domain/                Core types and interfaces (ports)
    types.go             CompiledConfig, CompiledDomain, CompiledIndex,
                         CompiledManifest, InstallConfig, WizardConfig
    ports.go             Interfaces: Installer, Compiler, StaleChecker,
                         FileExtractor
    errors.go            Domain sentinel errors

  embed/                 Defaults embedded in the binary
    defaults.go          embed.FS with all defaults/ files
    defaults/            The authoring source itself (SKILL.md, roles,
                         personas, schemas, contracts, templates)

  install/               Installation logic
    installer.go         Service.Install — orchestrates extract → config → gitignore → shim → compile
    wizard.go            Interactive wizard (collects mode, base_path, provider)
    active_yaml.go       Generates active.yaml from WizardConfig
    template.go          Copies template to active.yaml (silent mode)
    gitignore.go         ensureGitignore — adds .strategist/.compiled/ to .gitignore
    shim.go              Installs ~/.claude/skills/strategist/SKILL.md

  compile/               YAML artifact compilation → gzip+JSON
    all.go               Compiler.CompileAll — orchestrates index → domain → config → manifest
    config.go            Config() — active.yaml + personas/ + roles/ → .config.gz
    domain.go            Domain() — templates/domain/ → .domain.gz
    index.go             Index() — knowledge.index.yaml → .index.gz
    helpers.go           writeGzJSON, loadYAMLFile, mtime, sha256Artifact
    yaml.go              Internal YAML helpers

  dojo/                  Offline scenario/checker domain for contract and pipeline validation
    checker.go           Evaluates criteria against a workspace root
    checker_manifest.go  Validates provider/runtime manifest structure
    checker_pipeline.go  Detects pipeline and gate violations
    learning.go          Writes non-blocking scenario learning output

  stale/                 Stale artifact detection
    check.go             Checker.IsStale — compares source mtimes against values recorded in the artifact

  telemetry/             OTel provider setup and convenience helpers
    config.go            Config{Endpoint,ServiceName,Insecure} + FromEnv() — reads standard OTel env vars
    setup.go             Init(cfg) — registers TracerProvider (real or noop); bridges slog→OTel
    tracer.go            Tracer() trace.Tracer — accesses the package-level global tracer
    schema.go            Attribute constants: strategist.phase, strategist.cache.hit, etc.
                         No-op automatic when OTEL_EXPORTER_OTLP_ENDPOINT is not set.

  governance/            Synchronizes Strategist manifests with SDD governance metadata
  i18n/                  Language selection, reserved-term checks, and localized CLI strings
  integrity/             Runtime config integrity lock and warning support
  runtimefs/             Shared filesystem primitives for runtime artifact IO
  treasure/              Treasure Chest, jewel, and potion indexing/loading/mutation logic
  validate/              Runtime validation entry points

  testutil/              Shared test helpers
    testutil.go          MinimalRoot, temporary directory fixtures
```

---

## Installation Flow

```
strategist install [--wizard] [--target=<dir>]
        │
        ▼
embed.Extractor.Extract(strategistDir)
  └─ copies internal/embed/defaults/* → <target>/.strategist/
        │
        ▼
applyConfig(strategistDir, cfg)
  ├─ [silent] copyTemplate("templates/pragmatic-standalone.yaml") → active.yaml
  └─ [wizard] runWizard(stdin) → writeActiveYAML(strategistDir, wc)
        │
        ▼
ensureGitignore(target)
  └─ adds ".strategist/.compiled/" to .gitignore (creates if absent)
        │
        ▼
installShim(target)
  └─ writes ~/.claude/skills/strategist/SKILL.md
        │
        ▼
compile.Compiler.CompileAll(.strategist/, knowledge.index.yaml)
  └─ generates .strategist/.compiled/{.index.gz, .domain.gz, .config.gz, .manifest.gz}
```

**Automatic rollback:** if any step fails, `Install` removes all created files in reverse order (`manifest []string`). Non-empty directories are left intact.

### Flow notes (recent features)

- **Pipeline sequence**: Scout (pre-pipeline, internal) classifies the request and
  selects a route. Only the `full_pipeline` route reaches the three-slot chain, which
  remains `Ranger -> Archivist -> Approval Gate -> Sniper`. Scout never replaces a
  slot and is not itself a slot — see `docs/strategist-concepts.md` § Scout —
  Intake Router.
- **Scout route decision**: Scout (Intake Router) classifies the request immediately after intake and emits a compact `route_decision` (role, selected_route, route_reason, route_confidence, evidence_state, discovery_subtype, fallback_route, gate_required) — logged and telemetered, never written as a `pending/` artifact. See `schemas/scout-route-decision.schema.yaml` and `contracts/machine/scout-routing.yaml`. Runtime callers can validate the selected route against request facts with `domain.ValidateRouteDecision`.
- **Provider-fallback degradation events (ADR-0028)**: when a configured slot provider fails invocation and the agent applies the `ask`(confirmed)/`native` fallback policy, it appends a `fallback_decision` record (mission_id, slot, policy, outcome, configured/effective provider, reason) to `.strategist/memory/fallback-decisions.jsonl` via `internal/telemetry.AppendFallbackDecisionLine`, in addition to the narrative degradation log line. See `schemas/fallback-decision.schema.yaml` and `contracts/machine/provider-fallback.yaml`. `domain.ValidateFallbackDecision` cross-checks a claimed outcome against `domain.DecideSlotFallbackOutcome`'s policy table; `strategist metrics fallback` reports `auto_native_rate`/`ask_confirmed_rate` over the recorded history.
- **Canonical discovery handoff**: Ranger produces `<base_path>/pending/<mission_id>-analysis.md`; Archivist consumes that artifact and promotes the refined package to `<base_path>/refined/<mission_id>/`.
- **Documentation-only execution**: Sniper maintains the executor narrative, but its current execution is materialization of documentation, diagrams, analyses, and approved handoffs. Source code changes are outside the contract.
- **Opportunity Attack (`opportunity_attack`)**: Archivist-owned ADR evaluation after all four refined artifacts (`analysis.md`, `proposal.md`, `design.md`, `tasks.md`) are written.
- **Critical Hit**: analysis file management route for moving `.md` artifacts within `pending/`, `refined/`, and `archived/` inside `<base_path>`.
- **Side Quests**: cross-phase scope observations; Ranger, Archivist, and Sniper may detect them; Archivist consolidates pre-execution findings at the gate; Sniper reports newly discovered side quests.
- **Treasure Chest knowledge flow (`treasure_chests`)**: full documents in a configured
  chest are the source of truth. `strategist treasure-chest index` scans them offline
  and writes deduplicated `status: proposed` jewels and potions (compact, source-linked
  knowledge points) without altering the canonical pipeline.
  `strategist treasure-chest items` is the separate human curation step that promotes
  `proposed` items to `accepted` or `verified` (or marks them `deprecated`) — see
  [Jewels](cli-reference.md#jewels). At runtime, any role may consult `accepted`/
  `verified` jewels first as a token-economical hint, then fall back to expanding the
  full source document through a source card when the jewel alone is insufficient
  evidence. Evidence Packs remain separate mission artifacts (not treasure-chest
  runtime state) generated by `dossier-builder` — see
  [Storage Domain](configuration.md#storage-domain-track-t-h--sq-004--contract-only-not-implemented).

### Canonical human contracts

The human runtime of the skill should no longer depend on diffuse reading across loose files.
The runtime entry point is `.strategist/SKILL.md`, which routes to the numbered sequence in
`.strategist/contracts/`. Runtime defaults are authored in `internal/embed/defaults/` and
installed into `.strategist/`; the retired root `strategist/` authoring mirror is not part of
the current path model. The architectural decision for this ordering is consolidated in
`docs/adr/0010-ordered-contracts-and-mission-observability.md`.

A CLI-free, Markdown-centered *soft profile* of this same runtime was evaluated as feasible
(additive, not a replacement) — see
[`docs/design/soft-profile-orka-mapping.md`](design/soft-profile-orka-mapping.md) for the
capability disposition matrix and target packaging structure.

---

## Compilation Pipeline

`CompileAll` produces 4 artifacts in `.strategist/.compiled/`:

| Artifact | Purpose | Sources |
|----------|---------|---------|
| `.index.gz` | Compiled knowledge index | `knowledge.index.yaml` |
| `.domain.gz` | Compiled domain templates | `templates/domain/**/*.yaml` |
| `.config.gz` | Compiled configuration | `active.yaml` + `personas/*.yaml` + `roles/*.yaml` |
| `.manifest.gz` | Hashes of the 3 artifacts above | generated by `CompileAll` |

Each artifact is **gzip + JSON**. The JSON schema includes:
- `schema` — format version identifier (e.g. `strategist-compiled-config/1.0`)
- `compiled_at` — Unix timestamp of compilation
- `sources` — map of `path → mtime` for the sources used

The `sources` field is what allows `Checker.IsStale` to detect whether any source was modified after compilation.

---

## Staleness Detection

`stale.Checker.IsStale(artifactPath)` returns `true` when:

1. The artifact file does not exist
2. `.manifest.gz` does not exist in the same directory
3. Any source in `artifact.sources` no longer exists on disk
4. Any source has an `mtime` newer than the recorded value

Returns `false` (fresh) only when all sources exist and their mtimes are ≤ the recorded values.

The `check-stale` CLI exits with code `0` if fresh and `1` if stale — designed for use in CI scripts.

---

## Embedded Defaults

`internal/embed/defaults/` is the authoring source itself, included in the binary via `//go:embed all:defaults` (the former `strategist/` authoring mirror was retired in W7a — there is no sync step). This means `strategist install` works **without a network connection** and **without the repository cloned** — the binary carries all defaults in memory.

Extraction preserves the directory structure but does not overwrite pre-existing files (files are written via `os.WriteFile` directly — projects with a custom `.strategist/` should back up before re-installing).

---

## Domain Interfaces (`internal/domain/ports.go`)

| Interface | Method | Implemented by |
|-----------|--------|---------------|
| `Installer` | `Install(InstallConfig) error` | `install.serviceAdapter` |
| `Compiler` | `CompileAll(root, indexPath string) error` | `compile.Compiler` |
| `StaleChecker` | `IsStale(artifactPath string) (bool, error)` | `stale.Checker` |
| `FileExtractor` | `Extract(targetDir string) error` | `embed.Extractor` |

Interfaces are satisfied via compile-time verification (`var _ domain.X = Y{}`), ensuring no implementation silently diverges.

---

## Tests

| Suite | Package | Approach |
|-------|---------|---------|
| `stale_test.go` | `internal/stale` | 5 cases: absent, no manifest, fresh, stale source, removed source |
| `compile_test.go` | `internal/compile` | Config, Domain, Index, All (4 artifacts + manifest) |
| `install_test.go` | `internal/install` | Silent mode, gitignore, rollback |
| `installer_whitebox_test.go` | `internal/install` | `ensureGitignore`, error propagation |
| `fixtures_test.go` | `tests/` | Format of 5 security invariant fixtures |
| `cmd_test.go` | `cmd/strategist` | All CLI commands |
| `checker_*_test.go` | `internal/dojo` | Offline scenario criteria, manifest, and pipeline validation |
| `*_test.go` | `internal/treasure` | Treasure Chest, jewel, potion, and governed-loader behavior |

Race detector enabled on core test targets (`make test`, `make spec`, `make integration`).
Coverage gate thresholds are declared in `scripts/coverage-packages.tsv` and enforced
by `make cover-gate`; the current gate covers the packages listed in that manifest
rather than every `internal/` package.
