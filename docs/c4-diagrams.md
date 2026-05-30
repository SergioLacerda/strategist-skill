# Diagramas C4 — Strategist Skill

Documentação arquitetural em 4 níveis do modelo C4. Renderizado pelo GitHub via Mermaid.

---

## Nível 1 — Contexto do Sistema

Mostra o Strategist Skill no ecossistema: quem usa o sistema e como ele se relaciona com sistemas externos.

```mermaid
C4Context
    title Nível 1 — Contexto do Sistema

    Person(dev, "Desenvolvedor", "Cria e invoca missões de software via CLI e chat")

    System(strategist, "Strategist Skill", "Orquestra missões via discovery → refinamento → execução com approval gate obrigatório")

    System_Ext(claude, "Claude Agent (LLM)", "Executa o runtime da skill via conversa")
    System_Ext(github, "GitHub", "Hospeda código-fonte, CI/CD e releases do binário")
    System_Ext(target_repo, "Target Repository", "Repositório onde a skill é instalada e onde os artefatos de missão são escritos")

    Rel(dev, claude, "Invoca missões via chat")
    Rel(dev, strategist, "Instala e configura via CLI (strategist install)")
    Rel(claude, strategist, "Carrega SKILL.md e executa o pipeline de missão")
    Rel(strategist, target_repo, "Lê/escreve artefatos em .analysis/")
    Rel(github, strategist, "bootstrap.sh baixa o binário de GitHub Releases")
    Rel(strategist, github, "CI publica releases e verifica SHA256")
```

---

## Nível 2 — Containers

Mostra os containers (unidades executáveis e de armazenamento) dentro do sistema Strategist.

```mermaid
C4Container
    title Nível 2 — Containers

    Person(dev, "Desenvolvedor", "")

    System_Boundary(sys, "Strategist Skill") {
        Container(binary, "strategist", "Go binary", "CLI: install, compile, check-stale, validate, version")
        Container(skill_root, ".strategist/", "YAML + gzip/JSON", "Configs (active.yaml, personas/, roles/), artefatos compilados (.compiled/), memória (memory/)")
        Container(shim, "~/.claude/skills/strategist/SKILL.md", "Markdown", "Registro da skill no Claude Agent — aponta para o skill root")
        Container(analysis, ".analysis/", "Markdown", "Artefatos de missão: pending/ (discovery), refined/ (planos), done/ (reports)")
    }

    System_Ext(claude, "Claude Agent (LLM)", "Executa o runtime da skill")
    System_Ext(github, "GitHub Releases", "Hospeda binário e bootstrap.sh")

    Rel(dev, binary, "Executa comandos CLI", "shell")
    Rel(dev, claude, "Invoca missões via chat", "linguagem natural")
    Rel(binary, skill_root, "Extrai defaults embutidos e gera .compiled/", "fs.WalkDir + gzip+JSON")
    Rel(binary, shim, "Instala o shim após install", "os.WriteFile")
    Rel(claude, shim, "Resolve o skill root via source: declarado no shim")
    Rel(claude, skill_root, "Lê SKILL.md, protocol.md e artefatos .compiled/", "fs read")
    Rel(claude, analysis, "Escreve artefatos de missão via slots", "Ranger/Archivist/Sniper")
    Rel(github, binary, "bootstrap.sh baixa e verifica SHA256", "curl + sha256sum")
```

---

## Nível 3a — Componentes do Binário Go

Detalha os pacotes internos do binário `strategist` e suas dependências.

```mermaid
C4Component
    title Nível 3a — Componentes do Binário Go

    Container_Boundary(bin, "strategist (Go binary)") {
        Component(cmd, "cmd/strategist", "Go · cobra", "6 comandos CLI. Recebe flags, constrói configs e delega para internal/")
        Component(domain, "internal/domain", "Go · interfaces", "Types centrais (InstallConfig, WizardConfig, CompiledConfig) e ports (Installer, Compiler, StaleChecker, FileExtractor)")
        Component(embed_pkg, "internal/embed", "Go · embed.FS", "Defaults embutidos no binário em tempo de compilação. Extractor copia a FS para disco via fs.WalkDir")
        Component(install_pkg, "internal/install", "Go", "Service.Install: extract → applyConfig → ensureGitignore → installShim → CompileAll. Rollback automático em falha")
        Component(compile_pkg, "internal/compile", "Go · gzip/JSON", "Compiler.CompileAll: gera .index.gz, .domain.gz, .config.gz e .manifest.gz a partir dos YAMLs")
        Component(stale_pkg, "internal/stale", "Go", "Checker.IsStale: abre o .gz, lê sources map, compara mtime de cada fonte com o valor registrado")
    }

    Rel(cmd, install_pkg, "install, install-global → Service.Install(InstallConfig)")
    Rel(cmd, compile_pkg, "compile → Compiler.CompileAll(root, indexPath)")
    Rel(cmd, stale_pkg, "check-stale → Checker.IsStale(artifactPath)")
    Rel(cmd, domain, "constrói InstallConfig, WizardConfig")
    Rel(install_pkg, embed_pkg, "Extractor.Extract(strategistDir) — copia defaults")
    Rel(install_pkg, compile_pkg, "CompileAll após extração — não-fatal em falha")
    Rel(install_pkg, domain, "implementa domain.Installer via serviceAdapter")
    Rel(compile_pkg, domain, "implementa domain.Compiler; produz CompiledConfig, CompiledDomain, CompiledIndex")
    Rel(stale_pkg, domain, "implementa domain.StaleChecker")
    Rel(embed_pkg, domain, "implementa domain.FileExtractor")
```

