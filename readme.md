<p align="center">
  <img src="https://capsule-render.vercel.app/api?type=rect&color=0:140f08,50:1d150c,100:271c10&height=210&text=STRATEGIST&fontColor=f0d896&fontSize=74&fontAlignY=45&desc=%E2%9C%A6%20%20Sua%20experi%C3%AAncia%20com%20suas%20demandas%20nunca%20ser%C3%A1%20a%20mesma.%20%20%E2%9C%A6&descSize=16&descColor=cf7a2c&descAlignY=70" alt="Strategist — A experiência com suas demandas nunca será a mesma." width="100%" />
</p>

<div align="center">

<img src="https://img.shields.io/badge/CI-passing-3fae6f?style=flat-square&labelColor=1b1610" />
<img src="https://img.shields.io/badge/version-1.0-e8c25a?style=flat-square&labelColor=1b1610" />
<img src="https://img.shields.io/badge/license-CC_BY--NC_4.0-cf7a2c?style=flat-square&labelColor=1b1610" />
<img src="https://img.shields.io/badge/mode-pragmatic_·_epic-9b865d?style=flat-square&labelColor=1b1610" />

**Autores** · Sergio Lacerda &amp; Raphael Vernil

<br/>

<a href="https://sergiolacerda.github.io/strategist-skill/"><img src="https://img.shields.io/badge/⛨_DOCUMENTAÇÃO_ÉPICA-✦_ABRIR_LANDING_✦-cf7a2c?style=for-the-badge&labelColor=1b1610&color=e8c25a" alt="Documentação Épica" height="42" /></a>

<a href="readme.md"><img src="https://img.shields.io/badge/📖_DOCUMENTAÇÃO-🇧🇷_PORTUGUÊS-1b1610?style=for-the-badge&labelColor=2a2118&color=3fae6f" alt="Português" /></a><a href="README.en.md"><img src="https://img.shields.io/badge/-🇺🇸_ENGLISH-4a7fb0?style=for-the-badge" alt="English" /></a> &nbsp; <a href="#quick-workflow"><img src="https://img.shields.io/badge/❯_QUICK_WORKFLOW-9b865d?style=for-the-badge&labelColor=1b1610" alt="Quick Workflow" /></a>

━━━━━━━━━━━━━━━━━━━━ ⟡ ━━━━━━━━━━━━━━━━━━━━

</div>

<p align="center">
  <i>Uma skill autônoma que <b>orquestra missões multi-fase</b> através de três papéis plugáveis.<br/>
  O Estrategista orquestra a missão, confiando cada etapa a um especialista e aguarda no <b>portão de aprovação</b>.</i>
</p>

---

# Strategist Skill

## O que é

**Strategist** transforma uma solicitação técnica em missão governada: descoberta, refinamento e execução só com aprovação humana.

Pipeline canônico:
`Ranger → Archivist → approval gate → Sniper`

Para detalhes completos de pipeline, contratos e schemas: [readme_detailed.md](readme_detailed.md).

## Por que usar

- Discovery e refinement antes de execução.
- Approval gate obrigatório para escrita/implementação.
- Slots plugáveis (`discovery`, `refinement`, `execution`).
- Política de missão por `mission_mode` (análise vs entrega).
- Quick Draw, Opportunist Attack e Treasure Chests no mesmo fluxo.

## Como funciona

- **Ranger**: explora o contexto e gera discovery.
- **Archivist**: transforma em proposta, design e tasks.
- **Sniper**: executa somente após gate + política.
- **Opportunist Attack**: encontra side quests sem abrir pipeline paralelo.
- **Treasure Chests**: fontes offline de contexto por escopo.

## Instalação rápida

Linux / macOS / WSL:

```bash
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash
```

Versão fixada (recomendado para ambientes sensíveis):

```bash
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh \
  | bash -s -- --version=v1.0.0
```

Reconfigurar:

```bash
strategist install --wizard
```

## Arquivos gerados

| Caminho | Função |
|---|---|
| `.strategist/active.yaml` | modo, idioma, slots, policy da missão |
| `.strategist/knowledge.index.yaml` | fontes por `task_type` |
| `.analysis/` | artefatos `pending`, `refined`, `done` |

## Configuração mínima de slots

```yaml
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
```

Contratos esperados:
- Ranger: `write_pending`
- Archivist: `write_analysis`
- Sniper: `controlled`

## Fluxo Geral

![General Flow](docs/fluxo-geral.png)

## Fluxo de Integração SDD

![Integration Flow](docs/fluxo-integracao.png)

## Explore mais

- [readme_detailed.md](readme_detailed.md)
- [docs/configuration.md](docs/configuration.md)
- [docs/cli-reference.md](docs/cli-reference.md)
- [docs/architecture.md](docs/architecture.md)
- [docs/skill-internals.md](docs/skill-internals.md)
- [docs/c4-diagrams.md](docs/c4-diagrams.md)
- [docs/adr/](docs/adr/)
- [strategist/SKILL.md](strategist/SKILL.md)
- [strategist/protocol.md](strategist/protocol.md)

## Desenvolvimento e testes

```bash
make build
make test
make cover
```

## Licença

CC BY-NC 4.0. Uso comercial requer autorização prévia.

- Repositório: <https://github.com/SergioLacerda/strategist-skill>
- Documentação: <https://sergiolacerda.github.io/strategist-skill/index.html?lang=pt>
- Texto completo: [LICENSE](LICENSE)
