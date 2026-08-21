# ADR-0030 — World-Class Strategist Plugin Architecture

**Status:** Accepted  
**Date:** 2026-08-20  
**Mission:** `20260820-world-class-skill-plugin-refinement`  
**Extends:** ADR-0029

## Context

ADR-0029 established the correct external-skill lifecycle baseline: Strategist owns adapters rather than blindly vendoring upstream prompts; upstream and adapter revisions are independent; built-in metadata should come from one catalog; readiness must not be confused with live invocability; onboarding must complete or fail early; and native roles remain the resilient defaults.

That decision is necessary but not sufficient for a mature plugin platform. Its provider descriptor still groups information governed by different authorities: publisher package identity, Strategist compatibility, workspace installation state, operator binding, and consumer trust policy. The current implementation is even narrower: capability-mirror `skill.yaml` files, provider IDs in `active.slots`, duplicated/hardcoded registries, and mostly `risk_score`-based static validation.

A production-grade plugin model must additionally provide:

- a stable, versioned host API and explicit compatibility negotiation;
- immutable package identity and deterministic dependency locking;
- least-privilege permission grants without overstating host enforcement;
- trust roots, signatures/attestations, freshness, and revocation policy;
- runtime-neutral connector boundaries;
- staged, transactional activation with crash recovery and last-known-good rollback;
- evidence-backed conformance and readiness states;
- migration from current capability mirrors without breaking the native pipeline.

The design draws on established patterns: namespaced identity, host-engine compatibility and declared dependencies from extension platforms; digest-addressed artifacts and typed descriptors from OCI; separated trust/update roles from TUF; and artifact-bound provenance verified against consumer expectations from SLSA. These patterns inform the local architecture but do not require building a remote registry in the first release.

## Decision

Strategist will evolve its external-skill mechanism into a **local-first, content-addressed, adapter-based plugin platform**. The platform separates immutable published facts from workspace state and operator policy. External plugins extend the system; native Ranger, Archivist, and Sniper remain independently usable and recoverable.

### 1. Split plugin state into five resources

#### PluginPackage

Immutable publisher-facing identity:

- namespaced ID (`publisher/name`), semantic version, and immutable digest;
- artifact type, source URI, size, license, and creation metadata;
- upstream repository, relative skill path, revision, and content digests;
- signature and attestation references;
- manifest-schema version.

Versions and tags are human-facing resolution inputs. Locks, installations, grants, certifications, and active bindings reference immutable digests.

#### AdapterContract

Strategist-owned compatibility layer:

- compatible Strategist Plugin API range;
- supported roles, slots, and entrypoints;
- versioned input, output, and handoff contracts;
- declared capabilities and requested permissions;
- upstream package constraint and adapter revision;
- conformance profile, known limitations, and migration compatibility.

Upstream SemVer does not determine Strategist compatibility by itself.

#### InstalledInstance

Workspace-local materialization record:

- resolved package and adapter digests;
- connector ID and runtime locator;
- exact dependency-lock digest;
- trust-verification evidence and policy revision;
- installation/lifecycle state, health and probe results;
- local configuration reference and last-known-good status.

#### SlotBinding

Operator-owned activation choice:

- slot and installed-instance reference;
- configuration and permission-grant references;
- fallback policy and compatible native fallback;
- binding generation for compare-and-swap updates;
- enabled, disabled, or quarantined state.

An active slot never contains an unresolved provider string.

#### TrustPolicy

Consumer-owned verification expectations:

- trusted publisher identities, keys, or issuers;
- required signature and attestation types;
- allowed sources and connectors;
- freshness, expiry, revocation, and license policy;
- minimum conformance level and permission ceilings;
- explicit, scoped, expiring development exceptions.

No single manifest may become authoritative for all five resources.

### 2. Version the host contract independently

The platform tracks independent versions for:

- manifest schema;
- Strategist Plugin API;
- adapter revision;
- upstream package/skill;
- RuntimeConnector API.

Compatibility evaluation returns structured reasons across every dimension rather than one Boolean. Semantic ranges may select candidates, but `plugin.lock.yaml` pins the complete resolved graph by digest.

The Strategist Plugin API is declarative and message-oriented, not an in-process Go plugin ABI. Entrypoints such as discovery, refinement, or documentation materialization bind to versioned request/result schemas and role contracts. Invocation carries a bounded context envelope containing mission identity, approved scope, language, artifact references, permission grant, and gate state.

