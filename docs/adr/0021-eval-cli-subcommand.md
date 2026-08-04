# ADR-0021 — `strategist eval run`: Wrap `go test`, One Flexible Subcommand

**Status:** Accepted  
**Date:** 2026-08-04  
**Context:** `20260804-eval-cli-subcommand`

---

## Context

`SQ-001` asked for a `strategist eval` CLI sub-command wrapping the
`internal/eval` scenario battery, following the original 2026-07-28 draft's
proposed command set: `strategist eval unit|contracts|scenarios|regression|
adversarial|all`.

Refinement of this mission (`20260804-eval-cli-subcommand`) found that only
two of those six groups (`contracts`, `scenarios`) exist as real Go test
packages today; `unit`, `adversarial`, and `golden` don't exist at all, and
`regression` (from `SQ-002`'s `strategist eval harvest`) holds only fixture
data, no scenario tests. It also found that scenario definitions live as Go
struct literals inside `_test.go` files, not in a CLI-loadable data format
— meaning `go test` is the only architecturally viable way to execute them
without a much larger refactor. This codebase's only existing subprocess
precedent is `git` (`cmd/strategist/check_f3_conflict.go`); nothing
previously shelled out to `go`.

## Decision

**DEC-1:** `strategist eval run [pattern]` shells out to
`go test -race -tags=eval <pattern>` via `exec.Command` and streams its
output — it actually runs the scenario battery, not merely prints the
equivalent command. This is a new pattern for this codebase (first `go`
subprocess invocation), justified by this tool's realistic deployment
context: it runs inside this same Go monorepo, where the `go` toolchain is
already a safe assumption (built with Go, CI and developers already have
it) — unlike a general-purpose end-user CLI shipped standalone.

**DEC-2:** One flexible subcommand takes a package-pattern argument
(default `./tests/evals/...`) instead of fixed subcommands per the original
draft's six group names. Three of those six don't exist yet; hardcoding
subcommands for them would misrepresent coverage that isn't there and need
maintenance every time a group is added or renamed.

**DEC-3:** Named `run`, distinct from the existing `strategist eval
harvest` (fixture extraction, not test execution).

### Alternatives Considered and Rejected

- **Signpost/print-only command** (DEC-1). Rejected: near-zero value over
  existing `--help`/docs text; doesn't fulfill "wrapping the scenario
  battery."
- **Fixed subcommands per the original draft's six names** (DEC-2).
  Rejected: three don't exist yet; the other three would each be a thin
  wrapper around a hardcoded path, better expressed as one argument.
- **`strategist eval test`** (DEC-3). Rejected: redundant with `eval`'s own
  meaning here, invites confusion with Go's own `go test`.
- **`strategist eval all`** (DEC-3). Rejected: implies always running
  everything, less flexible than an explicit pattern argument.

## Consequences

- `strategist eval run` composes with the existing `evalCmd`/
  `resolveEvalActionRoot` (from `SQ-002`) without duplicating root
  resolution.
- This is the first codebase precedent for a `go`-toolchain subprocess
  dependency at runtime — future contributors should be aware `strategist
  eval run` requires `go` on `PATH`, unlike every other existing command.
- `unit`/`adversarial`/`golden` remain addressable via the same command
  once/if those test groups are built (`strategist eval run
  ./tests/evals/adversarial/...`), with zero new CLI code required.
- Exit-code propagation from `go test` failures is flagged as an
  implementation-time verification point (T3), not fully specified by this
  design — a silently-swallowed test failure would be a correctness bug.
- Implementation (T2–T5's code items) remains `implementation_handoff` —
  outside this Strategist mission's execution scope.
- This closes the last of the five `SQ-*` cards spawned from ADR-0016's
  (`20260804-test-framework-v2`) gate.
