# ADR-0026 — Deterministic Golden Testing for Generated Artifacts

**Title:** Deterministic Golden Testing for Generated Artifacts
**Status:** Accepted
**Date:** 2026-08-11
**Mission ID:** 20260811-golden-antidrift

## Context

Strategist should detect unintentional drift in stable artifacts and textual/structured
interfaces. The proposed golden-test system spans 8 artifact categories (compiled prompts,
manifests, compiled artifacts, handoffs, telemetry events, CLI help, generated docs,
rendered schemas), three comparison strategies (exact text, normalized JSON/YAML,
structural), volatile-field normalization, a human-reviewed update flow, and
integration with "contract provenance."

Repository inspection found that, unlike two sibling items in the same design set — item 2 (test
architecture) and item 6 (evaluation providers), where the proposed architecture had
already been examined and explicitly rejected in ADR-0016/0017 — **no ADR addresses a
golden-diff-and-update system for deterministic artifacts**. `tests/golden/` does not
exist; the classic Go golden-test idiom (`-update` flag) does not exist anywhere in the
codebase; `internal/compile/testdata/` is input fixtures for the compiler, not
output-comparison goldens. The only overlap with a prior decision is partial: ADR-0016
deferred (not structurally rejected, unlike `FakeProvider`) golden **prompt** tests
specifically, for lack of an LLM provider-invocation call site — a constraint that
applies to comparing LLM *responses*, not to comparing the deterministic *output of a
Go template* that produces a compiled prompt.

## Decision

Adopt the following design:

1. Build a deterministic golden-test system under `tests/evals/golden/` (a subdirectory
   of the already-existing `tests/evals/`, alongside `contracts/`, `fixtures/`,
   `regression/`, `scenarios/`) — not a new top-level `tests/golden/` — keeping the
   existing test macrostructure (`tests/{evals,integration,spec}/`, already diverging
   by name from the base document's original proposal per item 2's discovery) intact.
2. Cover the 6 unambiguously deterministic categories first: manifests/handoffs,
   telemetry events, CLI help, rendered schemas, compiled contracts, default
   configuration — each sourced from an already-existing deterministic Go surface
   (`internal/handoff/`, `internal/telemetry/schema.go`, Cobra commands,
   `.strategist/schemas/*.yaml`, `internal/compile/`, `internal/embed/defaults/`).
3. Include compiled prompts as a 7th, conditionally-scoped category: compiling a prompt
   template is a deterministic Go operation, distinct from generating or comparing an
   LLM *response* — the latter remains out of scope per ADR-0016, the former does not.
4. Implement three comparators (exact text, normalized JSON/YAML, structural),
   volatile-field normalization (timestamp/UUID/temp-path/hash/duration/hostname),
   reusing the sanitization pattern already established in
   `internal/telemetry/schema.go`'s `SanitizePath` (see ADR-0024).
5. Implement a `-update` flag following the classic Go idiom, never runnable in CI, with
   a required diff review — mirroring the "fail, never silently reconcile" posture
   already used by `docs-governance-gate`/`check-convergence.sh`.
6. Link critical goldens to contracts via metadata (`golden: {id, contracts: [...]}`),
   coordinating with the `contract-index.md` candidate already registered under
   ADR-0025, rather than creating a second source of truth for "which contracts does
   this artifact affect."

This is an analysis/documentation decision only. No code was written or modified by
Strategist as part of this mission; implementation of items 1–6 above is out of
Strategist's execution scope (`task_type: implementation_handoff`) and requires a
separate coding task.

## Consequences

**Positive:**
- Strategist gains systematic drift detection for artifacts that currently have no
  output-comparison testing at all (manifests, handoffs, telemetry events, CLI help,
  schemas, compiled contracts, default config).
- Reusing `tests/evals/` as the parent directory avoids introducing a fifth test-tree
  root and stays consistent with the macrostructure decision already recorded in the
  test-architecture item's reconciliation note.
- Scoping compiled prompts as deterministic (rather than excluding them by proximity to
  ADR-0016) recovers one of the base document's originally-proposed categories without
  reopening the LLM-response question ADR-0016 already settled.
- Golden-to-contract linkage avoids duplicating the contract-provenance concept already
  planned under ADR-0025.

**Negative / follow-up required:**
- The prompt-compilation golden category (item 6 in the tasks list) still requires
  locating the actual persona/role template source before it can be scoped precisely —
  not fully specified by this ADR.
- No CI workflow currently references a golden gate; wiring `tests/evals/golden/` into
  `ci-test` (or an equivalent gate) is a distinct, unscoped follow-up decision.
- The volatile-field normalization approach is recommended by precedent
  (`SanitizePath`) but not yet generalized into a reusable helper — implementation-time
  work, not decided here.
