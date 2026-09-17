# ADR-0044 — STARTUP Consumes `PreflightResult` Instead of Narrating Its Own Algorithm

**Status:** Accepted
**Date:** 2026-09-16
**Context:** `20260916-preflight-startup-consumption-onda3` (Roadmap Wave 3, Onda 3), refining `20260916-preflight-cli-enforcement-wave3`'s Task Group B design

## Context

`v2/03-preflight-cli-enforcement.md`'s own 4-wave migration for Preflight
CLI Enforcement is Onda 1 (mechanical `agent_only` conditions → Go), Onda 2
(a typed `PreflightResult`), Onda 3 (the agent's STARTUP procedure consumes
that result instead of reproducing the algorithm), Onda 4 (remove
redundant procedural instructions once equivalence is proven).

Onda 2 shipped earlier this thread: `domain.PreflightResult`
(`internal/domain/preflight_result.go`) plus a tested `strategist check
--json` surface (`internal/check/check_preflight_json.go`). Onda 1 shipped
today (`20260916-preflight-cli-enforcement-wave3`): all 7 conditions in
`contracts/machine/preflight.yaml` are now `enforced_by: machine_enforced`,
surfaced in `PreflightResult.Warnings`.

Despite both, `.strategist/agent-protocol.md` §1 STARTUP remained 4 lines
of narrative prose — it never invoked `strategist check --json` or
branched on its output. The original draft's own completion criterion
("two clients receiving the same workspace/configuration must observe the
same status, warnings, binding, and next step") was not met: nothing
prevented two LLM clients from narrating the same 4 steps slightly
differently, even with identical underlying state.

A blocking gap surfaced while resolving this: `buildPreflightResult`'s
`Next` field was only ever a message string ("resolve the warnings
below...") or empty — never a real phase token. STARTUP's target text
("proceed to `next`") is meaningless without one.

Separately, and freshly demonstrated this same session:
`contracts/machine/preflight.yaml` (a `NormativeRuntimeDefaultFiles()`
entry) was edited at its authoring source
(`internal/embed/defaults/contracts/machine/preflight.yaml`) and its
runtime copy hand-synced, but `strategist check` then failed with
`runtime_stale_conflict` — because the *installed CLI binary* still had
the old content compiled into its `go:embed` FS. Only a rebuild +
reinstall fixed it. `agent-protocol.md` is the same file class, at far
higher blast radius (every mission, on every route, bootstraps through
it).

## Decision

- **DEC-001.** `PreflightResult.Next` carries a real next-phase token
  (`"intake"`) when `Status == "ready"`, not an empty string. The blocked
  branch's message-string content is unchanged. This is additive, not
  breaking: no code reads `.Next` programmatically today (confirmed by
  repo-wide search before this decision), so widening its "ready" value
  has no existing consumer to break.
- **DEC-002.** `.strategist/agent-protocol.md` §1 STARTUP is rewritten to:
  check `.strategist/` exists → run `strategist check --json` once,
  capture the `PreflightResult` → branch on `Status` (`"blocked"`: emit
  `Warnings`, stop; `"ready"`: read the rest of this file, then proceed to
  `Next`). The prior step 3 ("Is `active.yaml` readable?") is dropped as a
  separate narration — Onda 1 already surfaces that failure inside the
  single `--json` call's `Warnings`.
- **DEC-003.** `.strategist/contracts/narrative/01-bootstrap.md`'s
  "Required Behavior" list drops its own narrative restatement of
  compiled-artifact-freshness/identity/directives handling — all 7 of
  those conditions are now `PreflightResult.Warnings` entries per Onda 1 —
  replacing that prose with a cross-reference to the machine-enforced
  contract. Every other bullet in that list (chat/docs language
  resolution, `governance_injection` forwarding, the mandatory stale scan,
  Keen Senses) is unrelated to preflight.yaml's 7 conditions and is left
  untouched.
- **DEC-004.** Any edit to a `NormativeRuntimeDefaultFiles()` entry (this
  ADR's own `agent-protocol.md`/`01-bootstrap.md` edits included) must be
  followed by an explicit rebuild-and-reinstall-and-verify sequence before
  the change is considered live: rebuild the `strategist` CLI binary from
  the edited authoring source, reinstall it to wherever `strategist`
  currently resolves on `PATH`, re-run `strategist check` and confirm zero
  `runtime_stale_*` diagnostics, and confirm the `.strategist/` runtime
  copy is byte-identical to the authoring source. This is now a normative
  practice for this class of file, not a one-off remediation for this
  mission alone.

## Consequences

### Positive

- STARTUP finally satisfies the original draft's own completion bar: the
  procedure is delegated to a single, versioned, tested machine output
  instead of client-narrated prose.
- `01-bootstrap.md` shrinks to only the behaviors genuinely not yet
  machine-enforced, making the boundary between `agent_only` and
  `machine_enforced` legible by inspection rather than by cross-referencing
  `preflight.yaml`'s comment block.
- DEC-004 turns today's `runtime_stale_conflict` incident into a documented,
  repeatable safeguard instead of a one-off lesson a future contributor has
  to rediscover.

### Negative

- An already-open agent session bootstrapped under the old STARTUP text
  will not observe this change until its next fresh bootstrap — not a
  defect, but worth naming so a rollout is not mistaken for instantaneous
  everywhere.
- `Next = "intake"` is hardcoded for the ready branch, assuming intake is
  always the correct next phase (true for every route today per
  `00-routing.md`'s sequences). A future route that needs STARTUP to skip
  intake would require revisiting this computation.

## Rejected Alternatives

- **Option (b): keep STARTUP's own hardcoded "proceed to intake" for the
  ready case, delegating only the blocked branch to `PreflightResult`.**
  Rejected — smaller schema change, but less faithful to the original
  draft's own example JSON (`"next": {"phase": "intake", ...}`) and leaves
  the "ready" branch's phase transition un-derived from the machine result,
  reintroducing exactly the client-narration risk this ADR closes for the
  blocked branch only.
- **Fold Onda 1's newly-machine-enforced conditions into `01-bootstrap.md`'s
  existing prose instead of replacing it with a cross-reference.**
  Rejected — would leave two descriptions of the same 7 conditions (the
  contract file and the narrative bullet list) that can drift independently;
  a cross-reference has exactly one source of truth.
- **Skip the rebuild-and-reinstall step for a "text-only" change.**
  Rejected — this session's own `runtime_stale_conflict` incident, for a
  much smaller file (`preflight.yaml`), demonstrates that "text-only"
  does not imply "binary-unaffected": the failure mode is in the
  `go:embed`-compiled binary's staleness, not the semantic size of the edit.

## References

- `.analysis/pending/cli_refactor/20260916-preflight-startup-consumption-onda3/`
  — the refined package this ADR documents (analysis.md KF-001–KF-006,
  design.md Tasks 1–3).
- `.analysis/pending/cli_refactor/20260916-preflight-cli-enforcement-wave3/`
  — Onda 1's implementation, whose completion this ADR's Context section
  relies on.
- `internal/domain/runtime_defaults.go` — `NormativeRuntimeDefaultFiles()`,
  `DecideRuntimeDefaultUpdate`, `FormatRuntimeStaleDiagnostic` — the
  mechanism DEC-004 is a normative-practice response to.
- `.analysis/pending/cli_refactor/v2/03-preflight-cli-enforcement.md` — the
  original 4-wave draft this ADR closes Onda 3 of.
