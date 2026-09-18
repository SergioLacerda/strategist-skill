# Role→Weapon Mutation Resistance

This repository distinguishes two forms of authorization evidence:

- **Static mutation resistance**: focused tests intentionally alter binding
  inputs or decision outcomes and verify that critical authorization predicates
  still reject invalid Role→Weapon combinations.
- **Live provider invocation**: runtime checks prove that the selected provider
  is installed, callable, and running from its declared runtime root.

Mutation tests do not prove live provider availability. They cover the local
authorization boundary only.

The planned critical matrix covers provider and slot identity, binding
cardinality, mode, Ranked certification and catalog affinity, Custom
`plugins.lock` persistence, and wizard propagation of provider and mode. Ranked
tests must use the embedded catalog as their authority; Custom tests must use
the persisted lock and manifest compatibility. A test that crosses these
authorities is invalid evidence and should be treated as a failed mutation
case.

Wizard coverage must include discovery and refinement independently, including
role-compatible filtering, Ranked/Custom parsing, defaults, and persisted
configuration. Execution coverage is limited to decisions already exposed by
the wizard contract.

The implementation handoff is expected to add a bounded local mutation command
and integrate its critical-mutant threshold with existing quality gates. A
surviving critical mutant or malformed mutation result must fail that command.
Until a separate decision is made, adoption of an external Go mutation engine
and broad mutation-score reporting remains side quest SQ-001.

For runtime confidence, pair the mutation command with the existing focused
binding/wizard tests and the appropriate `strategist check`/provider health
validation. Do not interpret a passing static mutation result as proof that an
external weapon can be invoked.
