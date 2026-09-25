# ADR-0043 — Ranked Pipeline Pilot: Catalog Fields, Wizard UX, and Pre-Generated Runtime Binding

**Status:** Accepted
**Date:** 2026-09-16
**Context:** `20260916-ranked-pipeline-implementation-plan`

## Context

ADR-0041 named the Ranked pipeline ("classe rankeada" — a Role bound to an
embedded weapon at compile time) and ADR-0042 decided its persistence
contract (a Ranked binding's runtime record, if any, is optional,
non-authoritative redundancy; only Custom's record is mandatory). Neither
ADR designed the actual implementation: what catalog fields mark a binding
as certified, how the Wizard presents the choice, and how validation avoids
running Custom-only checks against a Ranked binding.

`internal/domain/role_invocation_plan.go` and
`internal/rolevalidation/role_weapon.go` already contain explicit,
fail-closed rejection points for `mode: ranked` (added while implementing the
`mode` discriminator, before this ADR) — nothing produces or resolves a
Ranked binding today. This ADR scopes a pilot: prove the pipeline end to end
for exactly one pairing, Ranger↔brainstorming, the project's existing
permanent embedded pairing (ADR-0035).

## Decision

- **DEC-001. `default` and `ranked`/`certification_digest` are independent,
  coexisting catalog fields — never alternatives.** `default: true` keeps
  its ADR-0034 meaning ("preferred among interchangeable candidates, no
  extra guarantee"). `ranked: true` + `certification_digest` is a separate,
  stronger claim ("build-time certified"). The Ranger↔brainstorming catalog
  entry carries all three simultaneously, because it is both today's
  preferred candidate and the first to be certified. A future entry could
  have `ranked: true` without `default: true`, or vice versa — the two
  questions ("preferred?" and "certified?") stay independently answerable.
  This also fixes the Wizard-visible model for a slot: exactly two named
  choices, "Ranger rankeado" (the certified `default`+`ranked` entry) and
  "Customizado" (the existing full candidate list, including any
  third-party external skill the user supplies).

- **DEC-002. The Wizard is where the binding choice is made; the runtime
  lock is where it is reinforced afterward — never the other way round.**
  When a slot has a certified Ranked binding, the Wizard's menu pre-selects
  "Ranger rankeado" (reusing the existing `Default`-preselection path in
  `wizard_prompts.go`) while still listing "Customizado"'s full candidate set.
  The user must see and confirm the choice in the Wizard before anything is
  written — this is pre-selection, not auto-apply, per the pending draft's
  own rule that Ranked must never be a silent fallback. Once chosen, the
  decision is persisted to `.strategist/plugins.lock` as reinforcement, read
  back later by `strategist check`/`RoleInvocationPlan`.

- **DEC-003. A Ranked binding skips Custom-only runtime validation
  entirely.** `rolevalidation.persistedSlotBinding`'s manifest/risk_score/
  role-affinity checks and `check_trust_and_grants.go`'s trust-policy/
  permission-grant evaluation are Custom-specific (ADR-0041's pipeline-scope
  decision: Ranked never calls into Custom's runtime checks). A Ranked
  binding is validated against its catalog certification stamp instead —
  it is already validated and homologated at build time and needs no new
  runtime validation. Shipping the Wizard option without this would make a
  Ranked selection immediately fail `strategist check` via the existing
  fail-closed rejection — a regression, not a feature.

- **DEC-005. The runtime binding record is pre-generated at certification
  time, not constructed by the Wizard.** Since a Ranked binding is already
  validated and certified at build time (DEC-003's premise), the
  `.strategist/plugins.lock` fragment it would need
  (`domain.SlotBinding{Mode: SlotBindingModeRanked, InstalledInstanceID, ...}`)
  is computed once during `strategist plugin prepare-embedded`'s
  certification pass and shipped alongside the catalog's certification
  stamp. The Wizard's Ranked path becomes an activate/copy step, not a
  binding-construction step — simplifying the Wizard flow, per the explicit
  request that motivated this decision. Per ADR-0042 DEC-003, pre-generating
  this record does not make it load-bearing: it remains optional,
  non-authoritative redundancy regardless of when it is computed.

(DEC-004 — certification runs inside the existing
`strategist plugin prepare-embedded` step, not a new command — was already
implicit in DEC-005 and is recorded here for completeness: that step already
resolves, verifies, and merges packages into the catalog
(`internal/install/embedded_weapon_prepare.go`); the certification pass and
DEC-005's pre-generation both extend it rather than introducing a second
build-time entry point.)

## Consequences

### Positive

- Exactly one pairing (Ranger↔brainstorming) proves the full pipeline before
  any generalization — the certification/validation logic is keyed by
  catalog id/digest, not hardcoded, so extending to other pairings later
  does not require redesign.
- `default` and `ranked` staying independent avoids retroactively
  strengthening ADR-0034's "no extra guarantee" meaning for every existing
  catalog entry that already sets `default: true`.
- Pre-generating the runtime record (DEC-005) removes an entire class of
  Wizard-side construction bugs for the Ranked path — there is nothing to
  get wrong beyond "activate the record the build already produced."

### Negative

- All actual code changes (catalog schema, `prepare-embedded` certification
  pass, Wizard menu, `RoleInvocationPlan`/`rolevalidation`/
  `check_trust_and_grants.go` branches) remain `implementation_handoff`
  items requiring a separately authorized execution provider — this ADR
  authorizes no code changes.
- A future catalog entry with `ranked: true` but not `default: true` is
  possible under DEC-001 but not exercised by this pilot (only
  Ranger↔brainstorming, which has both) — its Wizard/validation behavior is
  unverified until a second pairing is attempted.

## Rejected Alternatives

- **Treat `ranked` as replacing `default`'s role once certification
  exists.** Rejected — the user explicitly requested both fields stay
  independently meaningful, not one subsuming the other.
- **Construct the Ranked `SlotBinding` in the Wizard at selection time,
  parameterized by the certification stamp.** Rejected — the user explicitly
  asked to pre-generate the runtime files at certification time specifically
  to simplify the Wizard flow; constructing it at Wizard time re-adds the
  complexity this decision removes for no stated benefit.
- **Defer the Custom-only-validation-skip (DEC-003) to a follow-up mission,
  ship the Wizard option first.** Rejected — a Ranked selection would
  immediately fail `strategist check`'s existing fail-closed rejection,
  shipping a broken feature instead of no feature.

## Scope Boundary

This ADR records the catalog-field, Wizard-UX, and runtime-binding
pre-generation decisions for the Ranked pipeline pilot only. It does not
authorize implementation of the certification pass, the Wizard menu change,
or the `RoleInvocationPlan`/`rolevalidation`/`check_trust_and_grants.go`
branches — all remain `implementation_handoff` items outside Strategist's
execution scope
(`.analysis/pending/cli_refactor/20260916-ranked-pipeline-implementation-plan/tasks.md`
groups 1–3), requiring a separately authorized coding task. It does not
extend Ranked to any pairing beyond Ranger↔brainstorming, and it does not
revisit ADR-0041's naming decision or ADR-0042's persistence-authority
decision, both unchanged and built on here.

## Amendment — Generic Evidence Computation for Future Ranked Pairings (added 2026-09-16, mission `20260916-conformance-wiring-and-adr0029-t2-decisions`)

**DEC-006.** A follow-up decision connects `internal/plugins/conformance/conformance.go`'s
`CertificationRecord`/`EvaluateCertification` (C0-C3 conformance levels) to
the Ranked pipeline's certification pass. `CertificationRecord.Validate()`
requires five digest fields unconditionally: `PackageDigest`,
`AdapterDigest`, `HostAPIDigest`, `ConnectorDigest`, `TestSuiteDigest`. The
last three have no computation anywhere in the codebase today, for any
provider — connecting them for real (not fabricated placeholders, which
this project never does) is genuinely new evidentiary work, not a small
wiring task, and this ADR's own DEC-004/005 pilot scope did not anticipate
it.

This work is authorized to proceed, **on the explicit condition that it is
designed generically over `(role, provider)` from the start** — not as a
second hardcoded special case alongside Ranger↔brainstorming. This
pairing was never meant to be the only one: Archivist↔`openspec-propose`
is an already-planned next Ranked pairing, and there is a longer-term
ambition to generalize part of the mission pipeline itself around this
same certification mechanism (that generalization is a future mission's
subject, not designed here — noted so this decision is not read as
foreclosing it).

Concretely, this means:

- `rankedCertificationPairs` (`internal/install/embedded_weapon_certification.go`)
  may remain a declarative, explicit allow-list — DEC-004's "one pairing at
  a time, deliberate addition" principle is unchanged and still applies to
  *which* pairings get certified.
- `validateRankedCandidate`, `rankedCertificationDigest`, and the new
  HostAPI/Connector/TestSuite evidence-computation functions this decision
  adds must all take `(role, provider)` as parameters and contain no
  `brainstorming`- or `ranger`-specific logic. The existing code already
  mostly satisfies this (only the pairs map itself is pairing-specific) —
  this constrains the *new* functions, not a rewrite of what exists.

*Rejected alternative:* ship a narrower, Ranked-pilot-specific
certification record type instead of extending `conformance.go`'s real
C0-C3 machinery. Rejected at the gate — with more pairings already planned,
investing in the real, generic evidence model now was preferred over a
pairing-scoped type that would need redesigning at the second pairing
anyway.

This amendment authorizes no code — the evidence-computation work remains
`implementation_handoff`, tracked in
`.analysis/pending/cli_refactor/20260916-conformance-wiring-and-adr0029-t2-decisions/`.

**Implemented 2026-09-16, outside Strategist (direct user authorization).**
`internal/install/embedded_weapon_conformance.go` and
`internal/check/check_readiness.go`'s `evaluateRankedConformance` — see
that mission's `tasks.md` task 1.1 for the full evidence summary. The
generic-over-`(role, provider)` constraint above is satisfied: no function
added by this implementation contains `brainstorming`- or `ranger`-specific
logic.

## References

- `docs/adr/0041-cli-enforcement-sequencing-and-role-invocation-plan-naming.md`
  §"Ranked Class Resolution" / §"Pipeline scope" — naming and scope this ADR
  builds on.
- `docs/adr/0042-ranked-custom-binding-persistence.md` — the persistence
  decision (Ranked's runtime record is optional/non-authoritative) DEC-005
  extends by fixing *when* that optional record is generated.
- `.analysis/pending/cli_refactor/v2/02-ranked-role-binding.md` — the
  pending draft describing the Ranked pipeline's original, fuller
  build-time certification procedure and Wizard mockup; this pilot scopes a
  subset of it.
- `.analysis/pending/cli_refactor/20260916-ranked-pipeline-implementation-plan/`
  — the mission that produced this decision.
- `internal/domain/role_invocation_plan.go`,
  `internal/rolevalidation/role_weapon.go` — the existing fail-closed
  `mode: ranked` rejection points this ADR's tasks will replace with real
  resolution/validation.
