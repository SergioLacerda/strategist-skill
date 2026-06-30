# ADR-0006 — E2E Test Entry Point via Full Install Pipeline

**Status:** Accepted
**Date:** 2026-06-02

---

## Context

The compilation pipeline (`embed.Extract` → `compile.CompileAll` → `stale.Check`) had no E2E tests using the real embed defaults. While writing these tests, two approaches were considered for assembling the test fixture:

**Option A — Direct:** call `embed.Extractor{}.Extract(tmpDir)`, write `active.yaml` manually from the template, then call `compile.Compiler{}.CompileAll(tmpDir, kiPath)`.

**Option B — Full install:** call `install.Service.Install` with real `embed.Extractor{}` and `compile.Compiler{}`, and use the resulting `.strategist/` directory as the test fixture.

## Decision

Option B was chosen. `install.Service.Install` is the canonical path that produces a `.strategist/` directory ready for bootstrap in production. Using it directly means the E2E tests exercise the same code path a real user goes through — including `active.yaml` generation from the template and the compile step embedded in the installer. This gives more confidence that what passes in CI is what works in the field.

Option A was rejected because it would duplicate the install logic in the tests, creating a second code path that could diverge from the real one — exactly the kind of gap the tests were designed to close.

## Consequences

- **Easier:** Any regression in `install.Service.Install` (extractor, template write, compile) is caught by the E2E tests, not just by install-specific unit tests.
- **Harder:** The E2E tests are slightly slower (full install including shim creation) and depend on the `install` package — a change to `install.Service` fields may require updating the `installReal` helper. This coupling is intentional: the helper should track the canonical install API.
- **Accepted trade-off:** `mockExtractor` in `tests/install_test.go` needed a `ReadFile` stub because `domain.FileExtractor` gained the method during the migration to Go. This was a latent interface drift that the E2E work exposed and fixed.
