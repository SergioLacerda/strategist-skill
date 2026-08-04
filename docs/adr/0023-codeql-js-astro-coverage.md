# ADR-0023 — CodeQL Coverage: `javascript-typescript` Matrix Leg for `web/landing/`

**Status:** Accepted  
**Date:** 2026-08-04  
**Context:** `20260804-codeql-js-astro-coverage`

---

## Context

`.github/workflows/codeql.yml`'s only job analyzed Go exclusively
(`languages: go`), leaving `web/landing/` — a real Astro + React +
TypeScript marketing-site app with substantive first-party source under
`src/` (`.astro`, `.tsx` files) — entirely unscanned by CodeQL. CodeQL
has no dedicated Astro extractor; `javascript-typescript` is the correct
available mechanism, covering `.js`/`.jsx`/`.ts`/`.tsx` files and the
`<script>`/TypeScript frontmatter blocks embedded in `.astro` files,
though not template markup outside those blocks.

`.github/workflows/test.yml`'s `site-build` job already established the
working Node/npm setup pattern for this subsystem (`actions/setup-node`
pinned SHA, `node-version: '22'`, npm cache keyed on
`web/landing/package-lock.json`, then `npm ci`).

## Decision

**DEC-1:** Add `javascript-typescript` as a second leg of a
`strategy.matrix.language` on the existing `analyze` job (not a second,
independent job). `fail-fast: false` preserves per-language failure
isolation. The Go leg's setup (`actions/setup-go`, `make build`) and the
new JS/TS leg's setup (`actions/setup-node`, `npm ci` in `web/landing/`)
are each conditioned on `matrix.language`. `category` generalizes to
`/language:${{ matrix.language }}`.

**DEC-2:** The JS/TS leg installs dependencies but runs no build step,
and is scoped via an explicit `paths: web/landing/**` allowlist on
CodeQL's `init` step (not a `dist/`/`coverage/` `paths-ignore`
blocklist).

### Alternatives Considered and Rejected

- **A second, fully independent job** (DEC-1). Rejected: duplicates
  job-level `permissions`/trigger configuration for no functional
  benefit over a matrix leg, and diverges from CodeQL's own default
  multi-language template without a repo-specific reason to.
- **No `paths` scoping, rely on `dist/`/`coverage/` being gitignored**
  (DEC-2). Rejected: correct today, but fragile — silently starts
  scanning any future non-`web/landing/` Node content (e.g. `promptfoo/`,
  added earlier this session, a second Node subtree with no first-party
  source yet but a real precedent) without a deliberate decision.
- **Add a build step (`make build-site`) before analysis** (DEC-2).
  Rejected: unnecessary per GitHub's own `javascript-typescript`
  extractor guidance (no build step needed), and would reintroduce the
  exact `dist/` noise the `paths` scoping is meant to avoid handling
  defensively.

## Consequences

- `web/landing/`'s first-party Astro/React/TypeScript source gains
  CodeQL `security-extended`/`security-and-quality` coverage, matching
  what the Go code already has.
- Astro template markup outside `<script>`/frontmatter TypeScript
  blocks remains unanalyzed — a known, accepted CodeQL/Astro ecosystem
  limitation, not something this decision closes or promises to.
- The CodeQL Security tab will show two distinct categories going
  forward (`/language:go`, `/language:javascript-typescript`) instead of
  one; existing Go findings/history are unaffected — nothing about the
  Go leg's configuration changed.
- One implementation-time item is explicitly deferred, not decided here:
  whether the pinned `codeql-action/init` SHA already used by the Go leg
  accepts a conditional/empty `paths` input cleanly inside one shared
  matrix step, or whether the two legs need fully separate `init`/
  `analyze` step pairs. Either shape satisfies DEC-1/DEC-2; it is a
  YAML-ergonomics detail to resolve against a real CI run, not an
  architectural decision.
- Future Node subdirectories added to this repository (beyond
  `web/landing/` and `promptfoo/`) should extend the `paths` allowlist
  deliberately rather than relying on default whole-checkout scanning.
