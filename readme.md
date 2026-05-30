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

Antes da análise principal, o Strategist varre o workspace detectando artefatos inconsistentes. O resultado vai para o Archivist como contexto — a execução das side quests acontece quando o Sniper é liberado pelo gate principal (junto com a missão principal).

| Fase | Função |
|------|--------|
| **Housekeeping Scan** _(Ataque de oportunidade)_ | Detecta artefatos stale em `todo/`, `pending/`, `refined/` e monta um manifesto de side quests. |
| **Side Quest** | Missões pequenas executadas pelo Sniper após aprovação no gate principal — mover tarefas prontas, promover artefatos, hotfixes pontuais. |

> Pipeline completo: `Ranger → housekeeping_scan → Archivist → approval gate → Sniper(side quests + main)`

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
| [docs/architecture.md](docs/architecture.md) | Arquitetura Go: mapa de pacotes, fluxo de instalação, pipeline de compilação, interfaces de domínio |
| [docs/cli-reference.md](docs/cli-reference.md) | Referência de todos os comandos CLI com flags e exemplos |
| [docs/configuration.md](docs/configuration.md) | Schemas completos: active.yaml, roles, personas, knowledge index |
| [docs/skill-internals.md](docs/skill-internals.md) | Sub-skills, contratos de fase, schemas de intake/progress, write scopes |
| [docs/c4-diagrams.md](docs/c4-diagrams.md) | Diagramas C4: contexto, containers, componentes Go e pipeline do runtime |
| [docs/adr/](docs/adr/) | Architecture Decision Records: 5 decisões fundamentais do projeto |
| [strategist/SKILL.md](strategist/SKILL.md) | Instruções completas do agente |
| [strategist/protocol.md](strategist/protocol.md) | Regras de roteamento obrigatórias e stop conditions |
| [strategist/skill.yaml](strategist/skill.yaml) | Contrato da skill (slots, pipeline, forbidden_behaviors) |

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
| `.strategist/active.yaml` | Modo, base_path, slots, language, adr_enabled |
| `.strategist/knowledge.index.yaml` | Fontes de conhecimento por task_type |
| `.analysis/` | Artefatos de missão (pending, refined, done) |

---

**Configurando os papéis (slots):**

Cada papel da missão é uma skill plugável. Os providers ficam diretamente em `active.yaml`, na chave `slots`:

```yaml
# .strategist/active.yaml
slots:
  discovery: brainstorming      # Ranger  — explora e documenta o problema
  refinement: openspec-explore  # Arquivista — refina e estrutura o plano
  execution: sdd-ask            # Sniper  — executa o plano aprovado
```

Para trocar um provider, edite `active.yaml` e aponte para qualquer skill disponível no seu ambiente. O preflight valida os contratos (`risk_score`) antes de iniciar a missão.

**Providers disponíveis por slot:**

| Slot | Contract exigido | Providers testados |
|------|-----------------|-------------------|
| Ranger (discovery) | `write_pending` | `brainstorming` |
| Arquivista (refinement) | `write_analysis` | `openspec-explore`, `openspec-propose`, `archivist`, `sdd-diagnose`, `sdd-review-architecture` |
| Sniper (execution) | `controlled` | `sdd-ask`, `sdd-ask-full`, `openspec-apply-change`, `sdd-converge`, `sdd-correct` |

Novos providers podem ser registrados em `.strategist/templates/known-providers.yaml` caso não declarem `risk_score` no próprio `skill.yaml`.

---

### Instalação local (build from source)

Para contribuir ou usar a versão mais recente do repositório sem aguardar um release:

```bash
# 1. Compilar o binário (embute os defaults atuais de strategist/ no binário)
make build          # → bin/strategist

# 2. Instalar no PATH
make install-local  # → ~/.local/bin/strategist

# 3. Garantir que ~/.local/bin está no PATH (adicione ao .bashrc/.zshrc se necessário)
export PATH="$HOME/.local/bin:$PATH"

# 4. Instalar a skill no repositório atual
strategist install --wizard
```

> **Por que `make build` antes do `install`?** O binário embute os arquivos de `strategist/` em tempo de compilação (`embed.FS`). Sem rebuild, o `strategist install` instala os defaults da versão anterior do binário, não os do repositório local.

> O Quick Workflow (`curl | bash`) não precisa desse passo — o bootstrap baixa um binário pré-compilado da release do GitHub.

---

## 🧪 Testes

### Pré-requisitos

```bash
# Go 1.26+
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
| Install | `tests/install_test.go` · `internal/install/installer_whitebox_test.go` | Silent mode, gitignore, whitebox (ensureGitignore, error propagation) |
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