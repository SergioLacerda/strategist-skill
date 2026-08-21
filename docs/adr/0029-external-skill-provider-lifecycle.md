# ADR-0029 — External Skill Provider Lifecycle and Upgrade Process

**Status:** Accepted  
**Date:** 2026-08-20  
**Mission:** `20260820-external-skill-upgrade-architecture`

## Context

Strategist currently ships two entries under `internal/embed/defaults/skills/`: `brainstorming` and `openspec-explore`. These entries are Strategist-authored capability mirrors (`skill.yaml`), not vendored or executable copies of the upstream skills. They declare slot metadata such as `risk_score`, category, and role taxonomy, but do not include the upstream `SKILL.md` content.

This distinction is architecturally useful but is not yet represented by a complete provider lifecycle. Both mirrors use the generic version `1.0.0` and omit upstream repository identity, skill path, release or commit, content hash, license, dependencies, compatibility state, and adapter revision. Consequently, a maintainer cannot mechanically determine which upstream revision a mirror was evaluated against, whether a downloaded update is compatible, or how to reproduce and roll back an adapter upgrade.

The setup wizard also exposes incomplete pluggability. Its built-in choices and installable manifest paths are hardcoded, while risk metadata is duplicated between YAML and Go. A user can enter an arbitrary provider ID, but the installer only materializes manifests for recognized hardcoded defaults. The unresolved ID may then fail preflight because `.strategist/skills/<provider>/skill.yaml` was never created. Static preflight checks primarily validate manifest readability and `risk_score`; they do not prove dependency availability, runtime visibility, behavioral compatibility, or live invocability.

The upstream revisions inspected for this decision were:

- `obra/superpowers`, package `6.3.0`, commit `b36e0829c6d0140e93cfef2ca599b1b07d4a7797`, skill `skills/brainstorming/SKILL.md`, SHA-256 `74edf03ea6d24ef53db48677b93558d14a979bdf052ca3f57ecdca0c66791608`;
- `Fission-AI/OpenSpec`, package `1.10.0`, commit `1ebddd17f40dde15dfd28289e4493c3cf05ee9df`, skill `skills/openspec-explore/SKILL.md`, SHA-256 `7e0281fc4f094dcced0c5469e7cbceba2dce9f113f22a4e34efcdce4315edf81`.

These pins are evidence for this decision, not permanent defaults.

The upstream workflows are not directly interchangeable with Strategist roles. Brainstorming has its own interactive routing, approval gates, design-document location, and `writing-plans` transition, whereas Ranger requires an autonomous canonical discovery artifact. OpenSpec Explore is intentionally free-form and depends on the OpenSpec CLI, whereas Archivist requires a four-file refined package and a strict handoff. Copying either prompt into the embedded defaults would not by itself make it contract-compatible or runtime-invocable.

## Decision

Strategist will use an **adapter-first provider lifecycle**. Upstream skills and their installation remain separate from Strategist's embedded capability mirrors. Strategist will not blindly vendor upstream `SKILL.md` files as its upgrade mechanism.

### 1. Canonical provider catalog

A single provider catalog will become the authoring source for built-in and connectable provider metadata. Generated capability mirrors, wizard choices, known-provider risk data, attribution fixtures, and relevant conformance fixtures must derive from this catalog rather than independent hardcoded registries.

Each provider entry must distinguish:

- stable provider ID and display name;
- canonical role, compatible slots, `risk_score`, provider class, and lifecycle status;
- adapter schema version and adapter revision;
- upstream repository URL, relative skill path, release/tag or commit, content SHA-256, license, and attribution URL;
- runtime, CLI, plugin, package, or version dependencies;
- connection strategy and whether Strategist can materialize the capability mirror;
- conformance status, known incompatibilities, deprecation, and replacement information.

Generated runtime mirrors contain only the fields required at runtime plus enough provenance to diagnose drift. Generated files must identify their source and be checked for drift in CI.

### 2. Independent upstream and adapter versions

Upstream revision and Strategist adapter revision are independent values. An upstream release can require no adapter change, while a Strategist contract change can require a new adapter revision without an upstream release. Compatibility and rollback must therefore never depend on one overloaded `version` field.

### 3. Explicit readiness tiers

Provider readiness must be reported as distinct states:

1. **Manifest valid:** the Strategist descriptor is syntactically and structurally valid for the selected slot.
2. **Dependency ready:** declared CLI, package, plugin, or runtime requirements are satisfied where they can be checked.
3. **Runtime visible:** the active agent environment can identify the provider as installed or callable where the runtime exposes such a check.
4. **Contract compatible:** the adapter's conformance checks cover the required Strategist role, handoff, write scope, and approval behavior.

No static manifest check may be described as proof of live invocation. When the agent runtime exposes no provider-introspection API, the state must remain explicitly unverified until an actual invocation succeeds or fails.

### 4. Complete custom-provider onboarding

The setup wizard must not activate an unresolved provider string. Connecting an external provider must use an explicit source mode:

- **Already installed:** identify the runtime-visible provider where supported, validate or create its capability mirror, and report any unverified tier.
- **Local path:** inspect the selected skill metadata, fingerprint its content, validate dependencies and compatibility, and request activation only after the descriptor is complete.
- **Supported plugin or package reference:** use a dedicated installer or connector integration. Writing a capability mirror must never be presented as installing the upstream skill.

