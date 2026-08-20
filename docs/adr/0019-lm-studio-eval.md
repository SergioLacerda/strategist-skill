# ADR-0019 — LM Studio Local Quality Review: Runbook, Not Code

**Status:** Accepted  
**Date:** 2026-08-04  
**Context:** `20260804-lm-studio-eval`

---

## Context

`SQ-003` ("LM Studio integration for local LLM evaluation") carried a note
suggesting it depended on a provider-invocation interface, mirroring
`SQ-005`. ADR-0017 (`20260804-eval-fake-provider`, DEC-3) already corrected
that: `SQ-003`'s real dependency, if any, is the external live-invocation
tooling direction (`SQ-004`/Promptfoo), not a Go interface.

This mission (`20260804-lm-studio-eval`) went further and re-examined the
original source critique (the runtime workspace's `todo/v2/tests/tests_v2.txt` and
`proposal.md`) that first proposed this work. Two things stood out:

1. The critique frames LM Studio as "the model under test" with an optional
   local model as judge — live-agent-behavior evaluation, structurally
   outside `internal/eval`'s Go-native, in-process model (the same category
   ADR-0017's DEC-2 already ruled out of that package).
2. The critique's own "Test Pyramid Position" diagram places this work
   directly next to "Human review" — optional, manual, above the automated
   scenario-eval layer — and separately states the core harness stays
   Go-native while Promptfoo covers the CI/external layer.

Both signals point the same direction: nothing in the evidence calls for
Go or CLI code for `SQ-003`.

## Decision

**DEC-1:** `SQ-003` resolves into a single runbook —
`docs/runbooks/local-llm-quality-review-lm-studio.{md,runbook.yaml}` —
documenting a manual, advisory-only procedure for pointing a harvested
Strategist artifact at a local LM Studio-hosted model for subjective
quality review. No code, no CLI command, no `implementation_handoff` item.
The runbook is standalone: it does not require `SQ-004` (Promptfoo,
unscoped) and instead forward-references a possible future
Promptfoo-integrated path.

**DEC-2:** LM Studio's local API shape (commonly port `1234`, OpenAI-compatible
`/v1/chat/completions`) is documented as operator-verifiable, not
workspace-confirmed — no LM Studio instance exists in this workspace to
check against, and the runbook says so explicitly.

### Alternatives Considered and Rejected

- **Design a Go/CLI integration** (e.g. a `strategist eval judge --local`
  command). Rejected: no evidence calls for it; it would duplicate the
  "optional/manual" positioning the source critique itself chose, and
  cannot be verified without a live LM Studio instance.
- **Defer until `SQ-004` is scoped**, so the runbook could reference a
  concrete Promptfoo config. Rejected: the manual procedure needs no
  Promptfoo dependency at all; deferring would block a usable deliverable
  on an unrelated, unscoped mission for no benefit.

## Consequences

- `SQ-003` closes with a real, immediately usable deliverable
  (`docs/runbooks/local-llm-quality-review-lm-studio.*`) requiring no
  further implementation authorization.
- The non-determinism warning from the source critique is preserved and
  made prominent: any local-model output is advisory only, never wired into
  `make eval` or CI.
- `SQ-004`, if scoped later, may choose to formalize this procedure as a
  Promptfoo provider config — a separate, future decision, not committed to
  here.
