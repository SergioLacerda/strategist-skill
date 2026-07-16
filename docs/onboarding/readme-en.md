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

**Strategist** turns a technical request into a governed documentation mission:
discovery, refinement, and approved documentation/handoff materialization.

Canonical pipeline:
`Ranger → Archivist → approval gate → Sniper`

For full pipeline/contracts/schema details: [readme-detailed-en.md](readme-detailed-en.md).

## Why use it

- Discovery and refinement before execution.
- Mandatory approval gate before Sniper materializes approved documentation or handoff work.
- Pluggable slots (`discovery`, `refinement`, `execution`).
- Mission policy through `mission_mode` (analysis vs delivery).
- Quick Draw, Opportunity Attack, Side Quests, Critical Hit, and Treasure Chests in the same flow.

## How it works

- **Scout** (internal, pre-pipeline): classifies the request and picks the route before any slot runs. Does not perform discovery itself.
- **Ranger**: explores context and produces discovery.
- **Archivist**: turns discovery into proposal, design, and tasks.
- **Sniper**: materializes approved documentation/handoff work only after gate + policy checks.
- **Opportunity Attack**: Archivist-owned ADR evaluation after all four refined artifacts are written.
- **Side Quests**: cross-phase scope observations; Archivist consolidates and presents at the approval gate; Sniper reports newly discovered ones.
- **Critical Hit**: analysis `.md` movement route within `<base_path>` folders (`pending/`, `refined/`, `archived/`).
- **Treasure Chests**: offline scoped context sources. `strategist treasure-chest index`/`mine` currently manage jewels (compact source-linked knowledge points, `proposed → accepted/verified/deprecated`); jewel runtime consultation and Scout telemetry are documented as target behavior — see `docs/observability-contract.md`.

## Generated files

| Path | Purpose |
|---|---|
| `.strategist/active.yaml` | mode, language, slots, mission policy |
| `.strategist/knowledge.index.yaml` | sources by `task_type` |
| `.analysis/` | `pending`, `refined`, `archived` artifacts |

## Minimal slot configuration

```yaml
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sniper
```

Expected contracts:
- Ranger: `write_analysis`
- Archivist: `write_analysis`
- Sniper: `controlled`

## General Flow

![General Flow](fluxo-geral_en.png)

## SDD Integration Flow

![SDD Integration Flow](fluxo-integracao_en.png)

## Explore more

- [readme-detailed-en.md](readme-detailed-en.md)
- [configuration.md](../configuration.md)
- [cli-reference.md](../cli-reference.md)
- [architecture.md](../architecture.md)
- [skill-internals.md](../skill-internals.md)
- [c4-diagrams.md](../c4-diagrams.md)
- [adr/](../adr/)
- [strategist/SKILL.md](../../strategist/SKILL.md)
- [strategist/protocol.md](../../strategist/protocol.md)

## Development and tests

- `make build` — compiles the local binary into `bin/strategist`
- `make test-lite` — runs the isolated test slices that do not download new dependencies
- `make test` — runs unit and package tests
- `make integration` — runs E2E/integration tests with `-tags=integration`
- `make test-all` — runs `test` + `integration`
- `make bench` — runs benchmarks
- `make cover` — generates per-package coverage
- `make cover-gate` — fails if any internal package is below 90% coverage
- `make cover-html` — generates a consolidated `coverage.html` report
- When contributing external skills used as wizard default providers, preserve attribution to the upstream project and include an installable canonical manifest at `.strategist/skills/<provider>/skill.yaml`.

```bash
make build
make test-lite
make test
make integration
make test-all
make bench
make cover
make cover-gate
make cover-html
```

### Role in the flow

- `brainstorming` is referenced as the basis for structured exploration in the `Ranger` role, including option discovery, risks, trade-offs, and early clarification.
- `openspec` is referenced as the basis for convergent formalization in the `Archivist` role, including change proposal, design, tasks, and acceptance criteria.
- When installed as wizard default providers, these skills remain independent; `strategist` only orchestrates their use inside the pipeline.