### 3. Introduce a RuntimeConnector SPI

Core lifecycle and policy remain independent of any one agent runtime. Connectors expose typed operations for:

- connector capabilities and enforceability;
- resolving an installed or local skill without activation;
- non-mutating visibility/compatibility probes;
- invocation through a structured envelope;
- safe removal of connector-owned materialization;
- observed compliance evidence where available.

Unsupported capabilities remain explicitly unsupported. A connector that cannot enforce a permission must never report that permission as enforced.

### 4. Separate capabilities, permissions, and evidence

The security model distinguishes:

- declared capabilities — what the adapter says it can do;
- requested permissions — files, network, subprocess, apps, secrets, and mutation scopes it asks for;
- granted permissions — the approved subset, bound to package and adapter digests;
- enforceable permissions — the subset the active connector/host can technically constrain;
- observed behavior — post-invocation evidence where available.

`risk_score` may remain a migration compatibility label but is not the permission authority. Any upgrade that expands requested permissions requires a new grant before activation.

### 5. Resolve dependencies deterministically

Plugin dependencies use namespaced IDs, compatibility ranges, optionality, and documented purpose. The resolver:

- distinguishes required dependencies from optional extension packs;
- rejects unsupported cycles and explains incompatible constraint graphs;
- selects one deterministic graph using documented precedence;
- pins every package, adapter, connector, and relevant policy input by digest;
- supports offline replay from materialized content;
- never silently upgrades transitive dependencies during activation.

The catalog is a discovery view; the lockfile is the reproducibility authority.

### 6. Verify trust, not only integrity

A digest proves content integrity relative to an expected value but not publisher identity, freshness, or expected build origin. Before staging, the platform:

1. validates schemas, paths, sizes, and extension namespaces as untrusted input;
2. resolves immutable package and adapter digests;
3. evaluates source and publisher against TrustPolicy;
4. verifies required signatures and attestations;
5. compares provenance source, builder, build type, and external parameters with expectations;
6. checks expiry, freshness, revocation, deprecation, and license policy;
7. verifies the dependency graph recursively;
8. records verification evidence and the exact policy revision.

Development exceptions are visible, auditable, scoped, and time-limited.

### 7. Make installation and activation transactional

Lifecycle states are explicit:

```text
discovered -> resolved -> verified -> staged -> probed -> active
                 |          |          |        |
                 +----------+----------+--------+--> failed/quarantined
```

Upgrade stages a candidate alongside the active instance. Only a successful probe may perform a compare-and-swap binding change. The previous instance remains last-known-good until activation is confirmed. Probe, activation, or health failure restores the last-known-good binding automatically.

Every transition is idempotent and journaled. Startup recovery completes or rolls back incomplete transactions. Concurrent operations use binding generations. Uninstall is blocked while an active binding or required dependent references the instance.

### 8. Represent readiness and conformance as evidence

Readiness is a state vector, not a single `ready` flag:

- descriptor valid;
- source resolved;
- trust verified or explicit development exception;
- dependencies locked and materialized;
- host API compatible;
- connector visible;
- entrypoint probed;
- permission grant complete;
- enforcement coverage reported;
- active binding healthy.

Conformance levels are:

- **C0 Descriptor:** schema-valid only;
- **C1 Contract:** role, handoff, and entrypoint fixtures pass;
- **C2 Runtime:** connector probe and controlled invocation fixtures pass;
- **C3 Trusted:** trust-policy verification and operational rollback tests pass.

Certification evidence binds to exact package, adapter, host API, connector, test-suite, and policy digests. It becomes stale when any bound input changes.

### 9. Keep v1 local and embedded

The first implementation supports embedded and local-path packages, deterministic locks, and runtime connectors. It adopts OCI-compatible concepts — typed descriptors, immutable digests, and separable attestations — without requiring an OCI registry.

Remote distribution requires a separate ADR after the local Plugin API, dependency lock, permission model, transaction lifecycle, and conformance system have production evidence.

## Migration

Migration preserves current behavior until parity is demonstrated:

