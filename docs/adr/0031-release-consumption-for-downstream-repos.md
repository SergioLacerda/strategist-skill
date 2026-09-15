# ADR-0031 — Release Consumption for Downstream/Internal Repos

**Status:** Accepted
**Date:** 2026-09-05
**Context:** `20260905-versioning-proposal-review`

## Context

An external proposal (`.analysis/pending/versionamento.txt`) asked how a
downstream/internal repository should consume this project's releases —
importing only the skill and compiled CLI, from a published tag, instead of
mirroring the full source repository (which would also carry the landing
page and development-only tooling the internal org has no use for).

Reviewing that proposal against this repo's actual, already-implemented
release architecture (`.goreleaser.yaml`, `.github/workflows/release.yml`,
`SECURITY.md`) surfaced two things: the proposal's core direction —
distribute compiled binaries from a tagged release, not a full clone — is
already this repo's practice, and goes further than the proposal assumed
(cosign signing, an SBOM, and a GitHub build-provenance attestation are
already published per release, none of which the proposal accounted for).
It also surfaced concrete corrections: the proposal's assumed file layout
(a `skill/` folder, a `pkg/` directory, a `VERSION` file, a combined zip
archive) does not match this repo, and one of its two suggested strategies
— sparse-checkout a subset of directories and build the CLI from source in
the internal repo — is not merely impractical here but technically
incorrect: `cmd/strategist` imports `internal/*` packages under this
module's own import path, and Go's compiler refuses to let code outside a
module's own tree import that module's `internal/` packages. A sparse
checkout under a different repository cannot compile this CLI from source
at all, regardless of which directories it includes.

## Decision

A downstream/internal repository consuming this project's releases should:

1. **Consume tagged releases' published binaries directly.** Every
   `v*.*.*` tag publishes, per platform (`linux`/`darwin`/`windows` ×
   `amd64`/`arm64`): the binary itself, a shared `SHA256SUMS` file, a
   per-binary cosign `.bundle`, and a CycloneDX SBOM. There is no `skill/`
   folder, no `pkg/` directory, no `VERSION` file, and no combined zip
   archive to look for — each is a separate release asset.

2. **Prefer cryptographic verification over checksum-only validation.**
   With network access to GitHub's API and Sigstore's public transparency
   log (the assumed case for this decision), verify with, in order of
   preference:
   ```bash
   gh attestation verify strategist-linux-amd64 --owner SergioLacerda
   ```
   ```bash
   cosign verify-blob strategist-linux-amd64 \
     --bundle strategist-linux-amd64.bundle \
     --certificate-identity-regexp "https://github.com/SergioLacerda/strategist-skill/.*" \
     --certificate-oidc-issuer "https://token.actions.githubusercontent.com"
   ```
   with `sha256sum --check SHA256SUMS` as the baseline every path should
   also satisfy. Both stronger commands are documented in `SECURITY.md`;
   an internal pipeline should not fall back to checksum-only validation
   as its primary path when it has the network access to do better.

3. **Do not sparse-checkout and build the CLI from source as an
   alternative strategy.** This does not compile: `cmd/strategist` imports
   `internal/*` packages under module path
   `github.com/SergioLacerda/strategist-skill`; Go's `internal/` import
   rule restricts those packages to code inside that same module tree. A
   consumer would need the entire module under the same import path, not a
   sparse subset under its own — the previously proposed fallback is a
   hard technical constraint, not a preference to reconsider later.

4. **Keep an internal provenance manifest**, extending the proposal's own
   `UPSTREAM.yaml` idea with a `verification` field naming what was
   actually checked:
   ```yaml
   upstream:
     repository: SergioLacerda/strategist-skill
     version: v1.4.0
     commit: 7b9f...
     synchronized_at: 2026-09-05
     verification: attestation   # one of: sha256 | cosign | attestation
   ```
   Recording only `sha256` when that was the actual check performed keeps
   the manifest honest about the strength of the guarantee it recorded,
   rather than implying uniform assurance across every synchronized
   version.

5. **Do not distribute the skill's source separately from the binary.**
   The skill's content (`SKILL.md`, contracts, personas, templates, roles)
   is embedded into the binary via `go:embed` and is materialized as
   `.strategist/` only by running `strategist install` against the binary
   itself. A verified binary is a complete, self-contained distribution
   unit — there is nothing to separately fetch, checksum, or vendor
   alongside it.

## Consequences

- No change to this repository's own release tooling — the existing
  pipeline already exceeds what this decision recommends consuming.
- A downstream/internal repository following this decision inherits the
  same tagged-commit CI gate (`make release-verify`, lint, web gates) this
  repo's own release workflow already enforces, since it only ever
  consumes tagged releases, never arbitrary commits.
- This decision does not cover an air-gapped consumption scenario (no
  network access to Sigstore/GitHub's API); that would need its own,
  separate decision about checksum-only validation as a primary path
  rather than a fallback.
