# ADR-0022 — `eval harvest --all`: Tolerant Scan, No Parser Change

**Status:** Accepted  
**Date:** 2026-08-04  
**Context:** `20260804-treasure-scan-sq-block-bug`

---

## Context

`20260804-eval-harvest`'s (ADR-0018) own acceptance checks state that
`strategist eval harvest --all` must run without error against this
workspace's real `.analysis/refined/` + `.analysis/done/` history.
Verifying that check found it failing: one real mission file
(`.analysis/done/2026-07-22-observability-doc-output-profile-error/tasks.md`,
and 15+ others sharing the same shape) breaks `treasure.ScanMissions`.

The root cause is `internal/treasure/scan_parse.go`'s
`normalizeLegacySideQuestFields`: it strips the leading `- ` from any
indented `- key: value` line, without distinguishing a standard
2-space-indented list item's own opening dash (e.g. `  - id: SQ-001` — a
normal, valid YAML list under `side_quests_approved:`) from the legacy
per-field-bullet continuation lines it was actually meant to normalize.
Stripping the opening dash turns `  - id: SQ-001\n    description: ...`
into `  id: SQ-001\n    description: ...` — a deeper-indented line
directly after a scalar `key: value`, which YAML rejects as "mapping
values are not allowed in this context". Reproduced against 5 real
mission files sharing this shape under `.analysis/done/`.

**Correction (2026-08-04):** this root cause was originally misattributed
during discovery to `sideQuestBlockEnd` not stopping at the frontmatter's
closing `---`, with an estimated "16+ affected files" based on a grep for
a sibling-frontmatter-key shape. That theory did not reproduce in
isolation and the file count was wrong (most of the 16 use a 0-indent
list style that parses fine). Corrected after building T4's test fixture
during implementation, which required an isolated repro to get right.
See `.analysis/refined/20260804-treasure-scan-sq-block-bug/analysis.md`'s
own Correction Note for the full account. DEC-1/DEC-2 and their
rejected-alternatives reasoning are unaffected by this correction — the
chosen fix treats any parse failure uniformly, regardless of cause.

`internal/treasure/scan.go` already provides `ScanMissionsTolerant` /
`ScanMissionsInDirTolerant`, built for exactly this failure mode: skip an
unparseable mission and report it as a `ScanWarning` instead of aborting
the whole scan. `RunScanPipeline` (the engine behind
`strategist treasure-chest index`) already uses it against this same
real workspace data, and `internal/treasurecli/treasure_chest_index.go`
already has a working precedent (`printTreasureChestIndexWarnings`) for
surfacing those warnings to the user.

`cmd/strategist/eval_harvest.go`'s `--all` branch
(`selectHarvestMissionIDs`) was the one caller still using the strict
`ScanMissions`.

## Decision

**DEC-1:** `selectHarvestMissionIDs`'s `--all` branch switches from
`treasure.ScanMissions` to `treasure.ScanMissionsTolerant`.
`internal/treasure/**` is not modified — the fix reuses an existing,
already-tested API rather than touching the parser.

**DEC-2:** `selectHarvestMissionIDs`'s return signature changes to
`([]string, []treasure.ScanWarning, error)`. `runEvalHarvest` — which
already holds `cmd *cobra.Command` — prints the returned warnings via
`cmd.PrintErrf`, mirroring `treasure_chest_index.go`'s
`printTreasureChestIndexWarnings` message shape exactly, instead of
threading a `*cobra.Command` parameter into the selection function.

### Alternatives Considered and Rejected

- **Fix `scan_parse.go`'s `normalizeLegacySideQuestFields` to stop
  over-stripping the opening dash of standard indented list items.**
  Rejected for this mission: it would recover correct side-quest data
  for the 5 affected missions (this fix only skips them with a warning),
  but requires touching `internal/treasure/**`, forbidden here on the
  same grounds ADR-0018 already established. Recorded as SQ-001 for a
  future, separately scoped mission.
- **Leave `--all` broken, document single-mission-only usage.** Rejected:
  contradicts ADR-0018's own stated acceptance check.
- **Pass `*cobra.Command` into `selectHarvestMissionIDs` and print
  inline.** Rejected: couples pure selection logic to Cobra's I/O surface
  for no benefit; returning warnings keeps the function unit-testable
  without a fake command and centralizes printing at the one call site
  that already owns `cmd`.

## Consequences

- `strategist eval harvest --all` will complete against this workspace's
  real mission history, printing one warning line per malformed mission
  file instead of aborting.
- The 5 real missions sharing the affected shape are skipped (not
  harvested) when running `--all`; they can still be harvested
  individually by `strategist eval harvest <mission_id>`, which never
  calls `ScanMissions`.
- `internal/treasure/scan_test.go`'s existing
  `TestScanMissionsInDir_PropagatesParseError` (strict-mode hard-fail) is
  unaffected and remains correct — this fix only changes which API
  `eval_harvest.go` calls, not the strict API's own behavior.
- SQ-001 (the underlying parser fix that would let the 5 missions'
  side-quest data actually be read, not just skipped) stays open as a
  separate, future, narrowly-scoped mission — this ADR does not close
  it, only documents why it is out of scope here.
- Future missions extending `eval_harvest.go` should reuse
  `printEvalHarvestWarnings`'s pattern for any other `treasure`-backed
  batch operation this command grows, rather than reintroducing a strict
  call site.
