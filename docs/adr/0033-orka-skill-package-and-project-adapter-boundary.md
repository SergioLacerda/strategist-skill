# ADR-0033 — Portable ORKA Skill Package and Project Adapter Boundary

**Status:** Accepted
**Date:** 2026-09-13
**Mission:** `strategist-orka-skill-template-adr-task`

## Context

The organization's skill-packaging standard, ORKA, defines a portable folder
contract: `SKILL.md` is mandatory; `README.md`, `references/` (capped at
~200 lines per file), `scripts/`, `templates/`, and `assets/` are optional.
`SKILL.md` carries YAML frontmatter (`name`, `description`, `metadata:
{version, author}`) plus Markdown instructions. ORKA says nothing about how a
package is invoked, authorized, or governed once it enters a specific host
project — that is left to each consuming project.

Strategist already runs a source-to-runtime pipeline, documented in
`docs/architecture.md`: `internal/embed/defaults/` is the single authoring
source, embedded into the binary via `go:embed`; `strategist install`
extracts it into `.strategist/` as the installed runtime; `strategist
compile` derives `.strategist/.compiled/*.gz` from that runtime. Source,
installed runtime, and compiled artifacts are three distinct, ordered
states today, with no equivalent distinction yet defined for an externally
authored ORKA skill moving from its own repository into a Strategist
release or workspace.

A prior exploration, `docs/design/soft-profile-orka-mapping.md`, mapped
Strategist's *own* runtime onto the ORKA folder structure as a packaging
reference case, and was explicit that the mapping does not authorize
building the package it describes. That document answers "how would
Strategist's own content fit ORKA's folders" — it does not answer "what
must a project require of any ORKA skill, including a third-party one,
before invoking it."

`.providence/templates/skill/skill.yaml.tpl` shows what a project-specific
integration profile looks like in practice: alongside package-shaped fields
(`triggers`, `forbidden`, `fallback_to`) it carries `execution_path`,
`allowed_tools`, `cli_fallback`, `required_permissions`, `budget_policy`,
`escalation_policy`, `telemetry_policy`, and `validation_policy` — all
Providence-specific integration concerns, not generic ORKA package
concerns. Nothing today prevents a new skill from copying this template
wholesale and coupling a portable package to one provider's runtime.

Strategist's plugin lifecycle (ADR-0029, ADR-0030) already separates
package, adapter, installed-instance, binding, trust policy, lock, and
transaction authorities for *slot providers*. A new ORKA-to-Strategist
integration contract should reuse that vocabulary and those authorities
rather than invent a second, competing lifecycle model.

No current contract states which layer — the portable ORKA package, or the
Strategist project consuming it — owns invocation, permissions, lifecycle,
output destination, compatibility, and runtime loading. This ADR makes that
boundary explicit.

## Decision

Adopt a two-layer contract: the ORKA package remains the portable
authoring unit; the Strategist project adds an explicit, separate adapter
that owns everything about *running* the package inside this project.
Neither layer may own the other's concerns.

### Ownership table

| Layer | Owns | Must not own |
|---|---|---|
| ORKA package | purpose, entrypoint, triggers, capabilities, inputs, outputs, forbidden actions, local fallback, package metadata | project permissions, runtime paths, provider commands, project gates |
| Project adapter | capability identity, compatibility, invocation context, permissions, write scope, lifecycle/gates, output destination, observability, handoffs, runtime ownership | rewriting the skill's purpose or duplicating ORKA instructions |
| Runtime/package pipeline | source-to-package-to-generated-to-runtime synchronization and evidence | treating generated/runtime state as authoring authority |

The adapter is mandatory for a skill to integrate with the Strategist
project, but never mandatory for the ORKA package itself to remain valid
and portable outside Strategist. Provider-specific fields (Providence's
`execution_path`, `budget_policy`, and similar) are additive adapter
*profile* material — they may exist in a profile layered on top of the
adapter, but they can never redefine or replace the provider-neutral core
fields above.

### Two-way contract shape

Information flows in exactly two declared directions; nothing is implicit
on either side:

- **Skill → project**: purpose, capabilities, supported inputs/outputs,
  triggers, forbidden actions, package metadata, local fallback behavior.
- **Project → skill**: invocation context, permissions, write scope,
  lifecycle and gate policy, output destination, observability, handoff
  protocol, compatibility policy, runtime loading rules.

The adapter is the explicit translation surface between these two
directions. A skill package must never declare project-side fields
(permissions, gates, runtime paths) directly in its own `SKILL.md` or
package metadata; a project adapter must never redefine the skill's stated
purpose or duplicate its instructions verbatim.

### Conformance levels

Conformance is progressive and evidence-based. A lower level is never
proof of a higher one, and a future validator must report the level
actually reached plus its evidence — never an unqualified pass/fail:

