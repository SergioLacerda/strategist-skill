# ADR-0014 - Monorepo and toolchain policy

**Status:** Accepted
**Date:** 2026-08-03
**Amended:** 2026-08-30 — extended scope to `web/design/` (see Context/Decision below)
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
- Design-system source (tokens, component specs, UI kits, and the design
  workspace's own guideline docs for the Strategist Console and landing
  surface) lives under `web/design/`. It is an editorial/authoring tree, not
  a build artifact of any CI pipeline — nothing under `internal/`, `cmd/`,
  or `web/landing/` currently consumes it programmatically at build time; it
  documents and stages the visual design that `web/landing/` and future
  console work implement by hand.

The same repository also carries strict toolchain declarations: `go.mod` sets
`go 1.26.4` and `toolchain go1.26.5`, while the web surface uses Node 22 in CI
and declares `engines.node >= 22.12.0`.

## Decision

Keep the Go CLI, embedded defaults, docs, and landing site in one repository
while the landing site remains a publication surface for the same Strategist
product and release train. The same reasoning extends to `web/design/`: keep
it in the same repository too, as a documentation/authoring surface, not an
independently released artifact.

`web/design/` specifically: it is kept alongside `web/landing/` because it
is the design-authoring input for that same surface (and for the future
Strategist Console), not because of an accidental/organic accumulation —
its content is reviewable design history and authored source, not derived
build output, so it does not belong in `.gitignore` the way `bin/`/
`coverage.out` do. It has no CI gate of its own today (no lint/build/test
job) because it produces no executable or published artifact directly —
`ci-web` validates `web/landing/`, the surface that actually ships.

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
- `web/design/` starts feeding an automated build step (e.g. token export
  consumed by `web/landing/` or a Console implementation) — at that point it
  needs its own CI gate like `web/landing/` has today, not just this ADR's
  monorepo-boundary umbrella.

## Consequences

- Contributors have one source of truth for Go versions and a clear Node policy
  for the web surface.
- CI changes that restate or relax toolchain versions must update this ADR and
  contributor docs.
- The monorepo boundary stays explicit without adding new package ownership
  machinery.
