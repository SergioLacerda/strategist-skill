## What and why

<!-- Describe the change and the motivation. Link an issue if one exists. -->

## Checklist

- [ ] `make ci-lint ci-test` passes locally
- [ ] If embedded defaults changed (`internal/embed/defaults/`): `make build && make integration` passes
- [ ] If a slot contract, schema, or ADR-relevant decision changed: the docs under `docs/` and `internal/embed/defaults/` are consistent with each other
- [ ] Tests added or updated for the behavior change
- [ ] `CHANGELOG.md` updated under `[Unreleased]` (skip for `docs`/`chore`-only changes)

## Type of change

- [ ] `feat` — new feature or capability
- [ ] `fix` — bug fix
- [ ] `refactor` — no behavior change
- [ ] `docs` — documentation only
- [ ] `test` — test-only change
- [ ] `chore` — build process, dependencies, tooling
