# Strategist Skill + SDD Harness

![CI](https://img.shields.io/badge/CI-passing-brightgreen)
![License](https://img.shields.io/badge/License-MIT-blue)
![Docs](https://img.shields.io/badge/Docs-available-orange)
![Version](https://img.shields.io/badge/Version-1.0-yellow)

**Strategist** é uma skill autônoma que explora, analisa, refina tarefas tecnicas e as executa, documentando cada etapa. Para isso, orquestra "missões" através de papeis(slots plugáveis) — **Ranger(ou discover) → Archivist(ou refinamento) → Sniper(ou agente executor)** — dentro de um fluxo governado com approval gate obrigatório. Standalone por padrão.

---

### Key Capabilities

- **Slots plugáveis** – Ranger (discovery), Archivist (refinement) e Sniper (execution) são providers configuráveis;
- **Approval Gate obrigatório** – nunca implementa alterações sem aprovação humana explícita.
- **Dois modos de operação** – `pragmatic` (tom analítico) e `epic` (tom estratégico); mesmo pipeline, vocabulário diferente.
- **Knowledge Index** – contexto seletivo por `task_type` antes de cada missão; prioridades ajustadas por learning loop.
- **Learning Loop não-bloqueante** – registra outcomes e source-hints com aprovação humana; falha nunca bloqueia o resultado da missão.
- **Integração SDD opcional** – A skill se submete a governança, registrável como plugin, Aderindo a mandates, rules e guidelines definidos externamente ao plugin.

---

### Quick Workflow

**Linux / Mac / WSL — instalar (wizard por padrão):**
```bash
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash
```

**Windows PowerShell — instalar:**
```powershell
irm https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.ps1 | iex
```

**Atualizar configuração (re-rodar wizard):**
```bash
# Linux / Mac / WSL
bash /path/to/strategist-skill/strategist/install.sh --wizard

# Windows PowerShell
.\bootstrap.ps1 -Silent  # baixa versão mais recente
# depois edite: .strategist/roles/default.yaml
```

**Instalação Local:**
```bash
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash -s -- --silent
```

**Onde ficam os arquivos após instalação:**

| Arquivo | Função |
|---------|--------|
| `.strategist/active.yaml` | Modo (pragmatic/epic), base_path, roles |
| `.strategist/roles/default.yaml` | Slot providers: Ranger, Archivist, Sniper |
| `.strategist/knowledge.index.yaml` | Fontes de conhecimento por task_type |
| `.analysis/` | Artefatos de missão (pending, refined, done) |

---

### Instalação local (sem curl / clone repo)

```bash
# silent: defaults pragmatic-standalone
bash strategist/install.sh

# wizard interativo
bash strategist/install.sh --wizard

# repositório alvo customizado
bash strategist/install.sh --target /path/to/repo
```

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
| [strategist-mission-pipeline/design.md](strategist-mission-pipeline/design.md) | Decisões de design e rationale da implementação |
