# ADR-0020 — Promptfoo Adapter: Formalized Content, No CI Wiring

**Status:** Accepted  
**Date:** 2026-08-04  
**Context:** `20260804-promptfoo-ci-adapter`

---

## Context

`SQ-004` ("Promptfoo adapter for external CI comparison") was accepted at
ADR-0016's (`20260804-test-framework-v2`) gate as a low-impact side quest,
citing the source draft's own reasoning that Promptfoo is "a better fit for
the external/CI layer" than DeepEval/Inspect AI, given this project's
Go-native core.

Refinement of this mission (`20260804-promptfoo-ci-adapter`) found two
things that shaped scope:

1. The same source critique describes Promptfoo's actual fit as "prompt +
   input → response → assertions" — single-shot evaluation, not full
   agentic-loop simulation — and separately documents that
   `docs/runbooks/local-llm-quality-review-lm-studio.md` (`SQ-003`, ADR-0019)
   already has real, working prompt content of exactly this shape, with an
   explicit forward-reference to this mission.
2. `make eval` — Phase 1's own deterministic, Go-native eval harness — is
   not currently wired into CI (`.github/workflows/test.yml` runs
   `ci-lint`, `ci-test`, `vuln-ci`, `validate-fixtures`, `ci-web`, never
   `eval`). The card's title assumed CI integration was the point; this
   finding shows the sibling capability it would extend isn't there yet
   either.

## Decision

**DEC-1:** The initial `promptfoo/promptfooconfig.yaml` formalizes the LM
Studio runbook's manual quality-review prompt as a reusable, versioned
config, with a local (LM-Studio-class) provider entry. Provider comparison
— Promptfoo's core mechanism — becomes available for free once this
exists; it needs no separate design. Adversarial/injection-style prompt
content is explicitly **not** built in this mission — that content was
deferred by ADR-0017 and has no evidence base here to build from.

**DEC-2:** Add `make eval-promptfoo` as a standalone, optional Makefile
target — not a dependency of `eval`, `test`, `test-all`, `ci-test`, or
`ci`. No `.github/workflows/*.yml` change.

### Alternatives Considered and Rejected

- **Build adversarial/injection tests as the primary content** (DEC-1).
  Rejected: no evidence in this mission defines those scenarios; they
  belong to a future mission with its own discovery.
- **Build all three candidate content types now** (DEC-1). Rejected as
  unbounded scope for a card marked "Impact: low."
- **Wire Promptfoo into CI now** (DEC-2). Rejected: contradicts the
  non-determinism/advisory-only posture already established for this exact
  content, and would put `SQ-004` ahead of `make eval` itself in CI
  maturity for no stated reason.
- **No Makefile target at all** (DEC-2). Rejected: a named target costs
  little and keeps naming consistency with the `eval`/`eval harvest`
  command family; optional doesn't require undiscoverable.

## Consequences

- `SQ-004` gets a concrete, evidence-grounded starting scope instead of an
  open-ended "add Promptfoo" mandate.
- `docs/runbooks/local-llm-quality-review-lm-studio.md` remains the
  human-readable procedure and its warnings still apply in full — this
  formalization does not weaken or automate away the advisory-only framing.
- `promptfoo/` introduces a new, narrowly-scoped Node.js dependency,
  separate from `web/landing/`'s existing, unrelated Node subsystem.
- CI (`ci`, `ci-test`, `.github/workflows/test.yml`) is unaffected by this
  work. A future mission may reconsider CI wiring for `make eval` and
  `make eval-promptfoo` together, but that is not decided here.
- Implementation (T2–T5's code/config items) remains `implementation_handoff`
  — outside this Strategist mission's execution scope.
