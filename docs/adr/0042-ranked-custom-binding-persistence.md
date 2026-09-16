# ADR-0042 — Ranked/Custom Binding Persistence: CLI Certification Is Authoritative, Runtime Files Are Mandatory Only for Custom

**Status:** Accepted
**Date:** 2026-09-16
**Context:** thread continuing `20260915-cli-enforcement-refactor-refinement` (task 1.1b) and `20260916-ranked-vs-custom-binding-pipelines`

## Context

ADR-0041's "Ranked Class Resolution" amendment (§"Pipeline scope") established
that Role→Weapon binding is two separate pipelines:

- **Custom** — a wizard-selected embedded or external weapon, validated and
  probed at wizard time, persisted to `.strategist/plugins.lock`
  (`domain.SlotBinding`, per ADR-0037). This is today's only implemented
  pipeline.
- **Ranked** ("classe rankeada") — a weapon bound to a Role at compile/build
  time, certified by the release build's own pipeline
  (`.analysis/pending/cli_refactor/v2/02-ranked-role-binding.md`'s
  "Build-time certification": manifest/dependency/affinity/contract-test/
  handoff-schema validation, digest calculation, catalog embedding). Not
  implemented yet.

That draft's own `plugins.lock` proposal already sketched a per-slot
`mode: ranked | custom` discriminator, with materially lighter fields for a
`ranked` entry (`{mode, binding, release, digest}`, sourced from the embedded
catalog) than a `custom` entry (`{mode, weapon, source, version, digest,
compatibility}`, sourced from wizard-time resolution). This discriminator was
implemented directly in code as of this thread
(`domain.SlotBinding.Mode`, `SlotBindingModeCustom`/`SlotBindingModeRanked`,
`EffectiveMode()`, `ValidMode()` — `internal/domain/plugin_state_types.go`).

What remained open was task 1.1b in
`.analysis/pending/cli_refactor/20260915-cli-enforcement-refactor-refinement/tasks.md`:
should `SlotBinding` split into its own on-disk resource (as the original,
pre-Ranked/Custom-split V1 draft proposed, citing "authority separation is
mandatory"), or stay bundled in the single `plugins.lock` file (as
`strategist-papeis-personagens-skills-nativas` Task 3.3 — "in the existing
lock" — separately assumed, without the Ranked/Custom distinction in view)?

## Decision

- **DEC-001.** Two binding-reinforcement mechanisms coexist, one per
  pipeline, not as redundant alternatives for the same pipeline:
  1. **CLI/build-time reinforcement** — the Ranked pipeline's certification
     pipeline, producing a deterministic, digest-pinned binding embedded into
     the release binary/catalog. This is the sole authority for a Ranked
     binding.
  2. **Runtime file reinforcement** — `plugins.lock`'s persisted
     `SlotBinding` records (ADR-0037). This is the sole authority for a
     Custom binding.

- **DEC-002.** Runtime file persistence remains **mandatory** for every
  Custom binding, unchanged from ADR-0037: a wizard-time, potentially
  external, mutable choice has no other durable record, and
  `strategist check`/`RoleInvocationPlan` must be able to read it back
  deterministically.

- **DEC-003.** Runtime file persistence for a Ranked binding is **optional
  redundancy**, never a correctness requirement. A `mode: ranked` entry in
  `plugins.lock`, if written at all, is a cache/mirror — useful for
  observability (e.g., `strategist check` displaying which Ranked binding an
  install last resolved, or an audit trail of when it changed across
  releases) — but never load-bearing: the embedded catalog and the release's
  own build-time certification remain authoritative regardless of whether a
  runtime copy exists, is stale, or is absent entirely. Nothing may treat a
  missing or stale Ranked runtime record as an error condition.

- **DEC-004 (resolves task 1.1b).** `SlotBinding` does **not** need to split
  into a separate on-disk resource (e.g. `bindings.yaml`). The original V1
  draft's "authority separation is mandatory" concern assumed every
  persisted binding record carries equal, independent authority — DEC-001–
  DEC-003 show this is false once Ranked and Custom are distinguished: only
  the Custom record is authoritative; a Ranked record, when present, is
  explicitly non-authoritative. A single `plugins.lock` file with the
  already-implemented `mode` discriminator satisfies "no overlapping
  authority" without a physical file split, because there is only one real
  authority (Custom) to separate anything from.

This confirms `strategist-papeis-personagens-skills-nativas`'s Task 3.3
("record bindings in the existing lock") as the direction going forward —
not because that mission decided the Ranked/Custom question (it did not
consider it), but because DEC-001–DEC-003 independently reach the same
conclusion once the pipelines are properly distinguished.

## Consequences

### Positive

- Task 1.1b is closed: no file-format migration is needed for
  `.strategist/plugins.lock`, in either direction.
- The `mode` discriminator implemented this thread already satisfies this
  decision as written — no further schema change is required to act on this
  ADR.
- Keeps exactly one required on-disk write path (Custom via ADR-0037),
  rather than two independently-evolving persistence mechanisms for what is,
  for Ranked, non-authoritative data.

### Negative

- If a future Ranked implementation does add an optional runtime mirror, it
  must be built with explicit staleness tolerance from day one (per DEC-003)
  — a contributor who reflexively treats a persisted record as a source of
  truth (the Custom-pipeline default assumption everywhere else in this
  codebase) could accidentally make Ranked's mirror load-bearing by mistake.
  No code enforces DEC-003 today beyond this ADR and `NewRoleInvocationPlanFromLock`'s
  existing fail-closed rejection of `mode: ranked` (it does not yet
  implement Ranked resolution at all, so this risk is latent, not live).

## Rejected Alternatives

- **Split `SlotBinding` into a separate `bindings.yaml`, per the original V1
  draft's file layout.** Rejected — DEC-004: the premise (every binding
  record is an independent authority) does not hold once Ranked is
  understood to be non-authoritative at the runtime-file layer.
- **Require runtime persistence for Ranked bindings too, mirroring Custom
  symmetrically.** Rejected — adds a mandatory write/consistency-check
  surface for state the build already fully determines, with no correctness
  benefit; optional redundancy captures the observability value without the
  obligation.
- **Forbid any runtime record for Ranked bindings.** Rejected — a
  read-only, best-effort mirror has real diagnostic value (e.g. surfacing in
  `strategist check`'s output which Ranked binding is active) and DEC-003
  already neutralizes the main risk (someone depending on it for
  correctness) by making non-authoritative status explicit policy.

## References

- `docs/adr/0041-cli-enforcement-sequencing-and-role-invocation-plan-naming.md`
  §"Ranked Class Resolution" / §"Pipeline scope" — the Ranked/Custom split
  this ADR builds on.
- `docs/adr/0037-wizard-role-binding-persistence.md` — the Custom pipeline's
  existing, unchanged persistence decision.
- `.analysis/pending/cli_refactor/v2/02-ranked-role-binding.md` — the
  pending draft describing the Ranked pipeline's build-time certification
  and original `plugins.lock mode:` proposal (still not itself approved for
  implementation).
- `.analysis/pending/cli_refactor/20260915-cli-enforcement-refactor-refinement/tasks.md`
  task 1.1b — the task this ADR closes.
- `internal/domain/plugin_state_types.go` — `SlotBinding.Mode`,
  `EffectiveMode()`, `ValidMode()` (implemented prior to this ADR, in the
  same thread).
