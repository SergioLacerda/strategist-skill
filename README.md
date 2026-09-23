# Strategist Skill

[![Release](https://img.shields.io/github/v/release/SergioLacerda/strategist-skill?label=release)](https://github.com/SergioLacerda/strategist-skill/releases)
[![CI](https://github.com/SergioLacerda/strategist-skill/actions/workflows/test.yml/badge.svg)](https://github.com/SergioLacerda/strategist-skill/actions/workflows/test.yml)
[![Coverage](https://img.shields.io/badge/coverage-92.8%25-green)](docs/test-styles.md)
[![Mutation](https://img.shields.io/badge/mutation-passing-brightgreen)](scripts/mutation-role-weapon.sh)
[![Go](https://img.shields.io/badge/Go-1.26-blue)](https://go.dev)
[![License](https://img.shields.io/github/license/SergioLacerda/strategist-skill)](LICENSE)

A governed AI mission orchestrator.

Strategist coordinates multi-phase work through pluggable role providers:

```text
Ranger → Archivist → Approval Gate → Sniper
```

It governs discovery, refinement, handoffs, approval, and artifact materialization while keeping implementation work explicitly separated from documentation approval.

## Install

Requires Go 1.26+.

```bash
go install github.com/SergioLacerda/strategist-skill/cmd/strategist@latest
```

## Setup

Run the installation wizard:

```bash
strategist install --wizard
```

The wizard configures the Strategist runtime, workspace, and available role providers.

## Check

Validate the installation and configuration:

```bash
strategist check
```

Use `check` after installation or configuration changes to confirm that the runtime is ready.

## Start a mission

Invoke Strategist with your mission:

```text
/strategist <your mission prompt>
```

For a guided first run, see the [Quickstart](QUICKSTART.md).

## How it works

```text
Mission
  ↓
Ranger
  ↓
Archivist
  ↓
Approval Gate
  ↓
Sniper
```

- **Ranger** performs discovery.
- **Archivist** refines the mission into a governed, reviewable package.
- **Approval Gate** requires explicit human approval before approved targets can be materialized.
- **Sniper** handles the approved execution/documentation stage defined by the workflow.

Acceptance at the Approval Gate authorizes only the declared documentation targets. Source code, tests, scripts, CI, and configuration changes remain a separate implementation handoff.

## Documentation

### Getting started

- [Quickstart](QUICKSTART.md)
- [Conceptual quickstart](docs/onboarding/quickstart-concepts.md)

### Reference

- [CLI reference](docs/cli-reference.md)
- [Configuration](docs/configuration.md)
- [Architecture](docs/architecture/overview.md)
- [Technical guide](docs/onboarding/readme-en.md)
- [Detailed guide](docs/onboarding/readme-detailed-en.md)

### Documentation modes

- [Pragmatic Mode](https://sergiolacerda.github.io/strategist-skill/pragmatic/) — direct technical guide.
- [Epic Mode](https://sergiolacerda.github.io/strategist-skill/epic/) — narrative documentation experience.
- [`docs/`](docs/) — repository documentation index.

## Security

See [SECURITY.md](SECURITY.md) for:

- vulnerability disclosure;
- release integrity;
- GitHub build provenance;
- Cosign keyless signatures;
- SHA256 checksum verification.

Detailed release-verification commands intentionally live outside the main README so the onboarding path remains short.

## Third-party skills and attribution

`strategist` acts as an orchestration layer for governed workflows and may integrate external skills as specialized role providers.

Recognized base skills:

- `brainstorming` — project `obra/superpowers`  
  Source: <https://claudemarketplaces.com/skills/obra/superpowers/brainstorming>
- `openspec` — project `itechmeat/llm-code`  
  Source: <https://claudemarketplaces.com/skills/itechmeat/llm-code/openspec>

When integrating external skills:

- preserve the upstream name, project, and public URL;
- do not imply ownership of upstream prompts, artifacts, or implementation;
- prefer canonical manifests/adapters in `.strategist/skills/<provider>/skill.yaml` instead of duplicating upstream content.

## License

MIT. Open source — free to use, modify, and redistribute with attribution under the terms of the [LICENSE](LICENSE).

- Repository: <https://github.com/SergioLacerda/strategist-skill>
- Documentation: <https://sergiolacerda.github.io/strategist-skill/index.html?lang=en>
