# ADR-0003 — Approval gate mandatory and never bypassable

**Status:** Accepted  
**Date:** 2026-05-28  
**Context:** Agent security guardrails (guardrails-20260529)

> 2026-06-26 note: the gate remains mandatory, but current Strategist execution is
> documentation/materialization/handoff work. Source-code mutation is outside the
> Strategist contract even after approval.

---

## Context

The Strategist orchestrates an execution agent (Sniper) that operates in code repositories with the potential to modify files, run scripts, and change configurations. Without control, any user input could trigger immediate execution.

The central question: should the approval gate be a **configurable preference** or a **system invariant**?

Alternatives considered:
- **Configurable gate** — allow `auto_approve: true` in `active.yaml` for CI pipelines
- **Gate by risk_score** — gates only for high-risk executions, free for low risk
- **Always mandatory gate** — no execution without explicit human approval, no exceptions

## Decision

The approval gate is **mandatory_pause** — a system invariant, not a configuration. It is declared as `type: mandatory_pause` in `skill.yaml` and as a forbidden behavior in `protocol.md`:

```yaml
forbidden_behaviors:
  - invoke_execution_slot_without_approval
```

Any path that reaches the execution slot without an affirmative user response is a bug, not a feature. If the gate is denied or there are no materialization tasks, the valid result is to deliver the analysis/refinement without executing Sniper.

The only foreseen exception is `sdd_injection`, which can inject the execution provider but cannot remove the gate.

## Consequences

**Positive:**
- Eliminates an entire class of "unintended execution" bugs — the agent can never act without the human having seen the plan
- Predictable behavior regardless of the provider configured in the execution slot
- Simplifies security reasoning: any path that reaches Sniper without an explicit gate is detectable as a violation
- Test fixtures (`approval-bypass.yaml`) encode the invariant as executable spec

**Negative:**
- Fully automated CI/CD is not possible with the skill in default mode — requires human intervention on every mission
- "Batch processing" flows need a different approach — the Strategist is not the right tool for unsupervised automation
- May seem excessive for small tasks, but the cost of a "yes" is less than the cost of an unintended execution
