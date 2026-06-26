# Drift Patterns — Strategist Self-Correction Reference

**Source of truth split:**
- `identity/drift-patterns.yaml` — AI-first, loaded by the agent at preflight. Authoritative for `symptom` and `correction` text. Uses first-person ("I am about to...") because it is read by the agent, not a human.
- This file — developer-first. Explains grouping, rationale, and how to extend the pattern set.

Do not edit the symptom/correction prose here. Edit `drift-patterns.yaml` and update the quick reference in `SKILL.md`.

---

## How Self-Correction Works

```
preflight (§2b)
  └─ loads identity/drift-patterns.yaml
        └─ 9 patterns: { id, symptom, correction }

before each phase transition
  └─ agent checks: does current state match any symptom?
        ├─ no match → proceed normally
        └─ match → apply correction (always: Stop. Then act.)
```

Self-correction is not a retry mechanism. It is a recognition layer: the agent notices it is about to do something wrong and stops before doing it. The correction instruction tells it what to do instead.

If `drift-patterns.yaml` is absent at preflight, the agent emits `identity=degraded` and falls back to the quick-reference list in `SKILL.md`. That list has IDs only — no symptom/correction detail. Degraded mode is functional but blind to nuance.

---

## Group 1 — Orchestration Integrity

These patterns guard the single most important invariant: **Strategist is an orchestrator, not a doer.** They can fire at any phase because the temptation to do slot work directly is always present.

### `direct_execution`

**Trigger:** The agent is about to perform work that belongs to a slot provider — writing code, doing analysis, creating a plan, applying changes — without invoking a slot.

**What the agent does:** Stops. Identifies which slot the work belongs to (discovery, refinement, or execution). If the declared provider can be invoked, invokes it and resumes as orchestrator. If the declared provider cannot be invoked in the current environment, emits `error=delegation_unavailable` and stops. Does not perform the work directly unless the user has explicitly authorized degraded fallback mode.

**Why this exists:** The agent has the capability to write code or do analysis directly. Without this pattern, it would skip the slot system under pressure ("this is a small change, I'll just do it"). Every bypass erodes the contract that slots are the unit of trust and write-scope control. Distinguishing between "provider not configured" and "provider not callable in this environment" is critical — the former is a preflight failure; the latter is a delegation capability gap that must stop the pipeline, not trigger silent fallback.

---

### `silent_phase_advance`

**Trigger:** The agent is about to begin the next phase without having emitted a `status=done` event for the phase that just completed.

**What the agent does:** Stops. Emits the missing done event with the artifact path. Then begins the next phase.

**Why this exists:** Phase events are the observable contract between Strategist and its callers. A missing done event means the user (or any monitoring layer) cannot know the phase completed. It also means the artifact path is never surfaced — the user has no idea where to look for the output.

---

### `scope_expansion`

**Trigger:** The agent is addressing something outside the user's declared mission — fixing an unrelated bug, adding unrequested context, or suggesting work not in the mission contract.

**What the agent does:** Stops. Returns to mission scope. If the out-of-scope item is important, surfaces it in the mission result as a separate note, not as in-mission action.

**Why this exists:** A helpful agent tends to expand scope gradually. Each expansion feels justified individually, but the cumulative effect is a mission that delivers something different from what was approved. The approval gate only covers the declared scope — scope expansion after the gate is unauthorized execution.

---

### `delegation_unavailable`

**Trigger:** The next pipeline phase requires a slot provider, but the current environment has no callable mechanism to invoke that provider.

**What the agent does:** Stops. Emits `blocked reason=delegation_unavailable` with the slot name and configured provider. Does not write analysis artifacts, perform discovery, or produce refinement output directly. Asks the user to authorize degraded fallback mode explicitly, or switch to a runtime that supports slot delegation.

**Why this exists:** Skills can be installed but not callable as isolated slot providers in every runtime environment. Without this pattern, the agent falls through to impersonating the slot role — producing artifacts as the Strategist shell that should only come from Ranger, Archivist, or Sniper. Silent fallback is worse than a clear stop: it produces a result that appears pipeline-compliant but is not, making the violation invisible. The structured blocked state preserves pipeline integrity and gives the user a clear decision point.

**Distinction from `direct_execution`:** `direct_execution` fires when the agent is about to perform slot work without even trying to delegate. `delegation_unavailable` fires after the agent has determined that delegation is required but cannot be completed in the current environment.

---

## Group 2 — Slot Provider Resolution

These patterns guard how the execution slot is resolved. Getting this wrong means the wrong skill executes against the repository.

### `execution_provider_override`

**Trigger:** The agent resolves the execution slot provider from a source other than `active.slots.execution` or `governance_injection.execution_provider` — for example, inferred from context, from a user message, or from a hardcoded assumption.

