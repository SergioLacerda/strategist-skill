<p align="center">
  <img src="pages/docs/banner.png" alt="Strategist — A experiência de suas demandas nunca serão a mesma" width="100%" />
</p>

<p align="center">
  <a href="https://sergiolacerda.github.io/strategist-skill/">
    <img src="https://img.shields.io/badge/⛨_Documentação_Épica-landing_page-e8c25a?style=for-the-badge&labelColor=1b1610" alt="Documentação Épica" />
  </a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/CI-passing-3fae6f?style=flat-square&labelColor=1b1610" />
  <img src="https://img.shields.io/badge/version-1.0-e8c25a?style=flat-square&labelColor=1b1610" />
  <img src="https://img.shields.io/badge/license-CC_BY--NC_4.0-cf7a2c?style=flat-square&labelColor=1b1610" />
  <img src="https://img.shields.io/badge/mode-pragmatic_·_epic-9b865d?style=flat-square&labelColor=1b1610" />
</p>

---

# Strategist Skill + SDD Harness

![CI](https://img.shields.io/badge/CI-passing-brightgreen)
![License](https://img.shields.io/badge/License-CC_BY--NC_4.0-blue)
![Docs](https://img.shields.io/badge/Docs-available-orange)
![Version](https://img.shields.io/badge/Version-1.0-yellow)

**Strategist** é uma skill autônoma que explora, analisa, refina tarefas tecnicas e as executa, documentando cada etapa. Para isso, orquestra "missões" através de papeis(slots plugáveis) — **Ranger(ou discover) → Archivist(ou refinamento) → Sniper(ou agente executor)** — dentro de um fluxo governado com approval gate obrigatório. Standalone por padrão.

Para o pipeline detalhado, fases, schemas e configuração de providers: [readme_detailed.md](readme_detailed.md).

---

### Key Capabilities

- **Slots plugáveis** – Ranger (discovery), Archivist (refinement) e Sniper (execution) são providers configuráveis;
- **Approval Gate obrigatório** – nunca implementa alterações sem aprovação humana explícita.
- **Dois modos de operação** – `pragmatic` (tom analítico) e `epic` (tom estratégico); mesmo pipeline, vocabulário diferente.
- **Knowledge Index** – contexto seletivo por `task_type` antes de cada missão; prioridades ajustadas por learning loop.
- **Learning Loop não-bloqueante** – registra outcomes e source-hints com aprovação humana; falha nunca bloqueia o resultado da missão.
- **Integração SDD opcional** – A skill se submete a governança, registrável como plugin, Aderindo a mandates, rules e guidelines definidos externamente ao plugin.

---

### Side Quests · Ataque de oportunidade

Antes da análise principal, o Strategist varre o workspace e despacha correções pontuais — sempre passando por um **mini approval gate** quando há itens detectados.

| Fase | Função |
|------|--------|
| **Housekeeping Scan** _(Ataque de oportunidade)_ | Em ações podemos encontrar problemas; despachamos em *side quests* para solucioná-los. |
| **Side Quest** | Missões pequenas com escopo simples — como mover tarefas prontas para a pasta correta ou hotfixes pontuais. |

> Pipeline completo: `Ranger → housekeeping_scan → [mini approval gate] → Sniper(side quests) → Archivist → approval gate → Sniper(main)`

---

### Fluxo Geral

![General Flow](docs/fluxo-geral.png)

---

### Fluxo de Integração SDD

![Integration Flow](docs/fluxo-integracao.png)

---

### Documentação

| Documento | Descrição |
|-----------|-----------|
| [readme_detailed.md](readme_detailed.md) | Documentação técnica completa: pipeline, slots, personas, knowledge system, SDD integration, forbidden behaviors |
| [strategist/SKILL.md](strategist/SKILL.md) | Instruções completas do agente |
| [strategist/protocol.md](strategist/protocol.md) | Regras de roteamento obrigatórias e stop conditions |
| [strategist/skill.yaml](strategist/skill.yaml) | Contrato da skill (slots, pipeline, forbidden_behaviors) |
| [.analysis/refined/bigbang-go-20260529/design.md](.analysis/refined/bigbang-go-20260529/design.md) | Decisões de design e rationale da implementação Go |

---

### Quick Workflow

**Linux / Mac / WSL — instalar (wizard por padrão):**
```bash
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash
```

> O bootstrap baixa o binário `strategist`, verifica o SHA256 e executa `strategist install`. Sem dependências externas (sem jq, yq, python3).

> **Risco de segurança — piping curl direto:** executar `curl | bash` sem especificar uma versão instala a última versão do branch `main`, sem garantia de integridade. Um ataque de supply chain ou MITM poderia substituir o script em trânsito. **Em ambientes de produção, sempre use uma versão fixada:**
```bash
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh \
  | bash -s -- --version=v1.0.0
```
> A versão fixada baixa o binário de uma release tagged no GitHub e verifica o SHA256 antes de instalar. O piping direto sem `--version` é aceitável em ambientes efêmeros (CI, dev containers), mas não em máquinas compartilhadas ou com acesso privilegiado.

**Atualizar configuração (re-rodar wizard):**
```bash
strategist install --wizard
```

---

**Onde ficam os arquivos após instalação:**

| Arquivo | Função |
|---------|--------|
| `.strategist/active.yaml` | Modo (pragmatic/epic), base_path, roles |
| `.strategist/roles/default.yaml` | Slot providers: Ranger, Archivist, Sniper |
| `.strategist/knowledge.index.yaml` | Fontes de conhecimento por task_type |
| `.analysis/` | Artefatos de missão (pending, refined, done) |



---

## 🧪 Testes

### Pré-requisitos

```bash
# Go 1.22+
go version

# Instalar dependências
go mod tidy
```

Sem jq, yq ou pyyaml. A suite de testes usa apenas `go test`.

---

### Rodar os testes

```bash
# Todos os testes (com race detector)
go test -race ./...

# Ou via Makefile
make test

# Com relatório de cobertura
make cover
```

---

### Suites

| Suite | Arquivo | Cobre |
|-------|---------|-------|
| Stale checker | `tests/stale_test.go` | 5 casos: absent, no manifest, fresh, stale source, source gone |
| Compile | `tests/compile_test.go` | Config, Domain, Index, All (4 artifacts + manifest) |
| Install | `tests/install_test.go` | Silent mode, gitignore |
| Fixtures | `tests/fixtures_test.go` | Formato dos 5 fixtures de invariantes de segurança |

---

### BDD Specs

`strategist/tests/specs/*.feature` — especificações formais dos invariantes de segurança (approval gate, slot contracts, forbidden behaviors, LearningBuffer). Documentação executável — não requerem runner separado.

---

## 📄 Licença

Este projeto está licenciado sob a **Creative Commons Atribuição-NãoComercial 4.0 Internacional (CC BY-NC 4.0)**.

Você pode usar, estudar, modificar e replicar este projeto para fins não comerciais, desde que a atribuição ao autor original seja preservada.
O uso comercial, revenda ou comercialização requer autorização prévia por escrito do titular dos direitos autorais.

- **Repositório:** <https://github.com/SergioLacerda/strategist-skill>
- **Documentação (GitHub Pages):** <https://sergiolacerda.github.io/strategist-skill/>
- **Texto completo da licença:** [`LICENSE`](LICENSE)