---

## Nível 3b — Pipeline do Runtime da Skill

Detalha as fases e sub-skills executadas pelo Claude Agent ao orquestrar uma missão.

```mermaid
flowchart TD
    subgraph bootstrap["⚙️ Bootstrap"]
        B1["LearningBuffer flush check\n(se outcomes.tmp ≥ 20 linhas)"]
        B2["Carrega .compiled/.config.gz\nou active.yaml + personas/ + roles/"]
        B1 --> B2
    end

    subgraph preflight["🔍 Preflight"]
        P1["Valida providers dos slots\n(risk_score por slot)"]
        P2["Carrega domain interno\n(.compiled/.domain.gz ou templates/domain/)"]
        P1 --> P2
    end

    subgraph intake["📋 Intake"]
        I1["prompt-intake\nClassifica task_type, risk_level\nextra restrições da missão"]
        I2["context-enrichment\nConsulta knowledge.index.yaml\npor task_type + source-hints"]
        I3["dossier-builder\nMonta dossier dentro do token budget\npara passar aos slot providers"]
        I1 --> I2 --> I3
    end

    subgraph discovery["🔭 Discovery — Ranger"]
        D1["Slot: discovery\nProvider configurável em roles/\nEscreve em pending/"]
        D2["housekeeping_scan\n(interno — sem slot)\nVarre pending/, refined/, done/\nproduz side_quest_manifest"]
        D1 --> D2
    end

    subgraph refinement["📐 Refinement — Archivist"]
        R1["Slot: refinement\nLê artefato de discovery\nProduz plano revisado em refined/"]
    end

    subgraph gate["🚦 Approval Gate (obrigatório)"]
        AG{"Usuário aprova\no plano refinado?"}
    end

    subgraph execution["⚡ Execution — Sniper"]
        E1["Side quests (se houver)\nnão-bloqueante — falha continua"]
        E2["Plano principal\nSlot: execution"]
        E1 --> E2
    end

    subgraph learning["📚 Learning (não-bloqueante)"]
        L1["response-critic\nAvalia output vs rubrica\ndo task_type"]
        L2["learning-curator\nPropõe entries para outcomes.jsonl\ne source-hints.yaml\n⚠️ requer aprovação do usuário"]
        L1 --> L2
    end

    RESULT(["✅ Resultado da missão\n.analysis/done/<id>-report.md"])
    PLAN_ONLY(["📄 Plan only\n.analysis/refined/<id>-plan.md"])

    bootstrap --> preflight --> intake --> discovery --> refinement --> gate
    gate -- "sim" --> execution --> learning --> RESULT
    gate -- "não" --> PLAN_ONLY

    style bootstrap fill:#1e2a3a,color:#ccc
    style preflight fill:#1e2a3a,color:#ccc
    style intake fill:#1e3a2a,color:#ccc
    style discovery fill:#2a1e3a,color:#ccc
    style refinement fill:#2a1e3a,color:#ccc
    style gate fill:#3a2a1e,color:#ccc
    style execution fill:#3a1e1e,color:#ccc
    style learning fill:#1e1e3a,color:#ccc
    style RESULT fill:#1e3a1e,color:#ccc
    style PLAN_ONLY fill:#2a2a2a,color:#ccc
```

---

## Referência rápida — Slots e Sub-skills

| Componente | Tipo | risk_score | Escreve em |
|------------|------|-----------|-----------|
| `prompt-intake` | sub-skill interna | `read_only` | — |
| `context-enrichment` | sub-skill interna | `read_only` | — |
| `dossier-builder` | sub-skill interna | `read_only` | — |
| Slot `discovery` (Ranger) | plugável | `write_pending` | `<base_path>/pending/` |
| `housekeeping_scan` | interno (sem slot) | — | — |
| Slot `refinement` (Archivist) | plugável | `write_analysis` | `<base_path>/refined/` |
| Slot `execution` (Sniper) | plugável | `controlled` | `<base_path>/done/` |
| `response-critic` | sub-skill interna | `read_only` | — |
| `learning-curator` | sub-skill interna | `read_only` | `memory/` (com aprovação) |

---

## Stop Conditions

O pipeline para imediatamente em qualquer destas condições:

| Código | Causa |
|--------|-------|
| `preflight_failed` | Qualquer check de preflight falhou |
| `slot_provider_not_found` | skill.yaml do provider não encontrado |
| `slot_risk_mismatch` | risk_score do provider incorreto para o slot |
| `intake_conflict_unresolved` | Dois aliases mutuamente exclusivos no prompt |
| `user_denies_execution` | Usuário recusou no approval gate (não é erro) |
| `discovery_failed` | Slot discovery não produziu artefato |
| `refinement_failed` | Slot refinement não produziu artefato |
| `slot_write_type_violation` | Slot tentou escrever tipo de arquivo não-`.md` |
| `slot_write_scope_violation` | Slot tentou escrever fora do escopo declarado |
