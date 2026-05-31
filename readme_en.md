<p align="center">
  <img src="https://capsule-render.vercel.app/api?type=rect&color=0:140f08,50:1d150c,100:271c10&height=210&text=STRATEGIST&fontColor=f0d896&fontSize=74&fontAlignY=45&desc=%E2%9C%A6%20%20Your%20experience%20with%20your%20demands%20will%20never%20be%20the%20same.%20%20%E2%9C%A6&descSize=16&descColor=cf7a2c&descAlignY=70" alt="Strategist — Your experience with your demands will never be the same." width="100%" />
</p>

<div align="center">

<img src="https://img.shields.io/badge/CI-passing-3fae6f?style=flat-square&labelColor=1b1610" />
<img src="https://img.shields.io/badge/version-1.0-e8c25a?style=flat-square&labelColor=1b1610" />
<img src="https://img.shields.io/badge/license-CC_BY--NC_4.0-cf7a2c?style=flat-square&labelColor=1b1610" />
<img src="https://img.shields.io/badge/mode-pragmatic_·_epic-9b865d?style=flat-square&labelColor=1b1610" />

**Authors** · Sergio Lacerda &amp; Raphael Vernil

<br/>

<a href="https://sergiolacerda.github.io/strategist-skill/"><img src="https://img.shields.io/badge/⛨_EPIC_DOCUMENTATION-✦_OPEN_LANDING_✦-cf7a2c?style=for-the-badge&labelColor=1b1610&color=e8c25a" alt="Epic Documentation" height="42" /></a>

<a href="readme.md"><img src="https://img.shields.io/badge/📖_DOCUMENTATION-🇧🇷_PORTUGUÊS-1b1610?style=for-the-badge&labelColor=2a2118&color=3fae6f" alt="Português" /></a><a href="README.en.md"><img src="https://img.shields.io/badge/-🇺🇸_ENGLISH-4a7fb0?style=for-the-badge" alt="English" /></a> &nbsp; <a href="#quick-workflow"><img src="https://img.shields.io/badge/❯_QUICK_WORKFLOW-9b865d?style=for-the-badge&labelColor=1b1610" alt="Quick Workflow" /></a>

━━━━━━━━━━━━━━━━━━━━ ⟡ ━━━━━━━━━━━━━━━━━━━━

</div>

<p align="center">
  <i>An autonomous skill that <b>orchestrates multi-phase missions</b> through pluggable roles.<br/>
  The Strategist orchestrates the mission, entrusting each step to a specialist and waiting at the <b>approval gate</b>.</i>
</p>

---

# Strategist Skill

## What it is

**Strategist** turns a technical request into a governed mission: discovery, refinement, and execution only with explicit human approval.

Canonical pipeline:
`Ranger → Archivist → approval gate → Sniper`

For full pipeline/contracts/schema details: [readme_detailed_en.md](readme_detailed_en.md).

## Why use it

- Discovery and refinement before execution.
- Mandatory approval gate for writes/implementation.
- Pluggable slots (`discovery`, `refinement`, `execution`).
- Mission policy through `mission_mode` (analysis vs delivery).
- Quick Draw, Opportunist Attack, and Treasure Chests in the same flow.

## How it works

- **Ranger**: explores context and produces discovery.
- **Archivist**: turns discovery into proposal, design, and tasks.
- **Sniper**: executes only after gate + policy checks.
- **Opportunist Attack**: finds side quests without parallel pipelines.
- **Treasure Chests**: offline scoped context sources.

## Quick install

Linux / macOS / WSL:

```bash
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash
```

Pinned version (recommended for sensitive environments):

```bash
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh \
  | bash -s -- --version=v1.0.0
```

Reconfigure:

```bash
strategist install --wizard
```

## Generated files

| Path | Purpose |
|---|---|
| `.strategist/active.yaml` | mode, language, slots, mission policy |
| `.strategist/knowledge.index.yaml` | sources by `task_type` |
| `.analysis/` | `pending`, `refined`, `done` artifacts |

## Minimal slot configuration

```yaml
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
```

Expected contracts:
- Ranger: `write_pending`
- Archivist: `write_analysis`
- Sniper: `controlled`

## General Flow

![General Flow](docs/fluxo-geral_en.png)

## SDD Integration Flow

![SDD Integration Flow](docs/fluxo-integracao_en.png)

## Explore more

- [readme_detailed_en.md](readme_detailed_en.md)
- [docs/configuration.md](docs/configuration.md)
- [docs/cli-reference.md](docs/cli-reference.md)
- [docs/architecture.md](docs/architecture.md)
- [docs/skill-internals.md](docs/skill-internals.md)
- [docs/c4-diagrams.md](docs/c4-diagrams.md)
- [docs/adr/](docs/adr/)
- [strategist/SKILL.md](strategist/SKILL.md)
- [strategist/protocol.md](strategist/protocol.md)

## Development and tests

```bash
make build
make test
make cover
```

## License

CC BY-NC 4.0. Commercial use requires prior authorization.

- Repository: <https://github.com/SergioLacerda/strategist-skill>
- Documentation: <https://sergiolacerda.github.io/strategist-skill/index.html?lang=en>
- Full text: [LICENSE](LICENSE)
