# ADR-0032 — External Skill CLI Embedding and Treasure Chest Ownership

**Status:** Accepted  
**Date:** 2026-09-13  
**Mission:** `strategist-treasure-chest-external-skill`

## Context

Treasure Chest is currently a Strategist core capability. Its domain, CLI
registration, runtime configuration, embedded defaults, compiled indexes,
curation lifecycle, tests, and documentation live in this repository. The
desired product boundary is different: Treasure Chest will have its own
repository and versioning, while Strategist will consume selected published
skill artifacts and continue exposing the familiar command surface:

```text
strategist treasure-chest ...
```

The release must know which skills it contains before installation. A
pre-build process therefore needs to resolve and verify immutable skill
artifacts, record the exact versions and digests in committed build state, and
generate the inputs that are embedded into the Strategist binary. This must
coexist with the existing wizard/plugin path, where users can select skills
from their own environment or runtime.

The current plugin lifecycle already distinguishes package, adapter, installed
instance, binding, trust policy, grant, lock, transaction, and runtime
connector. It does not yet define a general CLI-extension contract or a
build-time embedded-skill registry.

## Decision

Adopt an adapter-first, message-oriented CLI Extension API and transfer
Treasure Chest ownership to an independently versioned external skill.

### External package and adapter

An external CLI skill publishes an immutable package manifest and a Strategist
adapter. The adapter declares the package identity and digest, upstream and
adapter versions, supported Strategist/connector APIs, command namespace and
subcommands, entrypoints, input/output envelopes, permissions, source class,
dependencies, license, provenance, conformance, and known limitations.

The host must not depend on a portable in-process Go plugin ABI or load
arbitrary code without a connector boundary. The extension contract is
declarative and message-oriented.

### Embedded release skills

The project maintains a committed embedded-skill lock. A pre-build operation,
recommended as `strategist plugin prepare-embedded` and exposed through a build
target such as `make embed-skills`, resolves the locked artifacts, verifies
digest/license/API/provenance, stages or materializes generated embedding
inputs, and produces the embedded catalog and command registration data.

Normal release builds do not refresh mutable tags or silently select a newer
skill. Lock, artifact, provenance, catalog, and generated-output drift fail the
pre-build or release gate. Updating an embedded skill is an explicit reviewed
lock change.

### CLI ownership and compatibility namespace

`strategist treasure-chest ...` remains the compatibility namespace, but its
implementation is supplied by the embedded Treasure Chest skill through the
CLI extension registry. Strategist core owns command routing, reserved
namespaces, policy and permission envelopes, output/exit normalization,
telemetry boundaries, and readiness diagnostics. The external skill owns
Treasure Chest domain behavior, storage, indexing, curation, and migration.

### Embedded skills versus user plugins

Embedded skills are project/release-owned: they are selected in the repository
lock, verified before build, and present in the released catalog. User plugins
remain workspace/operator-owned: they are selected through the wizard/plugin
lifecycle and represented by local inventory, grants, bindings, probes, and
connectors. User plugins are not silently added to a release catalog.

User skills may expose CLI commands only when the runtime connector and CLI
Extension API explicitly support that capability. Slot-only plugins remain
valid and continue to use native fallback when external invocation is
unavailable.

### Staged Treasure Chest removal

Core Treasure Chest code and defaults are removed only after:

1. the external package, adapter, and CLI contract exist;
2. embedded ingestion and command routing work;
3. command parity and migration tests pass;
4. existing workspace data has an explicit, idempotent, provenance-preserving
   import path;
5. the breaking-release/deprecation boundary is announced.

Core removal never deletes legacy chest data. If the external skill is absent,
legacy data remains untouched and the user receives an actionable migration
state.

## Readiness boundary

The following states remain distinct:

```text
package_valid
artifact_digest_verified
license_policy_passed
adapter_api_compatible
embedded_in_build
runtime_catalog_visible
command_registered
permissions_granted
entrypoint_probed
invocation_supported
healthy
```

Embedding or registering a command does not prove live invocation or health.
Unsupported connector capabilities must be reported explicitly.

## Alternatives considered

### Keep Treasure Chest in core and only mirror an external skill

Rejected. This leaves two domain owners and makes the published skill an
optional duplicate rather than the feature's source of truth.

### Copy external source directly into Strategist during build

Rejected as the default. Blind copying obscures provenance, license, API
compatibility, and reproducibility. A pinned artifact and adapter boundary are
required; intentional vendoring can be a later package mode with the same
metadata and verification.

### Load arbitrary Go plugins at runtime

Rejected. The ABI is not a portable or safe contract for the supported release
targets. Use a versioned host message contract and connector-supported artifact
types.

### Make every user plugin a CLI extension immediately

Rejected. Existing user plugins are primarily slot bindings, and runtime
connectors do not universally expose command invocation. CLI capability is
opt-in and evidence-based.

### Delete legacy Treasure Chest data during core removal

Rejected. Data migration belongs to the external skill and must be explicit,
retryable, and provenance-preserving.

## Consequences

### Positive

- Releases have a reproducible, auditable inventory of embedded skills.
- External skill repositories can evolve independently from Strategist core.
- The familiar CLI command remains stable during ownership transfer.
- Existing package/adapter/lock/grant/connector lifecycle concepts are reused.
- User plugins remain isolated from project release contents.
- Missing or unsupported skills do not invalidate native Strategist behavior.

### Negative

- A CLI Extension API, artifact lock, pre-build ingestion, and parity suite must
  be maintained.
- Core removal requires a migration period and a breaking-release boundary.
- Some runtimes may support slot plugins but not CLI extensions.
- Live invocation may remain unverified when the host exposes no connector API.

## Implementation boundary

This ADR records the accepted architecture. It does not implement the external
repository, CLI host, build command, generated artifacts, data migration, core
removal, tests, or release changes. Those remain separate implementation work
and must begin by freezing the package and CLI Extension API.
