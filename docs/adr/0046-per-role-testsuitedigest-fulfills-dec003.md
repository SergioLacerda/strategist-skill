# ADR-0046 — Role-Specific Ranked Conformance Evidence

**Status:** Accepted
**Date:** 2026-09-16
**Related:** ADR-0045, DEC-003

## Context

ADR-0045 accepted a shared `TestSuiteDigest` as an interim for the
Archivist/`openspec-propose` Ranked pairing. Its follow-up assumed that an
Archivist downstream handoff policy did not yet exist. The current contract
already provides that policy through `handoff.DefaultPolicy()` for the
Archivist-to-Sniper transition.

The remaining gap is evidence specificity: the existing conformance file is
Ranger-focused, so a shared digest does not prove Archivist-specific challenge
coverage.

## Decision

Fulfill ADR-0045 DEC-003 with two changes:

1. Add an Archivist-specific exhaustive conformance test for every required
   challenge type in `handoff.DefaultPolicy()`.
2. Parameterize `testSuiteDigest(role)` so Ranger and Archivist certification
   records pin separate role-relevant test files.

The existing `DefaultPolicy()` remains the canonical handoff policy. No new
policy, parallel state machine, or edit to ADR-0045 is introduced.

## Consequences

- Archivist Ranked certification gains role-specific conformance evidence.
- Changes to Ranger and Archivist conformance tests invalidate only the relevant
  certification evidence.
- Catalog regeneration and parity validation become part of the certification
  workflow.
- The digest remains a freshness marker; it does not replace live provider
  invocation or runtime health evidence.

## Scope boundary

This ADR does not implement the remaining runtime lifecycle work: clean install,
reinstall, rollback, provider cwd delivery, or fail-closed runtime fixtures.
Those remain implementation handoff items in the unified Ranked Runtime
Conformance demand.

## References

- `docs/adr/0045-ranked-testsuitedigest-shared-pin.md`
- `.analysis/refined/20260916-unificar-ranked-runtime-conformance/`
- `internal/handoff/policy.go`
- `internal/install/embedded_skill_conformance.go`

