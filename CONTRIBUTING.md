# Contributing to Strategist

## Prerequisites

- Go 1.24+ (`go version`)
- Node.js 20+ (for `web/landing/` only)
- `make` (GNU Make)

## Setup

```bash
git clone https://github.com/SergioLacerda/strategist-skill.git
cd strategist-skill
go mod download
make build          # builds the CLI binary to bin/strategist
```

## Common tasks

| Command | What it does |
|---------|-------------|
| `make build` | Build the CLI binary |
| `make test` | Run unit tests |
| `make test-all` | Unit + spec + integration tests |
| `make lint` | Run formatting, golangci-lint, and informational source-size reports |
| `make cover` | Run tests with coverage report |
| `make cover-html` | Open coverage report in browser |
| `make sync-embed` | Sync embedded YAML files into the binary |
| `make compile-skill` | Compile the Strategist skill artifacts |
| `make build-all` | Build CLI + landing site |

> **Important:** After editing any file under `internal/embed/defaults/` or
> `strategist/`, run `make sync-embed && make build` to apply changes to the binary.

### Landing page: build before preview

`npm run preview` (Astro's own command) serves the contents of `dist/`,
which only exists after a build — running `preview` without building first
fails with "output directory ... does not exist". `make install-web` only
runs `npm ci`; it does not build.

```bash
make build-site          # installs + builds web/landing/dist/
cd web/landing && npm run preview
```

## Running tests

```bash
make test           # fast unit tests
make integration    # integration tests (requires built binary)
make spec           # spec/contract tests
```

## Project layout

```
cmd/strategist/      CLI commands (cobra)
internal/
  domain/            Core types, errors, ports
  install/           Installation service
  compile/           Skill compilation
  telemetry/         OTel spans + structured log attributes
  embed/defaults/    Embedded YAML/Markdown files (synced via make sync-embed)
strategist/          Skill contract files (personas, contracts, schemas)
docs/                Documentation
web/landing/         Astro landing site
```

## Commit conventions

Format: `<type>(<scope>): <description>`

| Type | When to use |
|------|-------------|
| `feat` | New feature or capability |
| `fix` | Bug fix |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `docs` | Documentation only |
| `test` | Adding or updating tests |
| `chore` | Build process, dependencies, tooling |

Examples:
```
feat(install): add structured logs to Install() happy path
fix(gate): prevent approval bypass when execution_gate=blocked
docs: add observability-contract.md
```

## Opening a pull request

1. Fork and create a branch: `git checkout -b feat/my-change`
2. Make your changes + add tests
3. Run `make test lint` — both must pass
4. If you changed embedded files: `make sync-embed && make build && make integration`
5. Open a PR against `main` with a clear description of what and why

## Governance files

The `.sdd/` directory contains governance artifacts managed by the SDD CLI.
Do not edit files under `.sdd/` manually — use `sdd` commands or the Strategist
skill to propose changes through the pipeline.

## Getting help

- Read `docs/mental-model.md` for how Strategist thinks about missions
- Read `docs/architecture.md` for the component layout
- Open an issue on GitHub for bugs or questions