If onboarding cannot create or import a valid descriptor, the wizard must fail early with an actionable remediation and leave the active slot unchanged. A provider may be registered as `configured_unverified` when runtime visibility cannot be established, but it must not be reported as ready.

### 5. Native providers are the installation defaults

Fresh installations must default to native Ranger, Archivist, and Sniper behavior that is guaranteed by the Strategist runtime. External providers are opt-in after applicable readiness checks.

The wizard must not imply that discovery is provider-selectable while routing unconditionally resolves discovery to native Ranger. `openspec-explore` must not remain the default refinement provider unless a genuinely invocable, contract-compatible adapter has been installed. This extends the resilient baseline established by ADR-0027 and ADR-0028.

## External Skill Upgrade Procedure

Upgrading an external skill adapter is a controlled maintainer operation:

1. **Select the candidate.** Fetch or update the upstream checkout outside the embedded Strategist build. Record the repository, release/tag or commit, skill-relative path, and license.
2. **Verify provenance.** Confirm repository identity, compute the skill content hash, and compare it with the currently pinned catalog entry. Do not promote a mutable branch name as the only provenance.
3. **Review the upstream diff.** Classify changes to workflow, gates, required tools, output paths, downstream skill dependencies, write behavior, and runtime assumptions.
4. **Evaluate role compatibility.** Compare the candidate with the Strategist role and handoff contracts. Record incompatibilities explicitly; a successful manifest parse is insufficient.
5. **Update the adapter.** Change the upstream pin/hash and compatibility metadata. Increment `adapter_revision` whenever Strategist-owned translation or conformance behavior changes.
6. **Generate artifacts.** Regenerate capability mirrors, wizard metadata, risk registry output, attribution, and fixtures from the catalog. Manual edits to generated mirrors are not the upgrade source of truth.
7. **Validate.** Run catalog-schema validation, generation-idempotency and drift checks, adapter conformance tests, installer and upgrade tests, `strategist check`/compile tests, dependency diagnostics, and license/attribution checks. Normal CI should use offline pinned fixtures; network refresh is an explicit maintainer action.
8. **Promote atomically.** Accept the new pin and generated artifacts only after every required check passes. Preserve existing user-owned runtime configuration according to the current merge rules.
9. **Record the result.** Document incompatible upstream changes, migrations, readiness limitations, and replacement/deprecation decisions in the release change record.

### Rollback

Rollback restores the previous catalog pin, adapter revision, and generated artifacts as one unit. No upgrade step overwrites user-modified runtime configuration with `--force` implicitly. When a runtime provider descriptor needs transformation, use an explicit versioned migration rather than treating extraction overwrite as a migration system.

## Consequences

### Positive

- Provider upgrades become reproducible, reviewable, and reversible.
- Strategist contracts remain the authority instead of inheriting arbitrary upstream workflow changes.
- One catalog eliminates drift among wizard defaults, risk registries, attribution, and embedded mirrors.
- Users receive truthful diagnostics about what is configured, installed, compatible, and actually callable.
- Custom-provider onboarding either completes the capability boundary or fails before producing a broken active configuration.
- Native defaults keep fresh installations operational without optional external integrations.

### Negative

- Maintaining catalog schema, generators, conformance fixtures, and provenance checks adds engineering cost.
- Live invocability can remain unknown on runtimes that expose no provider-inspection interface.
- Some upstream updates will require manual behavioral review rather than automated copying.
- Adapter and upstream revisions introduce more lifecycle states than the current generic version field.

### Risks

- A catalog entry can still overstate behavioral compatibility unless conformance tests cover the real role and handoff boundaries.
- Automatically fetching upstream content during ordinary builds would harm reproducibility and supply-chain safety; refresh must remain explicit.
- Local-path providers may change after registration, so their fingerprints must be rechecked during readiness diagnostics.
- License and attribution requirements differ across upstream projects and must block promotion when unresolved.

## Rejected Alternatives

- **Vendor the current upstream `SKILL.md` files.** Rejected because copying prompts increases coupling and license/merge burden while leaving role-contract and runtime-invocation gaps unresolved.
- **Keep manual mirrors and add only a checklist.** Rejected because duplicated registries and absent provenance remain mechanically undetectable.
- **Accept any wizard provider ID and diagnose it later.** Rejected because this converts setup into a delayed and predictable preflight or invocation failure.
- **Use a single provider version.** Rejected because upstream and adapter changes have independent compatibility and rollback semantics.
- **Treat `risk_score` validation as readiness.** Rejected because permission metadata does not prove dependencies, runtime visibility, contract compliance, or invocability.

## Validation Requirements

- Validate catalog schema, unique IDs, source modes, immutable provenance, and license fields.
- Generate mirrors, wizard choices, risk metadata, attribution, and fixtures deterministically from the catalog.
- Fail CI when checked-in generated outputs drift from the catalog.
- Exercise native, already-installed, local-path, invalid, and `configured_unverified` wizard paths.
- Prove that an unresolved custom ID cannot alter an active slot.
- Test installation, upgrade, rollback, and preservation of user-modified runtime files.
- Prove that static manifest validity is never labeled live invocability.
- Maintain adapter conformance fixtures for required role, handoff, write-scope, and approval behavior.

## Scope Boundary

This ADR records the provider lifecycle and upgrade decision. It does not itself implement the provider catalog, generator, wizard changes, readiness diagnostics, tests, manifest migrations, or attribution correction. Those remain separate implementation handoffs requiring explicit code-mutation authorization outside the Strategist documentation mission.