1. introduce read-only domain types and schemas;
2. represent current Brainstorming and OpenSpec mirrors as synthetic PluginPackage and AdapterContract records;
3. generate existing `.strategist/skills/<id>/skill.yaml` files as compatibility views;
4. derive InstalledInstance and SlotBinding records from existing `active.slots` without changing routing;
5. add dual-read diagnostics with legacy state authoritative initially;
6. switch resolution behind a feature flag after golden/parity tests pass;
7. preserve native fallback throughout the cutover;
8. stop writing legacy registries only after a documented deprecation period while retaining read migration.

## Delivery Sequence

Implementation proceeds in deployable vertical slices:

1. vocabulary, resource schemas, version rules, and bounded loaders;
2. canonical built-in catalog and deterministic legacy-view generation;
3. embedded/local resolver and digest-pinned lock;
4. RuntimeConnector SPI and truthful readiness vector;
5. digest-bound grants and trust verification;
6. journaled stage/probe/activate/rollback lifecycle;
7. wizard plan/preview/apply integration and legacy migration;
8. conformance kit, certification, telemetry, quarantine, and compatibility matrix;
9. remote transport evaluation only after local exit criteria pass.

Each slice must preserve an operational native-only Strategist installation.

## Consequences

### Positive

- Publisher metadata, local installation state, operator choices, and consumer trust no longer conflict in one schema.
- Upgrades become deterministic, permission-aware, auditable, and recoverable.
- Runtime differences are isolated behind connectors without contaminating core policy.
- Diagnostics state exactly what is known, verified, enforceable, or unavailable.
- Content-addressed locks and certification evidence make rollback and reproduction reliable.
- Local-first delivery avoids premature marketplace infrastructure while preserving future transport portability.

### Negative

- The model introduces more resources, schemas, states, and tests than the current capability mirrors.
- Connector behavior and enforcement guarantees will differ across agent runtimes.
- Trust/signature support requires policy and operational key-management decisions.
- Transaction journals, dependency resolution, migrations, and compatibility matrices increase maintenance cost.
- Conformance cannot completely prove prompt behavior; it supplies bounded evidence rather than certainty.

### Risks

- A broad initial implementation could become a platform rewrite. Vertical slices and feature-flagged migration are mandatory.
- Claiming unenforceable permissions as sandboxed would create false assurance; UI and telemetry must preserve the distinction.
- A mutable catalog or tag used as activation identity would undermine reproducibility; bindings and locks must use digests.
- Crash recovery that is not fault-tested at every transition can still strand partial state.
- Building remote distribution before local contract stability would freeze immature interfaces.

## Rejected Alternatives

- **Keep extending `ProviderManifest`.** Rejected because it would combine publisher truth, Strategist contracts, local health, grants, and bindings under one authority.
- **Use the catalog as installation inventory.** Rejected because catalog metadata cannot authoritatively represent workspace health or operator decisions.
- **Use SemVer ranges without a lockfile.** Rejected because dependency resolution and rollback would not be reproducible.
- **Grant permissions by provider ID.** Rejected because mutable names and upgraded content can request different behavior.
- **Activate before probing.** Rejected because a partial or invisible provider could replace the operational baseline.
- **Load Go plugins in-process.** Rejected because it couples ABI, process trust, portability, and failure isolation unnecessarily.
- **Build a remote OCI registry in phase one.** Rejected because distribution is not the current bottleneck; the Plugin API, lock, policy, and lifecycle must stabilize first.

## Validation Requirements

- Schema fuzzing and explicit limits for untrusted manifests.
- Golden generation and legacy parity for current capability mirrors.
- Property tests for deterministic resolution, cycles, and conflict explanations.
- Offline lock replay without silent transitive upgrades.
- Connector contract tests, including typed unsupported capabilities.
- Permission escalation, re-consent, and unenforceable-permission presentation tests.
- Signature, provenance, expiry, revocation, license, and development-exception fixtures.
- State-machine, concurrency, compare-and-swap, and crash-recovery tests at every transition.
- End-to-end install, failed probe, activation, upgrade, rollback, disable, quarantine, and uninstall tests.
- Positive and malicious/non-compliant adapter conformance fixtures.
- Compatibility matrix across host API, adapter, connector, and manifest-schema versions.
- Proof that native roles remain usable after every external-plugin failure mode.

## Scope Boundary

This ADR records the target architecture and phased migration. It does not authorize or implement schemas, connectors, resolver, lockfile, trust system, permission grants, transaction engine, wizard changes, tests, remote distribution, or source/config mutations. Those remain separately authorized implementation work.
