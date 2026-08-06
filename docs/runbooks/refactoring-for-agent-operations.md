# Runbook: Refactoring for Agent Operations

## Symptom

Use this runbook when an agent receives a focused refactoring request driven by
a concrete signal, for example:

- `golangci-lint` reports a narrow rule violation such as `gocritic`
  `unlambda` or `wrapcheck`;
- complexity tooling reports a function or test above a stated threshold;
- the user asks to reduce complexity below a numeric limit;
- the intended validation is local to touched packages.

This runbook is for behavior-preserving cleanup. It is not a license for broad
architecture work.

## Root Cause

These failures usually come from local control-flow accumulation, repeated test
setup/assertion logic, or boundary errors returned without context. The fix is
normally a small reduction in the nearest unit:

- replace pass-through wrappers with direct function references;
- wrap external package errors at the boundary with `%w`;
- extract predicates or IO helpers from scan/flush loops;
- move repeated test setup and assertions into `t.Helper()` helpers;
- use standard comparison helpers such as `slices.Equal` instead of manual
  loops when available.

The important distinction is scope. A linter or complexity finding identifies a
local maintenance problem; it does not by itself justify renaming ownership
boundaries, changing thresholds, or reorganizing unrelated modules.

## Resolution Steps

1. Capture the exact signal: tool, rule, file, function, line, and target
   threshold.
2. Classify the finding:
   - direct simplification, such as assigning a handler function directly;
   - error-boundary wrapping with `fmt.Errorf("context: %w", err)`;
   - production complexity reduction;
   - test complexity reduction.
3. Reduce the smallest unit that clears the signal:
   - extract a predicate when a loop mixes scanning, parsing, and matching;
   - extract read/write/truncate helpers when IO error handling inflates the
     caller;
   - extract repeated test assertions into helpers marked with `t.Helper()`;
   - replace manual slice comparison with `slices.Equal` when order matters and
     the Go version supports it.
4. Preserve behavior:
   - do not change linter thresholds unless explicitly requested;
   - do not broaden to unrelated files;
   - do not convert refactoring into feature work;
   - do not change persistence, locking, or error semantics unless the finding
     explicitly requires it.
5. Validate with the same signal that triggered the work, then run focused tests.
   In sandboxed sessions, put caches under `/tmp` when home cache directories
   are read-only:
   ```bash
   GOCACHE=/tmp/go-cache $(go env GOPATH)/bin/gocognit -over 6 internal/telemetry
   GOCACHE=/tmp/go-cache go test ./internal/telemetry
   GOCACHE=/tmp/go-cache GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache $(go env GOPATH)/bin/golangci-lint run ./internal/telemetry ./cmd/strategist
   ```
6. Report the result with:
   - files changed;
   - detector output after cleanup;
   - tests run;
   - any sandbox cache workaround used.

## Stop Conditions

Stop or ask before continuing when:

- the cleanup would touch unrelated modules;
- reducing complexity requires changing behavior;
- the requested threshold conflicts with readability;
- the lint finding points to a broader design problem that needs a separate
  demand;
- validation cannot be run and the risk is not low.

The runbook is worth maintaining only while it stays procedural and recurring.
It is not worth the aggregate maintenance effort if it grows into a generic
style guide or accumulates one-off preferences that future agents cannot reuse.

## Reference

- `internal/treasurecli/treasure_chest_doctor.go` — direct handler assignment for a
  pass-through `RunE` wrapper.
- `internal/telemetry/outcome.go` — external error wrapping and helper
  extraction for scan/flush behavior.
- `internal/telemetry/outcome_test.go` — test fixture and assertion helpers.
- `internal/telemetry/sniper_conflict_test.go` — `slices.Equal` instead of a
  manual comparison loop.
- `internal/telemetry/mission_run.go` — fallback helper extraction for repeated
  timestamp defaulting.
