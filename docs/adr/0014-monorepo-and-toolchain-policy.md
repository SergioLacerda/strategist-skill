# ADR-0014 - Monorepo and toolchain policy

**Status:** Accepted
**Date:** 2026-08-03
**Amended:** 2026-08-30 — extended scope to `web/design/`
**Amended:** 2026-10-07 — landing site and `web/design/` removed; the public site is published from the external ILUSIONISTA surface release
**Context:** `20260803-cicd-policy-eval-residuals`

---

## Context

Strategist ships a Go CLI and its documentation from one repository:

- Go runtime code lives under `cmd/` and `internal/`.
- Embedded Strategist defaults live under `internal/embed/defaults/`.
- Documentation lives under `docs/`.

The public site is no longer part of this repository. Since the Pages
migration (ILUSIONISTA surface `v0.1.0`) it is published by
`.github/workflows/pages.yml` from an external release owned by another
project, and the former in-repository Astro landing (`web/landing/`) and its
design workspace (`web/design/`) were removed.

The repository carries strict toolchain declarations: `go.mod` sets
`go 1.27.1` and `toolchain go1.27.1`. Node.js is not part of the product; it
is only a tooling prerequisite (see Toolchain policy).

## Decision

Keep the Go CLI, embedded defaults, and docs in one repository and one release
train.

CI ownership is split by surface:

- Go validation is owned by Make targets such as `ci-lint`, `ci-test`,
  `release-verify`, `cover-gate`, `quality-budget-gate`, and
  `release-reproducible-check`.
- Documentation governance is owned by `docs-governance-gate` and the spec tests
  that assert normative wording.
- Release publication is owned by the release workflow plus the Make release
  gates.
- Public site publication is owned by `pages.yml`, which only consumes a pinned
  external release and builds nothing locally.

Toolchain policy:

- Go version authority is `go.mod`. Workflows should use `go-version-file:
  "go.mod"` instead of restating the Go version.
- `go 1.27.1` is the module/language target.
- `toolchain go1.27.1` is the exact patch toolchain for CI-compatible local
  verification.
- Node is tooling-only: the host Node `>=20.19.0` required by the embedded
  OpenSpec runtime, and the manual `promptfoo/` evals. It is not a product
  surface and has no application build in this repository.
- Toolchain pins may be relaxed or bumped only when CI, local verification, and
  contributor documentation are updated together.

## Reconsider When

- A web or Node surface returns to this repository; it then needs its own CI
  gate and an amendment to this ADR.
- The Go CLI no longer needs to package Strategist defaults from the same tree.
- Release verification can prove a wider Go toolchain range without changing
  emitted binaries or generated defaults.

## Rollback

`web/landing/` and `web/design/` exist in every commit up to and including
`8fd5d16d8b0f692e00ebdbf9a2bc062b9f90c475` (the parent of the removal).
Restoring them means checking out those paths from that commit together with
the previous `pages.yml`. To roll back only the public site, point the pins in
`pages.yml` at a previous ILUSIONISTA release tag and run it via
`workflow_dispatch`.

## Consequences

- Contributors have one source of truth for Go versions and a clear, narrow
  Node policy.
- CI changes that restate or relax toolchain versions must update this ADR and
  contributor docs.
- The monorepo boundary stays explicit without adding new package ownership
  machinery.