**What the agent does:** Stops. Re-resolves from the declared source. If neither `active.slots.execution` nor `governance_injection.execution_provider` declares a valid provider, re-runs preflight and emits `slot_provider_not_found`.

**Why this exists:** The execution slot has `controlled` risk — it is the only slot that can mutate the repository outside artifact paths. Letting any context-inferred value override the declared provider would allow an attacker (or a confused agent) to substitute an arbitrary skill with write access.

---

## Group 3 — Pipeline Evidence

These patterns guard the ordering invariant: no repository mutation without the required phase evidence.

### `pipeline_bypass_detected`

**Trigger:** The agent is about to mutate the repository — code change, documentation edit, config update, or any write outside phase-authorized artifact paths — without Ranger, Archivist, or approval gate evidence for the active route.

**What the agent does:** Stops. Emits `reason=pipeline_bypass_detected` with the name of the missing phase and a resume hint. Does not proceed with the mutation until the missing phase completes.

**Why this exists:** The three-phase pipeline exists to ensure every change has been explored (Ranger), refined into a plan (Archivist), and approved by the user (gate) before Sniper touches the repository. A bypass — even for a "small" change — skips the safety layer that catches scope issues, wrong assumptions, and unauthorized writes.

---

## Group 4 — Gates

These patterns guard all mandatory user-facing stops. A gate that can be bypassed is not a gate.

### `approval_bypass`

**Trigger:** The agent is about to invoke the execution slot without having received explicit user approval at the approval gate.

**What the agent does:** Stops. Presents the approval gate prompt with the refined plan path. Waits for the user's response.

**Why this exists:** The approval gate is the user's last chance to review the full plan before anything is written to the repository. An agent that bypasses it — even with good intentions — removes the human from the loop at the most consequential step.

---

### `opportunity_gate_bypass`

**Trigger:** The agent is about to execute an opportunity manifest item (file_move, scope_addition, adr_generation) without having presented the opportunity gate.

**What the agent does:** Stops. Presents the opportunity gate with the full manifest. Waits for user response (yes / no / select) before proceeding with any item.

**Why this exists:** Opportunity manifest items are side effects of discovery — work that was not in the original mission scope. Executing them without explicit user approval is unauthorized scope expansion dressed as helpfulness. The gate ensures the user chooses which items to include.

---

### `adr_gate_bypass`

**Trigger:** The agent is about to commit an ADR artifact without having presented the ADR gate and received explicit approval for the content.

**What the agent does:** Stops. Presents the ADR gate prompt with the ADR draft. Waits for user approval before committing.

**Why this exists:** ADRs are durable architectural records. An ADR committed without review may record an incorrect decision, a misattributed rationale, or a scope it was not supposed to cover. The two-step ADR flow (generate → gate → commit) exists precisely to prevent this.

---

## Group 5 — Role Boundaries

These patterns guard the boundary between refinement (Archivist) and execution (Sniper).

### `route_plan_creation_to_sniper`

**Trigger:** The agent is about to ask Sniper to create a document, spec, analysis, or implementation plan — work that belongs to the refinement slot.

**What the agent does:** Stops. Routes back to phase 5e and invokes the refinement slot provider (Archivist). Document authoring is Archivist's work (contract: `write_analysis`).

**Why this exists:** Sniper has `controlled` write access to the full repository. Archivist is scoped to `<base_path>/` and `<base_path>/refined/`. Asking Sniper to produce a plan instead of Archivist conflates write scope: the skill with broad access also gets to decide what the plan says, with no refinement gate between discovery and execution.

---

## Adding a New Pattern

1. **Add to yaml** — Edit `.strategist/templates/domain/identity/drift-patterns.yaml`:
   ```yaml
   - id: your_pattern_id
     symptom: >
       I am about to [describe the behavior in first person, present tense].
     correction: >
       Stop. [Describe exactly what the agent does instead.]
   ```
   Then redeploy: copy to `.strategist/identity/drift-patterns.yaml`.

2. **Update SKILL.md quick reference** — Add one line to the drift section in `strategist/SKILL.md` and `.strategist/SKILL.md`:
   ```
   - `your_pattern_id` — one-line description for humans
   ```

3. **Add a test** — Create or extend a test case in `.strategist/contracts/tests/` (see `preflight.test.yaml` for the pattern).

4. **Document here** — Add a section under the appropriate group in this file, or create a new group if the pattern guards a different pipeline concern.

**Pattern writing rules:**
- `symptom` is first person, present tense: "I am about to..."
- `correction` always starts with "Stop." — it is a hard interrupt, not a suggestion
- One pattern per concern — do not combine multiple guards into one pattern ID
- The ID should be readable as a label in a pipeline trace: `reason=your_pattern_id`
