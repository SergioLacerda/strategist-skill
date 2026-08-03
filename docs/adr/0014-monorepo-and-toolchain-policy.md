# ADR-0014 - Monorepo and toolchain policy

**Status:** Accepted
**Date:** 2026-08-03
**Context:** `20260803-cicd-policy-eval-residuals`

---

## Context

Strategist ships a Go CLI and a small Astro landing/documentation surface from
one repository. That shape is intentional, but it was only implied by the file
tree and CI jobs:

- Go runtime code lives under `cmd/` and `internal/`.
- Embedded Strategist defaults live under `internal/embed/defaults/`.
- Documentation lives under `docs/`.
- The landing site lives under `web/landing/`.

The same repository also carries strict toolchain declarations: `go.mod` sets
`go 1.26.4` and `toolchain go1.26.5`, while the web surface uses Node 22 in CI
and declares `engines.node >= 22.12.0`.

## Decision

Keep the Go CLI, embedded defaults, docs, and landing site in one repository
while the landing site remains a publication surface for the same Strategist
product and release train.

CI ownership is split by surface:

- Go validation is owned by Make targets such as `ci-lint`, `ci-test`,
  `release-verify`, `cover-gate`, `quality-budget-gate`, and
  `release-reproducible-check`.
- Documentation governance is owned by `docs-governance-gate` and the spec tests
  that assert normative wording.
- Web validation is owned by `ci-web`, `lint-web`, `test-web`, `cover-web`, and
  landing build jobs.
- Release publication is owned by the release workflow plus the Make release
  gates.

Toolchain policy:

- Go version authority is `go.mod`. Workflows should use `go-version-file:
  "go.mod"` instead of restating the Go version.
- `go 1.26.4` is the module/language target.
- `toolchain go1.26.5` is the exact patch toolchain for CI-compatible local
  verification.
- Node 22 is the supported major version for `web/landing/`; the package-level
  floor is `>=22.12.0`.
- Toolchain pins may be relaxed or bumped only when CI, local verification, and
  contributor documentation are updated together.

## Reconsider When

- The landing site gains an independent release cadence or ownership model.
- Web dependencies require a Node major version that no longer matches CI.
- The Go CLI no longer needs to package Strategist defaults from the same tree.
- Release verification can prove a wider Go toolchain range without changing
  emitted binaries or generated defaults.

## Consequences

- Contributors have one source of truth for Go versions and a clear Node policy
  for the web surface.
- CI changes that restate or relax toolchain versions must update this ADR and
  contributor docs.
- The monorepo boundary stays explicit without adding new package ownership
  machinery.
