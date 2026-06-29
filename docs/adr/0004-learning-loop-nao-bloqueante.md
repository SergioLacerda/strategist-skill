# ADR-0004 — Non-blocking learning loop

**Status:** Accepted  
**Date:** 2026-05-28  
**Context:** Learning phase and learning-curator design

---

## Context

After each mission, the Strategist proposes recording the outcome (`memory/outcomes.jsonl`) and adjusting knowledge source priorities (`memory/source-hints.yaml`). This phase involves: running `response-critic`, running `learning-curator`, presenting a checkpoint to the user, and waiting for a response.

Any of these steps can fail: LLM timeout, user ignores the checkpoint, sub-skill returns an error, memory file is corrupted.

The question: if learning fails, what happens to the mission result?

Alternatives considered:
- **Blocking** — mission only returns after learning is complete; learning failure fails the mission
- **Configurable optional** — user chooses whether to enable learning via `active.yaml`
- **Non-blocking** — learning runs after the mission; failure does not alter the result delivered to the user

## Decision

The learning phase is **non-blocking**: it runs after execution, but any failure (timeout, sub-skill error, user declines the checkpoint) results in a log and the mission result being returned unchanged.

Declared in `protocol.md` and `skill.yaml`:
```yaml
- stage: learning
  skill: learning-curator
  blocking: false
```

`learning-curator` has forbidden behavior `block_mission_result_on_learning_failure` — the sub-skill cannot withhold the mission result while waiting for learning.

The learning buffer (`memory/outcomes.tmp`) has a limit of 20 entries before being flushed to `outcomes.jsonl` — protection against infinite accumulation if the main flush never occurs.

## Consequences

**Positive:**
- The user always receives the mission result — memory infrastructure failure never blocks work
- Simplifies mission reasoning: the result is deterministic regardless of the memory state
- Learning is an accumulated benefit, not a requirement for operation — system degrades gracefully
- Learning failures are observable via log without impacting the main flow

**Negative:**
- If learning fails systematically (e.g. corrupted memory file), the problem may go unnoticed across many missions without the user realizing
- The learning buffer is a second write path for outcomes — two paths to the same data can cause duplicates if the main flush and buffer flush coincide
- A user who repeatedly ignores the checkpoint loses the accumulated benefit of the knowledge system without explicit feedback
