# Contributing to Strategist

## Prerequisites

- Go matching `go.mod` (`go 1.26.4`, toolchain `go1.26.5`)
- Node.js 22 (for `web/landing/` only, matching CI)
- `make` (GNU Make)

Go versions are sourced from `go.mod`: `go 1.26.4` defines the language/module
target and `toolchain go1.26.5` pins the patch toolchain used by CI-compatible
local verification. Node is intentionally scoped to `web/landing/`; CI uses
Node 22 and the landing package declares `engines.node >= 22.12.0`. Relax or bump
these pins only through the toolchain policy in ADR-0014.

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
| `make ci-lint` | Run formatting, module, vet, build, and quality-budget gates |
| `make ci-test` | Run unit, spec, integration, convergence, contract, and coverage gates |
| `make cover` | Run coverage for packages listed in `scripts/coverage-packages.tsv` |
| `make cover-html` | Generate `coverage/coverage.html` without opening a browser |
| `make quality-budget-gate` | Enforce Go file-size and cognitive-complexity budgets |
| `make compile-skill` | Compile the Strategist skill artifacts |
| `make build-all` | Build CLI + landing site |

> **Important:** `internal/embed/defaults/` is the single authoring source for
> packaged Strategist defaults. After editing embedded defaults, run
> `make build` and the relevant tests; installed `.strategist/` directories are
> runtime instances and should not be treated as source mirrors.

## Release history

`CHANGELOG.md` is the curated source for unreleased changes and the `1.0.0`
baseline. For patch releases after `1.0.0`, GitHub Releases are authoritative
for published notes and assets. Contributors should not invent or reconstruct
patch notes in `CHANGELOG.md`; verify the tag and release page instead.

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
  embed/defaults/    Embedded YAML/Markdown defaults packaged via go:embed
.strategist/         Local runtime instance generated from embedded defaults
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
3. Run `make ci-lint ci-test` — both must pass
4. If you changed embedded defaults: `make build && make integration`
5. Open a PR against `main` with a clear description of what and why

## Governance files

The `.sdd/` directory contains governance artifacts managed by the SDD CLI.
Do not edit files under `.sdd/` manually — use `sdd` commands or the Strategist
skill to propose changes through the pipeline.

## Getting help

- Read `docs/mental-model.md` for how Strategist thinks about missions
- Read `docs/architecture.md` for the component layout
- Open an issue on GitHub for bugs or questions
