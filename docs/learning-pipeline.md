# Learning Pipeline

**Status:** Accepted  
**Last Updated:** 2026-06-07

## Current Baseline

The current learning pipeline remains intentionally simple:

- mission outcomes are appended to `.strategist/memory/outcomes.tmp`
- the buffer is flushed into `.strategist/memory/outcomes.jsonl`
- `outcomes.jsonl` is the historical source of truth
- `source-hints.yaml` remains the manual priority overlay

Semantic retrieval is not part of the required runtime baseline.

## Corpus Readiness Snapshot

Snapshot taken on 2026-06-07 in the current workspace:

- `.strategist/memory/outcomes.tmp`: `12` entries
- `.strategist/memory/outcomes.jsonl`: `0` entries observed
- `.strategist/memory/source-hints.yaml`: `0` learned hints

Conclusion: the corpus is not mature enough to justify a semantic index yet.

## Structured Outcome Contract

The preferred historical record shape is defined in:

- `strategist/schemas/outcome-entry.schema.yaml`

This schema is additive. Current producers may still emit the minimum fields
required by the protocol while evolving toward richer structured outcomes.

## Retrieval Benchmark Policy

Any future retrieval benchmark must compare at least:

1. tag and hint based retrieval
2. lexical search over `outcomes.jsonl`
3. semantic retrieval using a local derived index

Minimum evaluation dimensions:

- top-3 usefulness
- cold latency
- warm latency
- index size on disk
- rebuild cost and operational complexity

## Storage Decision

Current recommendation:

- baseline now: `JSONL + tags/hints + lexical retrieval`
- optional future target: local embedded semantic store only if benchmark proves value
- rejected for current stage: FAISS/HNSW class solutions

## Activation Criteria

Semantic retrieval stays deferred until all conditions are true:

1. at least `50` real mission outcomes in `outcomes.jsonl`
2. at least `3` concrete retrieval failures using tags or hints
3. benchmark evidence of material gain
4. explicit decision to operate a local semantic index