```text
package-static     structure, frontmatter, required sections, references,
                    forbidden actions, and declared inputs/outputs are
                    structurally valid
project-contract    an adapter exists and its permissions, lifecycle,
                    outputs, fallback, ownership, and compatibility claims
                    validate against the project contract
live-invocation     an actual invocation proved resolution, authorization,
                    execution, output, fallback, and observability behavior
```

`package-static` evidence must never be reported as `project-contract`
evidence, and neither may be reported as `live-invocation` evidence. This
mirrors the same non-equivalence rule Strategist's plugin lifecycle already
applies to `PluginReadinessVector` (`internal/domain/plugin_readiness.go`):
a dimension left `unknown` is not silently treated as `ready`.

### Compatibility

Compatibility is a tuple, not a single package version:

- `package_version` — the skill's own upstream/authoring version.
- `adapter_contract_version` — the version of the adapter schema itself.
- `project_runtime_contract_version` — the Strategist project/runtime
  contract version the adapter was validated against.

A project may accept, degrade, or reject an integration based on any one
dimension of this tuple independently; a matching `package_version` alone
never implies the other two are compatible. The exact version syntax is
deferred to the adapter schema's own implementation mission (T2); this ADR
only requires that the three dimensions stay independently evaluable and
independently auditable.

### Authority and synchronization

Four states are distinct and ordered, mirroring the source/embedded/
runtime distinction `docs/architecture.md` already documents for
Strategist's own defaults:

| State | Authority | Verification |
|---|---|---|
| authoring source | skill/adapter source repository | source review and schema checks |
| published package | release artifact and metadata | package checksum/version |
| generated/embedded defaults | project build inputs | build manifest and stale detection |
| installed runtime | `.strategist/` instance | runtime check and compiled manifest |

Authoring source is always authoritative. Published package, generated/
embedded defaults, and installed runtime are derived states that must be
independently verifiable against their source — an installed runtime being
present is never itself proof that it matches current source. This is the
same anti-drift posture ADR-0025 already establishes for generated
documentation; this ADR extends it to skill packages and adapters instead
of introducing a separate drift policy.

## Alternatives considered

### Ship Providence's `skill.yaml.tpl` as the generic ORKA template

Rejected. Its `execution_path`, `budget_policy`, `escalation_policy`, and
`telemetry_policy` fields are Providence-specific integration semantics,
not portable ORKA package concerns. Using it as the generic template would
make every new skill implicitly Providence-coupled.

### `SKILL.md` alone, with no separate adapter

Rejected. ORKA's `SKILL.md` has no vocabulary for project permissions,
lifecycle/gates, output destination, or runtime loading — leaving those
implicit would make every project reinvent an undocumented, ad hoc
integration contract per skill.

### Copy the full project contract into every skill package

Rejected. Strategist already has typed handoffs, approval gates, and
source/runtime ownership rules (`.strategist/contracts/`). Duplicating them
into each skill package creates drift between the copies and the
authoritative contracts, and gives no single place to update the rule.

### Choose the concrete adapter manifest format in this ADR

Rejected as premature. The exact representation (frontmatter extension,
`skill.yaml`, a separate integration manifest, or an equivalent) depends on
schema and validator design that has not happened yet (T2). Fixing the
format here would couple an architecture decision to an implementation
detail before that detail is scoped.

## Consequences

### Positive

- New skills can be authored once, portably, and integrated into Strategist
  (or any other ORKA-consuming project) without rewriting their core
  purpose per project.
- Provider-specific integrations (Providence today, others later) become
  additive profiles instead of forks of the package format.
- Conformance claims become auditable and cannot silently overstate what
  was actually verified.
- Compatibility evaluation reuses the existing package/adapter/lock
  vocabulary from ADR-0029/ADR-0030 instead of a third competing model.
- Source-to-runtime anti-drift for skills follows the same posture
  ADR-0025 already established for generated documentation.

### Negative

- Every new skill now requires two authored artifacts (package + adapter)
  instead of one, adding authoring overhead for the simplest cases.
- A validator, schema, and synchronization tooling must still be built
  (T2–T5) before any of this is enforced rather than merely documented.
- Until a provider-neutral template and example exist (T3), authors have no
  concrete starting point and may default back to copying a provider
  profile.

## Implementation boundary

This ADR records the accepted architecture only. It does not implement the
adapter schema, the ORKA template/scaffold, the validator, the
synchronization/anti-drift tooling, the Providence profile, or the
publication/compatibility policy. Those are six independently scoped
implementation-handoff missions (T2–T7 in the source refinement's
`tasks.md`), each requiring its own selected scope and Approval Gate before
touching `cmd/`, `internal/`, `.strategist/`, or `.providence/`.
