# ADR-0051 — Remote Provider Acquisition and Trust Policy

**Status:** Proposed
**Date:** 2026-09-21
**Related:** ADR-0029, ADR-0030, ADR-0028

## Context

`strategist provider validate|add` accepts only a local, already-materialized
provider directory. Remote git, URL and registry sources are rejected with
`remote_source_deferred` (`internal/provider/source.go`). That boundary was a
deliberate v1 decision: a network fetch must not turn onboarding into an
unreviewed installer.

The trust building blocks already exist. The trust-policy schema declares
`trusted_publishers`, `trusted_sources`, `allowed_licenses`,
`required_signatures`, `required_attestations`, `freshness_days`,
`revocation_sources`, `revoked_digests`, `deprecations` and
`development_exceptions`. `internal/plugins/trust` verifies a package against
that policy and reports stable reasons such as `publisher_not_trusted`,
`source_not_trusted`, `missing_required_signature`,
`missing_required_attestation`, `digest_revoked`, `digest_deprecated` and
`freshness_expired`. What is missing is the acquisition step that produces the
evidence the verifier consumes, and the rules for when that evidence is
sufficient.

## Decision (proposed)

1. **Acquire, then validate, then add.** A remote source is first fetched into
   a quarantine directory outside the governed runtime. The existing local
   `validate` and `add` then run unchanged against the quarantined copy. No
   fetch code writes to `.strategist/`, `active.yaml` or `plugins.lock`.
2. **Digest pinning is mandatory.** A remote reference must carry an expected
   package digest (or an immutable commit id for git). A floating branch, tag
   or "latest" reference is rejected. The fetched bytes are hashed and compared
   before any parsing of provider content.
3. **Policy decides trust, never the source.** Acquisition collects evidence
   (source URI, signatures, attestations, builder identity); `trust.Verify`
   decides. An empty trust policy trusts nothing remote. Trusted publishers and
   sources are operator-owned and are never distributed by a provider or by the
   catalog.
4. **Signatures and attestations are required by policy, not by default.**
   The operator states which signature and attestation types are required.
   A provider that lacks a required one fails with the existing
   `missing_required_signature` / `missing_required_attestation` reasons.
5. **Revocation and freshness fail closed.** Revocation sources are consulted
   at add time and at `check` time. When a revocation source is unreachable the
   result is `revocation_unknown`, which blocks activation unless the policy
   explicitly allows offline operation. A package older than `freshness_days`
   is refused with `freshness_expired`.
6. **Development exceptions stay explicit and expiring.** An untrusted remote
   source can be admitted only through a `development_exceptions` entry bound
   to one digest and one expiry, and the result is recorded as
   `DevelopmentException` evidence on the installed instance.
7. **Native role authority is unchanged.** Remote acquisition cannot activate an
   external discovery provider; the `native_role_authority` rejection applies
   regardless of source.
8. **Fetching is opt-in per source class.** git, URL and registry are separate
   source classes, each disabled until the operator enables it in policy. Network
   access is never implied by the presence of a catalog entry.

## Consequences

- The local-only v1 behavior and its reason codes do not change until this ADR
  is accepted and implemented.
- New reason codes (`revocation_unknown`, `remote_digest_unpinned`,
  `remote_digest_mismatch`, `source_class_disabled`) must be cataloged in
  `machine/errors.yaml` before the first fetch path ships.
- A registry protocol, key distribution and transparency-log integration are
  out of scope here and need their own decision once a concrete registry exists.

## Open questions

- Which signature scheme (for example Sigstore keyless versus operator-held
  keys) is the first supported one.
- Where revocation lists are hosted and how they are themselves authenticated.
- Whether quarantine lives under the OS temp directory or a Strategist-owned
  scratch root.

## Non-goals

No fetch, network client, registry, or signature-verification service is
implemented by this ADR.
