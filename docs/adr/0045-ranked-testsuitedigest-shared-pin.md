# ADR-0045 — Ranked `TestSuiteDigest`: Shared Pin Documented as Honest Interim, Not a Per-Role Guarantee

**Status:** Accepted
**Date:** 2026-09-16
**Context:** `20260916-second-ranked-pairing-archivist-openspec-propose`

## Context

ADR-0043's DEC-006 amendment required the Ranked-certification
evidence-computation code
(`internal/install/embedded_skill_conformance.go`) to be generic over
`(role, provider)`, anticipating a second pairing beyond the pilot
(Ranger↔`brainstorming`). `hostAPIContractDigest(defaultsRoot, role)` is
genuinely role-parameterized — it reads `roles/<role>.yaml` +
`internal_skills/<role>/SKILL.md`, real content specific to whichever role
is being certified. `connectorDigest()` is a fixed pin
(`internal/plugins/connectors/runtime_connector.go`), and honestly so: it
is Strategist's own shared native connector implementation, the same for
every role by construction — no role-specific content exists to pin
instead.

`testSuiteDigest()` is also a fixed pin
(`internal/handoff/role_provider_conformance_test.go`), but unlike
`connectorDigest()`, this is **not** honestly role-agnostic content. Read
in full during this mission's discovery: the file's actual tests exercise
`handoff.RangerToArchivistPolicy()` specifically
(`TestHandoffRequiredChallengeTypesMatchRangerRoleContract` and its
siblings) — it verifies Ranger's own outgoing-handoff contract, not a
generic "any role" conformance suite. No `ArchivistToGatePolicy()` or
equivalent exists anywhere in `internal/handoff/` to test Archivist's own
downstream handoff obligations.

Adding a second pairing — Archivist↔`openspec-propose` — means this same
pinned file becomes that pairing's own `TestSuiteDigest` "contract test"
evidence too, despite testing something unrelated to Archivist.

## Decision

- **DEC-001.** Ship the second Ranked pairing now, using the existing
  shared `TestSuiteDigest` pin — do not block on writing a real
  Archivist-specific conformance suite first.
- **DEC-002.** Correct `testSuiteDigest()`'s doc comment to state the
  limitation plainly: this digest is a pinned code-freshness marker shared
  across every Ranked pairing today (any change to the pinned file
  invalidates every existing certification, regardless of role), not a
  role-specific contract-test guarantee. Ranger's own certification
  already carried this same limitation, silently, before this ADR — this
  decision makes it explicit rather than newly introducing it.
- **DEC-003.** Name, but do not build, the real fix: a genuine
  Archivist-specific conformance test suite plus a role-parameterized
  `testSuiteDigest()` (mirroring `hostAPIContractDigest`'s already
  role-aware `map[string]string`-of-role pattern), tracked as
  `.analysis/pending/20260916-archivist-conformance-test-suite.md`.

## Consequences

### Positive

- The second Ranked pairing is not blocked on a larger, unscoped piece of
  work (specifying and testing Archivist's own downstream handoff
  contract) that nobody has asked for yet.
- The evidence's actual meaning is now documented accurately at its
  source (`testSuiteDigest()`'s own doc comment), rather than silently
  implying a guarantee it does not provide — a future reader inspecting
  Archivist's `TestSuiteDigest` will not be misled about what it proves.
- DEC-003's named follow-up gives the real fix a concrete home instead of
  leaving the limitation as tribal knowledge in this ADR alone.

### Negative

- Until DEC-003 ships, Archivist's Ranked certification's `TestSuiteDigest`
  dimension provides no role-specific assurance — it is evidence that *a*
  test file exists and is unchanged, not that Archivist's own conformance
  is tested. This is a real, accepted gap, not a cosmetic one.
- A future contributor changing `role_provider_conformance_test.go` to
  genuinely cover multiple roles would not be automatically prompted to
  revisit this ADR's own DEC-002 comment — no test enforces that link.

## Rejected Alternatives

- **Option 2 (build first): write a real Archivist-specific conformance
  test suite and make `testSuiteDigest()` role-parameterized before
  certifying the pairing.** Rejected for this mission — more correct, but
  requires first specifying Archivist's own downstream handoff obligations
  precisely enough to write fixtures against, work with no existing scope
  or owner. Named as DEC-003's follow-up instead of blocking on it.
- **Option 3 (block): do not certify the second pairing until Option 2
  ships.** Rejected — the most conservative option, but delays real,
  independently-valuable pilot expansion (the Wizard offering a certified
  Ranked choice for the refinement slot) behind unscoped future work, with
  no correctness benefit over Option 1's honestly-labeled interim.

## References

- `.analysis/pending/cli_refactor/20260916-second-ranked-pairing-archivist-openspec-propose/`
  — the refined package this ADR documents (analysis.md KF-006, UNC-001).
- `docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md` DEC-006
  — the generic-evidence-computation requirement this ADR's DEC-002/003
  respond to.
- `internal/handoff/role_provider_conformance_test.go` — the pinned file
  whose Ranger-specific content motivates this ADR.
- `.analysis/pending/20260916-archivist-conformance-test-suite.md` —
  DEC-003's named follow-up.
