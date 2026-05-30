# Arquitetura — Strategist Skill

## Visão Geral

O projeto é composto por duas camadas independentes:

| Camada | Localização | Responsabilidade |
|--------|-------------|------------------|
| **Binário Go** | `cmd/` + `internal/` | Instalar, compilar e validar artefatos da skill |
| **Runtime da Skill** | `strategist/` | Instruções ao agente: pipeline, slots, personas, contratos |

O binário **não executa missões**. Ele prepara o ambiente para que o agente (Claude) execute a skill corretamente.

---

## Mapa de Pacotes Go

```
cmd/strategist/          Comandos CLI (cobra)
  main.go                Entrypoint; chama execute()
  root.go                Registra todos os subcomandos
  install.go             strategist install
  install_global.go      strategist install-global
  compile.go             strategist compile
  check_stale.go         strategist check-stale
  validate.go            strategist validate
  version.go             strategist version

internal/
  domain/                Tipos centrais e interfaces (ports)
    types.go             CompiledConfig, CompiledDomain, CompiledIndex,
                         CompiledManifest, InstallConfig, WizardConfig
    ports.go             Interfaces: Installer, Compiler, StaleChecker,
                         FileExtractor
    errors.go            Sentinel errors do domínio

  embed/                 Defaults embutidos no binário
    defaults.go          embed.FS com todos os arquivos de defaults/
    defaults/            Cópia completa do strategist/ (SKILL.md, roles,
                         personas, schemas, contratos, templates)

  install/               Lógica de instalação
    installer.go         Service.Install — orquestra extract → config → gitignore → shim → compile
    wizard.go            Wizard interativo (coleta mode, base_path, provider)
    active_yaml.go       Geração de active.yaml a partir de WizardConfig
    template.go          Cópia de template para active.yaml (modo silent)
    gitignore.go         ensureGitignore — adiciona .strategist/.compiled/ ao .gitignore
    shim.go              Instala ~/.claude/skills/strategist/SKILL.md

  compile/               Compilação de artefatos YAML → gzip+JSON
    all.go               Compiler.CompileAll — orquestra index → domain → config → manifest
    config.go            Config() — active.yaml + personas/ + roles/ → .config.gz
    domain.go            Domain() — templates/domain/ → .domain.gz
    index.go             Index() — knowledge.index.yaml → .index.gz
    helpers.go           writeGzJSON, loadYAMLFile, mtime, sha256Artifact
    yaml.go              Helpers YAML internos

  stale/                 Detecção de artefatos obsoletos
    check.go             Checker.IsStale — compara mtime das fontes com o registrado no artefato

  testutil/              Helpers compartilhados para testes
    testutil.go          MinimalRoot, fixtures de diretório temporário
```

---

## Fluxo de Instalação

```
strategist install [--wizard] [--target=<dir>]
        │
        ▼
embed.Extractor.Extract(strategistDir)
  └─ copia internal/embed/defaults/* → <target>/.strategist/
        │
        ▼
applyConfig(strategistDir, cfg)
  ├─ [silent] copyTemplate("templates/pragmatic-standalone.yaml") → active.yaml
  └─ [wizard] runWizard(stdin) → writeActiveYAML(strategistDir, wc)
        │
        ▼
ensureGitignore(target)
  └─ adiciona ".strategist/.compiled/" ao .gitignore (cria se não existir)
        │
        ▼
installShim(target)
  └─ escreve ~/.claude/skills/strategist/SKILL.md
        │
        ▼
compile.Compiler.CompileAll(.strategist/, knowledge.index.yaml)
  └─ gera .strategist/.compiled/{.index.gz, .domain.gz, .config.gz, .manifest.gz}
```

**Rollback automático:** se qualquer etapa falhar, `Install` remove em ordem reversa todos os arquivos criados (`manifest []string`). Diretórios não-vazios são deixados intactos.

---

## Pipeline de Compilação

`CompileAll` produz 4 artefatos em `.strategist/.compiled/`:

| Artefato | Função | Fontes |
|----------|--------|--------|
| `.index.gz` | Knowledge index compilado | `knowledge.index.yaml` |
| `.domain.gz` | Domain templates compilados | `templates/domain/**/*.yaml` |
| `.config.gz` | Configuração compilada | `active.yaml` + `personas/*.yaml` + `roles/*.yaml` |
| `.manifest.gz` | Hashes dos 3 artefatos acima | gerado por `CompileAll` |

Cada artefato é **gzip + JSON**. O schema JSON inclui:
- `schema` — identificador de versão do formato (ex: `strategist-compiled-config/1.0`)
- `compiled_at` — Unix timestamp da compilação
- `sources` — mapa `path → mtime` das fontes usadas

O campo `sources` é o que permite ao `Checker.IsStale` detectar se alguma fonte foi modificada após a compilação.

---

## Detecção de Staleness

`stale.Checker.IsStale(artifactPath)` retorna `true` quando:

1. O arquivo do artefato não existe
2. `.manifest.gz` não existe no mesmo diretório
3. Qualquer fonte em `artifact.sources` não existe mais no disco
4. Qualquer fonte tem `mtime` mais recente que o valor registrado

Retorna `false` (fresco) somente quando todas as fontes existem e seus mtimes são ≤ ao registrado.

O CLI `check-stale` sai com código `0` se fresco e `1` se stale — projetado para uso em scripts de CI.

---

## Defaults Embutidos

`internal/embed/defaults/` é uma cópia exata de `strategist/` incluída no binário via `//go:embed all:defaults`. Isso significa que `strategist install` funciona **sem conexão de rede** e **sem o repositório clonado** — o binário carrega todos os defaults na memória.

A extração preserva a estrutura de diretórios, mas não sobrescreve arquivos pré-existentes (os arquivos são escritos via `os.WriteFile` diretamente — projetos com `.strategist/` personalizado devem fazer backup antes de re-instalar).

---

## Interfaces do Domínio (`internal/domain/ports.go`)

| Interface | Método | Implementada por |
|-----------|--------|------------------|
| `Installer` | `Install(InstallConfig) error` | `install.serviceAdapter` |
| `Compiler` | `CompileAll(root, indexPath string) error` | `compile.Compiler` |
| `StaleChecker` | `IsStale(artifactPath string) (bool, error)` | `stale.Checker` |
| `FileExtractor` | `Extract(targetDir string) error` | `embed.Extractor` |

As interfaces são satisfeitas via verificação em tempo de compilação (`var _ domain.X = Y{}`), garantindo que nenhuma implementação diverge silenciosamente.

---

## Testes

| Suite | Pacote | Abordagem |
|-------|--------|-----------|
| `stale_test.go` | `internal/stale` | 5 casos: ausente, sem manifest, fresco, fonte stale, fonte removida |
| `compile_test.go` | `internal/compile` | Config, Domain, Index, All (4 artefatos + manifest) |
| `install_test.go` | `internal/install` | Silent mode, gitignore, rollback |
| `installer_whitebox_test.go` | `internal/install` | `ensureGitignore`, propagação de erros |
| `fixtures_test.go` | `tests/` | Formato dos 5 fixtures de invariantes de segurança |
| `cmd_test.go` | `cmd/strategist` | Todos os comandos CLI |

Race detector ativo em todos os testes (`go test -race ./...`). Gate de cobertura: 90% por pacote interno (`make cover-gate`).
