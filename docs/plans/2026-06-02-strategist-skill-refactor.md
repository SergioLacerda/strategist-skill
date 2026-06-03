# Strategist Skill Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor `strategist` into smaller, objective contract files so the agent response layout becomes stable and `SKILL.md` stops accumulating phase logic.

**Architecture:** Keep `strategist/SKILL.md` as the entrypoint/router only. Move the strict response envelope into `strategist/protocol.md`, then split the large phase blocks into focused contract docs with one responsibility each. Preserve current runtime behavior; this refactor is about readability, contract clarity, and reducing drift.

**Tech Stack:** Markdown skill files, existing `strategist` contracts, focused Go tests in `strategist/tests`, `make`, `go test`.

---

## Non-Goals

- No change to mission semantics.
- No new skills or subskills.
- No change to CLI behavior unless a contract extraction exposes a pre-existing drift.
- No mandatory `make integration` gate for the first wave.

---

### Task 1: Freeze the response envelope

**Files:**
- Modify: `strategist/protocol.md`
- Modify: `strategist/tests/spec_alignment_test.go`
- Optional new test: `strategist/tests/response_contract_test.go`

**Step 1: Add a focused contract check**

Add or extend a test that asserts the Strategist response envelope has a fixed order:
1. pipeline/progress preamble
2. phase status events
3. compliance summary
4. mission result

Keep this test textual and small. It should fail if the response layout is re-ordered or if a required section disappears.

**Step 2: Confirm current drift**

Run: `go test ./strategist/tests -run TestE2EResponseContract -count=1`
Expected: fail until the contract is explicit and enforced.

**Step 3: Write the authoritative contract**

Add a short `Response Contract` section to `strategist/protocol.md` that defines:
- required header
- required section order
- mandatory footer
- blocking/approval response behavior

Reference that contract from `strategist/SKILL.md` instead of repeating the full layout inline.

**Step 4: Re-run the contract check**

Run: `go test ./strategist/tests -run TestE2EResponseContract -count=1`
Expected: pass.

---

### Task 2: Turn `SKILL.md` into a router

**Files:**
- Modify: `strategist/SKILL.md`
- Create: `strategist/contracts/routing.md`
- Create: `strategist/contracts/bootstrap.md`
- Create: `strategist/contracts/intake.md`
- Create: `strategist/contracts/mission-phases.md`
- Create: `strategist/contracts/approval-gate.md`
- Create: `strategist/contracts/adr.md`
- Create: `strategist/contracts/learning.md`

**Step 1: Add a structure check**

Add a lightweight test that verifies `strategist/SKILL.md` is compact and references the contract files instead of embedding all phase bodies inline.

**Step 2: Confirm the current file is still monolithic**

Run: `go test ./strategist/tests -run TestSkillStructure -count=1`
Expected: fail before extraction.

**Step 3: Extract the content**

Move the existing sections into dedicated files:
- `routing.md` — route selection and quick-draw detection
- `bootstrap.md` — learning buffer, bootstrap, preflight
- `intake.md` — intake, mission checkpoint, context handoff
- `mission-phases.md` — Ranger, Archivist, opportunity attack rules
- `approval-gate.md` — approval gate and execution handoff
- `adr.md` — ADR opportunity
- `learning.md` — learning phase and compliance summary

Keep the first pass behaviorally identical; only move ownership and make boundaries explicit.

**Step 4: Rewire the entrypoint**

Update `strategist/SKILL.md` so it:
- routes the mission
- names the phase order
- points at the extracted contracts
- avoids re-listing the full phase bodies

**Step 5: Re-run the structure check**

Run: `go test ./strategist/tests -run TestSkillStructure -count=1`
Expected: pass.

---

### Task 3: Reduce ambiguity inside the new contracts

**Files:**
- Modify: `strategist/skill.yaml`
- Modify: `strategist/protocol.md`
- Modify: `strategist/contracts/routing.md`
- Modify: `strategist/contracts/bootstrap.md`
- Modify: `strategist/contracts/intake.md`
- Modify: `strategist/contracts/mission-phases.md`
- Modify: `strategist/contracts/approval-gate.md`
- Modify: `strategist/contracts/adr.md`
- Modify: `strategist/contracts/learning.md`

**Step 1: Add targeted ambiguity checks**

Add checks for the most failure-prone rules:
- when opportunity attack is allowed
- when approval gate must block execution
- when ADR is skipped
- what each slot may write

**Step 2: Confirm ambiguity still exists**

Run: `go test ./strategist/tests -run TestStrategistContracts -count=1`
Expected: fail before normalization.

**Step 3: Rewrite repeated prose into rule blocks**

Normalize the new contract files with a small repeated shape:
- input
- output
- stop condition
- write scope

Keep slot authority explicit in the file that owns it; do not duplicate the same rule in multiple files.

**Step 4: Re-run the ambiguity check**

Run: `go test ./strategist/tests -run TestStrategistContracts -count=1`
Expected: pass.

---

### Task 4: Validate against the current runtime surface

**Files:**
- Test: `strategist/tests/spec_alignment_test.go`
- Test: `strategist/tests/*.feature`
- Test: `tests/e2e_cli_happy_path_test.go`

**Step 1: Run the local-focused checks**

Run:
`make test-lite`
`go test ./strategist/tests -run TestE2EFeatureFilesCoverHappyPathContracts -count=1`

Expected: pass.

**Step 2: Confirm text/runtime alignment**

Verify the `.feature` specs still match the revised strategist layout and phase names.

**Step 3: Run the happy-path CLI tests only if needed**

If the refactor touches the contract envelope in a way that could affect CLI text, re-run:
`go test -tags=integration ./tests/... -run TestE2E -count=1`

This is a follow-up validation, not a required gate for the refactor itself.

**Step 4: Commit the refactor**

Commit once docs and targeted tests agree.

