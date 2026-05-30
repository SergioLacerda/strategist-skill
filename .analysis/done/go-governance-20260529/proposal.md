# Proposta: Go como Linguagem de Governança + Skill Compilada
**Mission ID:** go-governance-20260529  
**Date:** 2026-05-29  
**Status:** plan_only

---

## Resposta Direta

**Sim, impacta significativamente — mas de forma cirúrgica.**

A camada comportamental do agente (SKILL.md, personas, roles, contratos, pipeline de missão) fica **intacta**. O impacto é 100% na camada de infraestrutura: os shell scripts que o agente usa como ferramentas viram um binário Go.

---

## Mapa de Impacto

```
ALTO IMPACTO (reescrita)         BAIXO IMPACTO (ajuste)    ZERO IMPACTO (intacto)
────────────────────────         ──────────────────────    ──────────────────────
strategist/scripts/*.sh    →     SKILL.md                  personas/*.yaml
strategist/install.sh      →     bootstrap.sh              roles/*.yaml
strategist/tests/*.sh      →     README                    contracts/*.yaml
                           →     CI release.yml            schemas/*.yaml
                                                           .analysis/ workspace
                                                           mission pipeline
                                                           approval gate
                                                           forbidden behaviors
```

---

## O que é a "skill compilada"

O binário `strategist` substitui os shell scripts. O agente, em vez de chamar:

```bash
sh .strategist/scripts/check-stale.sh .compiled/.config.gz
```

chama:

```bash
strategist check-stale .compiled/.config.gz
```

O binário pode conter os arquivos padrão embutidos via `//go:embed`:

```go
//go:embed defaults/SKILL.md defaults/personas/ defaults/roles/ defaults/contracts/
var defaultsFS embed.FS
```

Assim, `strategist install` extrai tudo do próprio binário — sem precisar baixar YAML separadamente. O cliente recebe **um único binário** que contém toda a skill.

---

## Estrutura do repositório após migração

```
strategist-skill/
├── cmd/
│   └── strategist/
│       ├── main.go
│       ├── install.go       ← install.sh reescrito
│       ├── compile.go       ← compile-all.sh reescrito
│       └── check_stale.go   ← check-stale.sh reescrito
├── pkg/
│   ├── compile/
│   │   ├── config.go        ← compile-config.sh
│   │   ├── domain.go        ← compile-domain.sh
│   │   ├── index.go         ← compile-knowledge-index.sh
│   │   └── all.go           ← compile-all.sh
│   ├── stale/
│   │   └── check.go         ← check-stale.sh
│   └── install/
│       ├── installer.go     ← install.sh (silent + wizard)
│       └── wizard.go        ← modo interativo
├── defaults/                ← embutidos no binário via go:embed
│   ├── SKILL.md
│   ├── personas/
│   ├── roles/
│   ├── contracts/
│   └── schemas/
├── tests/                   ← go test ./...
│   ├── unit/
│   ├── integration/
│   └── fixtures/
├── go.mod
├── go.sum
├── bootstrap.sh             ← ajustado: baixa binário em vez de tarball
├── bootstrap.ps1            ← simplificado ou descontinuado
├── .goreleaser.yaml         ← novo
└── strategist/              ← mantido apenas para YAML de config runtime
    ├── active.yaml          ← template para install
    ├── knowledge.index.yaml
    └── skill.yaml           ← contrato da skill
```

---

## CLI pós-migração

```
strategist install [--silent | --wizard] [--target <path>]
strategist compile [--root <path>]
strategist check-stale <artifact.gz>
strategist version
```

---

## Dependências Go

```go
// go.mod — dependências mínimas
require (
    gopkg.in/yaml.v3                        // YAML parsing
    github.com/spf13/cobra                  // CLI
    // opcional para wizard interativo:
    github.com/charmbracelet/bubbletea      // TUI (se quiser wizard bonito)
    // ou:
    github.com/AlecAivazis/survey/v2        // prompts simples
)
```

---

## Distribuição

```yaml
# .goreleaser.yaml
builds:
  - goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    
archives:
  - format: binary   # binário direto, sem tarball
  
checksum:
  name_template: SHA256SUMS
  
release:
  github: true
```

`bootstrap.sh` atualizado:
```bash
# em vez de: curl .../archive.tar.gz | tar -xz
curl -L "https://github.com/SergioLacerda/strategist-skill/releases/download/${VERSION}/strategist-linux-amd64" \
  -o strategist
sha256sum --check SHA256SUMS  # já planejado em seguranca-testes Bloco A
chmod +x strategist
./strategist install
```

---

## Timeline por fase

| Fase | Entregável | Esforço |
|------|-----------|---------|
| 1 | go.mod, estrutura de pacotes, CI `go build` verde | 1 dia |
| 2 | `pkg/stale` + `pkg/compile/*` | 4–5 dias |
| 3 | `cmd/strategist` — subcommands check-stale, compile | 2 dias |
| 4 | `pkg/install` + `cmd/strategist install` (maior fase) | 4–5 dias |
| 5 | `//go:embed` defaults no binário | 1–2 dias |
| 6 | `go test ./...` — validators + unit + integration | 2 dias |
| 7 | goreleaser + CI multi-platform | 1–2 dias |
| 8 | `bootstrap.sh` + `SKILL.md` + README | 1 dia |
| **Total** | | **3–4 semanas** |

**Fase crítica:** pkg/install (reescrever `install.sh` com wizard interativo) — é onde a maior parte do esforço se concentra.

---

## Ordem recomendada de missões Strategist

1. **Setup Go** — go.mod, estrutura, CI build básico
2. **pkg/compile** — os 5 scripts de compilação (independentes, testáveis)
3. **pkg/stale** — check-stale (simples, boa entry point)
4. **cmd check + compile** — CLI funcional sem install
5. **pkg/install** — o coração; fazer em sub-missões (silent vs wizard separados)
6. **go embed** — embutir defaults
7. **go tests** — migrar suite de testes
8. **goreleaser** — distribuição
9. **SKILL.md + bootstrap** — ajustes finais
