# Discovery: Impacto de Migrar para Go como Linguagem de Governança
**Mission ID:** go-governance-20260529  
**Date:** 2026-05-29  
**Contexto:** decisão de adotar Estratégia B + modelo de skill compilada

---

## O que "skill compilada" significa neste projeto

Atualmente, ao instalar o Strategist, o cliente recebe:
- Um tarball com scripts shell + YAML
- `install.sh` copia arquivos para `.strategist/`
- O agente (Claude) lê SKILL.md e chama `sh .strategist/scripts/check-stale.sh`

Com Go, "skill compilada" significa:
- Um binário `strategist` é o artefato de distribuição
- O binário CONTÉM a lógica compilada de: check-stale, compile-config, compile-domain, compile-all, install
- O agente chama `strategist check-stale`, `strategist compile`, `strategist install`
- Os arquivos YAML de config (active.yaml, personas, roles, contracts) continuam externos
- SKILL.md continua externo — o agente ainda o lê para suas instruções comportamentais

**Extensão possível (Go embed):** o binário pode conter os defaults embutidos via `//go:embed`:
- Personas padrão, roles padrão, contratos, schemas, SKILL.md
- `strategist install` extrairia tudo do próprio binário — 100% self-contained
- A config do usuário (`active.yaml`, customizações) continuaria em `.strategist/`

---

## Inventário: o que muda vs. o que fica

### Substitutos diretos (shell → Go package)

| Script atual | Go equivalente | Linhas shell → Go |
|-------------|---------------|------------------|
| `scripts/check-stale.sh` | `pkg/stale/check.go` | 60sh → ~80go |
| `scripts/compile-config.sh` | `pkg/compile/config.go` | 69sh → ~120go |
| `scripts/compile-domain.sh` | `pkg/compile/domain.go` | 75sh → ~110go |
| `scripts/compile-knowledge-index.sh` | `pkg/compile/index.go` | 47sh → ~100go |
| `scripts/compile-all.sh` | `pkg/compile/all.go` | ~60sh → ~80go |
| `install.sh` (348 linhas) | `cmd/strategist/install.go` + `pkg/install/` | maior reescrita |
| `bootstrap.sh` (131 linhas) | atualizado para baixar binário | parcial |
| Suite de testes shell (510 linhas) | `go test ./...` | substituto direto |

**Total shell a reescrever: ~764 linhas → ~700-900 linhas Go** (mais verboso, mais robusto)

### Fica igual (zero impacto na lógica do agente)

| Componente | Status |
|-----------|--------|
| SKILL.md — pipeline, approval gate, forbidden behaviors | **intacto** |
| personas/*.yaml, roles/*.yaml, contracts/*.yaml | **intacto** |
| schemas/*.yaml, active.yaml | **intacto** |
| `.analysis/` workspace (pending, refined, done) | **intacto** |
| Missão pipeline (Ranger → Archivist → Sniper) | **intacto** |
| BDD specs (.feature) | **intacto** |
| SDD integration model | **intacto** |

### Muda levemente (ajuste, não reescrita)

| Componente | Mudança |
|-----------|---------|
| SKILL.md — referências a shell | `sh .strategist/scripts/check-stale.sh` → `strategist check-stale` |
| `bootstrap.sh` | baixa binário em vez de tarball |
| README — instalação | atualiza comandos de instalação |
| CI `release.yml` | cross-platform Go builds (goreleaser) |

---

## Impacto por camada

### Camada de infraestrutura — ALTO impacto

Tudo que hoje é shell script para reescrever em Go.
`install.sh` é o maior desafio: 348 linhas com lógica de wizard (perguntas interativas, validação de providers, gitignore, shims de agente). Em Go, isso vira um CLI Cobra com subcommands.

### Camada de distribuição — IMPACTO FUNDAMENTAL

Hoje: `bootstrap.sh` baixa tarball → extrai → `install.sh`  
Com Go: `bootstrap.sh` baixa binário pré-compilado por plataforma

Requer:
- **goreleaser** (standard para Go binary distribution)
- Builds para: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
- SHA256SUMS assinados — já planejado em seguranca-testes (Bloco A)
- Binário: ~8–15 MB vs tarball atual (~50 KB de text files)

### Camada de CI — SIGNIFICATIVO

| Atual | Com Go |
|-------|--------|
| shellcheck | `go build ./...` |
| bash tests | `go test ./...` |
| nenhum build multiplataforma | goreleaser matrix (5 plataformas) |
| release: sh + zip | release: goreleaser |

### Camada comportamental do agente — BAIXO impacto

O que o agente faz (fases, aprovações, orquestração) não muda. Apenas as invocações de ferramenta mudam de `sh script.sh` para `strategist <command>`. SKILL.md recebe edições pontuais, não reescrita.

---

## Benefícios específicos de Go sobre shell (Estratégia B)

1. **Windows nativo sem PS1/WSL** — binário Go é cross-platform de verdade
2. **Zero dependência runtime de jq/yq/python3** — tudo está no binário
3. **Testes sem subprocess** — funções Go testadas diretamente, sem `os/exec`
4. **Tipagem** — erros de schema em YAML detectados em compile time
5. **`go install`** — contribuidores podem instalar direto do source
6. **goreleaser** — Homebrew tap, Scoop (Windows), Docker image opcionais

---

## Timeline realista (Estratégia B completa)

| Fase | Escopo | Esforço |
|------|--------|---------|
| 1. Setup módulo Go | go.mod, estrutura de pacotes, CI build | 1 dia |
| 2. pkg/stale + pkg/compile | 5 scripts → packages Go | 4–5 dias |
| 3. cmd/strategist — check e compile | CLI Cobra básico | 2 dias |
| 4. pkg/install + cmd/strategist install | maior fase: reescrever install.sh | 4–5 dias |
| 5. go embed (defaults) | embutir YAML + SKILL.md no binário | 1–2 dias |
| 6. Go tests (Strategy A) | validators + unit + integration | 2 dias |
| 7. goreleaser + CI | multi-platform build, release workflow | 1–2 dias |
| 8. bootstrap.sh update | baixar binário em vez de tarball | 0,5 dia |
| 9. SKILL.md update | ajustar referências de comandos | 0,5 dia |
| **Total** | | **3–4 semanas** |

A maior fase é pkg/install (reescrever install.sh com wizard interativo em Go).

---

## Riscos

1. **install.sh wizard é complexo** — 348 linhas com readline, validação de providers, git, shims. Em Go, readline interativo requer `github.com/charmbracelet/bubbletea` ou `survey` — adiciona dependência e complexidade.

2. **bootstrap.sh ainda é shell** — o downloader inicial continua sendo shell (inevitável). O elo "curl | sh" não desaparece; muda o que é baixado.

3. **Tamanho do binário** — 8–15 MB vs ~50 KB atual. Não é problema técnico, mas é uma mudança de percepção.

4. **Contribuição** — hoje qualquer dev com bash contribui. Com Go, a barra de entrada sobe.
