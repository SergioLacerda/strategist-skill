# Phase D evidence checklist

- Static: validate budget, cache identity, evidence projection, and extension registration inputs.
- Persisted: retain source evidence and last-known-good values; do not rewrite Custom locks.
- Structural: run focused unit tests, race tests, vet, lint, quality budgets, and OpenSpec validation.
- Live: treat unavailable or static-only evidence as non-certifying.
- Compatibility: enable additive `v1` extension behavior only after identity validation.
- Rollback: disable cache or extension independently and retain uncached/core behavior.

Stop on malformed budgets, divergent identities, authority conflicts, unsupported
extensions, validation failure, or unavailable live evidence presented as success.

Mission CLI/context invocation, live-provider CI publication, OCI/remote
distribution, hosted release lineage, and arbitrary workflow graphs remain out
of scope for Phase D.
