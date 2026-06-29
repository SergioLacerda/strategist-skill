# ADR-0010 — Ordered contracts and mission observability

**Status:** Accepted  
**Date:** 2026-06-18

---

## Context

The Strategist had accumulated operational rules distributed across `SKILL.md`, loose contracts,
schemas, personas, and auxiliary documentation. This format produced three types of drift:

- difficulty identifying the canonical reading order of contracts
- ambiguity in the handoff between discovery and refinement
- partial observability over mission identity, generated artifacts, and gates

There was also an explicit goal of making the user experience more
deterministic: the response needs to make it clear that the agent is "inside the
mission", which contract is being followed, and which artifacts are part of the flow.

Finally, the input/output contracts needed to align with a telemetry baseline
compatible with OpenTelemetry, without forcing the consumer to infer
semantic fields from free text.

## Decision

The canonical Strategist baseline becomes:

1. `strategist/SKILL.md` acts as a thin routing shell
2. operational contracts are read on demand, in an explicit and stable order
3. the discovery → refinement flow uses canonical, unambiguous artifacts
4. the mission response envelope is externalized in a dedicated contract
5. structured events and logs carry stable attributes compatible with OTEL

### 1. Ordered contracts

Human contracts are partitioned and numbered in `strategist/contracts/`
as the canonical reading sequence:

1. `00-routing.md`
2. `01-bootstrap.md`
3. `02-intake.md`
4. `03-discovery.md`
5. `04-refinement.md`
6. `05-approval-gate.md`
7. `06-execution.md`
8. `07-adr.md`
9. `08-learning.md`
10. `09-response.md`
11. `10-telemetry.md`

`SKILL.md` does not duplicate the detailed operational rule. It only routes to
this sequence and to specific derived skills.

### 2. Canonical handoff between Ranger and Archivist

The official flow between discovery and refinement becomes:

- Ranger generates `<base_path>/pending/<mission_id>-analysis.md`
- Archivist uses that artifact as its mandatory base
- Archivist generates the package:
  - `refined/<mission_id>/proposal.md`
  - `refined/<mission_id>/design.md`
  - `refined/<mission_id>/tasks.md`

`pending/` is no longer the main artifact for this handoff. Any prior reference
that treated discovery as a draft producer in `pending/` is drift.

### 3. Mission response contract

The narrative and structural contract for the response is externalized in
`strategist/protocol.md#response-contract` and operationalized by
`strategist/contracts/09-response.md`.

Every final mission response must:

- explicitly identify the mission or its operational context
- make the mission result clear
- expose a flow compliance summary when applicable
- reflect the mission state without requiring inference by the user

### 4. Rich input/output contracts

Inputs and outputs are no longer just implicit conventions in flowing text.
They are now described by dedicated contracts and enforced by handoff schemas, response envelope, and telemetry.

The goal is not to maximize the number of fields, but to produce contracts rich
enough to:

- reduce drift between persona, runtime, and documentation
- enable automated validation
- preserve quick human readability under query

### 5. OTEL-compatible observability baseline

Strategist logs, events, and spans now prioritize stable structured attributes
for mission, component, gate, artifact, and correlation.

The baseline includes, where applicable:

- `mission_id`
- `correlation_id`
- `component`
- `selected_skill`
- `artifact_path`
- `runtime_mode`
- `output_profile`
- `gate_type`
- `gate_status`
- `gate_response`
- `approval_policy`
- `transition_group`
- `checkpoint_path`

Free text continues to be allowed as an operator message, but does not replace
the structured attributes required for correlation, audit, and analysis.

## Consequences

**Positive:**

- reduces cognitive cost of locating the correct skill rule
- eliminates ambiguity in the Ranger → Archivist handoff
- makes the response narrative predictable and auditable
- improves adherence between human contracts, schemas, and runtime
- strengthens compatibility with OTEL-based observability pipelines

**Negative:**

- increases the discipline needed to keep contracts, schemas, and embeds
  synchronized
- any future flow change must update multiple canonical surfaces
- legacy consumers that expected artifacts in `pending/` may need adjustment

## Maintenance rules

Future changes to this flow must respect:

1. Active ADRs before reinterpreting contracts
2. `strategist/contracts/` as the ordered human source
3. schemas and embeds as executable mirrors of the same contract
4. structured telemetry as a runtime requirement, not an optional detail
