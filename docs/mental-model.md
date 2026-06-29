# Strategist — Mental Model

**Status:** Accepted
**Last Updated:** 2026-06-26

## One sentence

Strategist is a **project manager for AI analysis work**: it breaks your request into
phases, delegates each phase to a specialist, and blocks at an approval gate before
approved documentation materialization or handoff work begins.

---

## The problem it solves

Without governance, AI agents drift:

- They expand scope beyond what you asked
- They execute before you've reviewed the plan
- They make decisions you can't audit or roll back
- Multiple agents interfere with each other's work

Strategist adds a mandatory checkpoint between "analysis" and "materialization" — the
**approval gate**. Source-code changes are outside the current Strategist contract.
Documentation and analysis artifacts are the normal output.

---

## The pipeline

```
You → [Ranger]    → discovers requirements, maps context
      [Archivist] → refines into an actionable specification
      [Gate]      → STOPS and waits for your explicit approval
      [Sniper]    → materializes exactly the approved documentation/handoff
```

Each phase writes versioned artifacts to `.analysis/` so the process is auditable
and resumable. If something goes wrong mid-execution, you can see exactly where it stopped.

### What each role does

| Role | Neutral name | Responsibility |
|------|-------------|----------------|
| Strategist | Orchestrator | Coordinates the mission; guards the gate |
| Ranger | Discoverer | Gathers requirements and maps the terrain |
| Archivist | Spec Writer | Turns the discovery report into an actionable spec |
| Sniper | Executor | Materializes the approved documentation/handoff — surgically, minimally |
| Wizard | Installer | Installs the party into your repository |

---

## The approval gate is not bureaucracy

The gate is the only inviolable rule. Here's why it exists:

An AI agent that executes without review is making decisions on your behalf that you
can't verify until after the damage is done. The gate creates a moment where:

1. You see the full plan before approved documentation or handoff files are touched
2. You can add constraints, change scope, or cancel
3. The agent has explicit evidence that you approved _this specific plan_

`execution_gate=allowed` from governance is a **precondition**, not approval.
Approval is you typing `sim` at the gate prompt.

---

## Strategist vs. alternatives

| | Direct Claude request | CI/CD pipeline | Strategist |
|---|---|---|---|
| Human sees plan before execution | ❌ | ❌ | ✅ |
| Auditable artifacts | ❌ | Partial | ✅ |
| Scope enforcement | ❌ | Manual | ✅ |
| Resumable after interruption | ❌ | ✅ | ✅ |
| Works with any LLM | ✅ | ✅ | ✅ |
| Requires setup | ❌ | ✅ | Minimal |

---

## What Strategist is not

- **Not an agent framework** — it's an orchestration contract that any agent can follow
- **Not a CI/CD tool** — it works inside your IDE session, not a pipeline runner
- **Not opinionated about LLMs** — the Ranger/Archivist/Sniper slots accept any model
- **Not magic** — it's a structured protocol written in YAML and Markdown that your AI reads

---

## The artifact trail

Every mission leaves a trail in `.analysis/`:

```
.analysis/
  pending/
    <mission_id>-analysis.md    ← Ranger output (frontmatter: mission_status)
  refined/
    <mission_id>/
      analysis.md               ← Archivist-promoted analysis
      proposal.md               ← Archivist: what and why
      design.md                 ← Archivist: how
      tasks.md                  ← Archivist: approved-materialization checklist
  archived/
    <mission_id>-report.md      ← Sniper execution report
```

The `mission_status` frontmatter field in the analysis file acts as a **distributed lock**:
a Sniper checks and sets it to `sniper_running` before materialization, preventing
two agents from executing the same mission simultaneously.

---

## Quickstart mental map

```
"I want to add feature X"
        ↓
  /strategist <description>
        ↓
  Ranger maps context → writes analysis.md
        ↓
  Archivist refines → writes proposal + design + tasks
        ↓
  🚦 Gate: review the plan, type "sim" to approve
        ↓
  Sniper materializes approved documentation/handoff tasks 1-by-1, updates mission_status
        ↓
  Done — report in .analysis/archived/
```

That's it. The lore is optional. The gate is not.
