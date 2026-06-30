# ADR-0007 — Structural Compression Pipeline — Agent Contract vs Go Runtime

**Status:** Accepted
**Date:** 2026-06-02

---

## Context

The token economy design specified Phase 4 as a Go `CompressionProvider` interface with a `builtin-structural` provider. This component would sit between source retrieval and LLM invocation, deterministically producing scored source cards.

At decision time, `context-enrichment/skill.yaml` already declared `source_cards` and `compression_metrics` as output fields in the agent contract. No Go implementation existed in `internal/`.

Context that led to the evaluation: the context-enrichment pipeline fetches knowledge sources by `task_type` and delivers them to the LLM. Without structured compression, the LLM receives all sources within the token budget and decides internally which to prioritize. The alternative would be a Go component that scores and filters sources before passing them along — guaranteeing determinism and auditability at the cost of maintenance complexity.

## Decision

Phase 4 was deferred. Structural compression is implemented as a **behavioral agent contract** via `context-enrichment/skill.yaml`, not as Go runtime.

The LLM implements semantic compression natively by following the skill contract (score by `task_type` match, trust tier, keyword overlap; apply budget limits per mode; produce source cards in evidence → interpretation → impact format).

The Go `CompressionProvider` interface remains in the backlog — to be considered when:
1. Phase 2 (chest index files) is complete and provides deterministic ranking inputs
2. Non-deterministic agent compression causes measurable quality regression
3. Audit/testability requirements demand reproducible compression

## Consequences

**Accepted trade-offs:**
- Compression behavior is non-deterministic and not unit-testable
- No fallback guarantee if the LLM ignores the contract
- Metrics in `chest-usage.jsonl` are self-reported by the agent, not computed

**What gets easier:**
- Zero Go maintenance cost for compression
- Contract evolves in YAML without recompilation
- Works with all LLM providers that follow the skill contract

**What gets harder:**
- Auditing exactly which sources were selected and why
- Testing compression quality regression
- Deterministic offline behavior (no network/LLM)
