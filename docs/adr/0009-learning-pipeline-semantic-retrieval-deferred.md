# ADR-0009 — Semantic Retrieval Deferred In Learning Pipeline

**Status:** Accepted  
**Date:** 2026-06-07

---

## Context

The Strategist already has:

- an outcomes buffer at `.strategist/memory/outcomes.tmp`
- governed flush to `.strategist/memory/outcomes.jsonl`
- `source-hints.yaml` as a manual overlay
- a non-blocking learning phase

The refined analysis `2026-06-07-learning-pipeline-embeddings` concluded that the current gap
is not "absence of embeddings" but rather insufficient maturity of the historical corpus.

In the current workspace state:

- `outcomes.tmp` contains `12` entries
- `outcomes.jsonl` does not yet show an observable consolidated corpus
- `source-hints.yaml` has no learned hints

There is no evidence that tag-based and lexical retrieval already fail often enough
to justify semantic indexing.

## Decision

Semantic retrieval via embeddings is **explicitly deferred**.

The canonical baseline remains:

1. structured outcomes in `outcomes.jsonl`
2. retrieval by tags and hints
3. lexical search over the historical corpus

Any future semantic index must obey these rules:

- it is optional, never mandatory
- it is derived from `outcomes.jsonl`
- it is local and rebuildable
- its failure never blocks the mission
- `context-enrichment` must degrade to tags or lexical search

## Consequences

**Positive:**

- avoids premature operational complexity
- keeps the learning pipeline auditable and simple
- forces a benchmark-based decision rather than a hypothesis-based one
- preserves a robust fallback with no dependency on an external model

**Negative:**

- cross-mission retrieval remains limited to tags, hints, and lexical search
- semantic similarity is not available in the short term
- a future benchmark still needs to be designed and run once the corpus matures

## Reevaluation Criteria

Reopen the decision only when all criteria below are satisfied:

1. `outcomes.jsonl` has at least `50` real missions
2. there are at least `3` documented cases where tags or hints failed
3. a benchmark exists comparing tags, lexical, and semantic retrieval
4. local operation of the semantic index is explicitly accepted
