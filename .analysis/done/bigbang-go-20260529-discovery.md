# Discovery: Plano Big Bang — Migração Total para Go
**Mission ID:** bigbang-go-20260529  
**Date:** 2026-05-29  
**Constraint:** sem retrocompatibilidade, sem prazo, big bang completo

---

## O que "big bang" elimina do plano anterior

A abordagem incremental (8 fases) precisava manter shell scripts funcionando enquanto Go era construído. Big bang elimina esse custo:

- `strategist/scripts/*.sh` → **deletados no dia 1** (não mantidos em paralelo)
- `strategist/install.sh` → **deletado** (substituído por `strategist install`)
- `bootstrap.ps1` → **descontinuado** (binário Go resolve Windows nativamente)
- `bootstrap.sh` → **reescrito do zero** (downloader simples sem preocupação com versão atual)
- Todos os shell tests → **deletados** (substituídos por `go test ./...`)
- Sem shims de compatibilidade, sem migration paths, sem warnings de deprecation

---

## Inventário completo do repositório

### Deletar (substituído por Go)

```
strategist/install.sh              ← 348 linhas → cmd install
strategist/scripts/check-stale.sh  ← 60 linhas  → pkg/stale
strategist/scripts/compile-all.sh  ← 60 linhas  → pkg/compile
strategist/scripts/compile-config.sh ← 69 linhas → pkg/compile
strategist/scripts/compile-domain.sh ← 75 linhas → pkg/compile
strategist/scripts/compile-knowledge-index.sh ← 47 linhas → pkg/compile
bootstrap.ps1                       ← descontinuado
strategist/tests/*.sh               ← substituídos por go test
```

### Reescrever (mantido com novo conteúdo)

```
bootstrap.sh      ← downloader: deteta OS/arch, baixa binário, executa strategist install
```

### Mover para defaults/ (embedded no binário)

```
strategist/SKILL.md              → defaults/SKILL.md
strategist/personas/             → defaults/personas/
strategist/roles/                → defaults/roles/
strategist/contracts/            → defaults/contracts/
strategist/schemas/              → defaults/schemas/
strategist/templates/            → defaults/templates/
strategist/skills/               → defaults/skills/
strategist/knowledge.index.yaml  → defaults/knowledge.index.yaml
strategist/memory/               → defaults/memory/
strategist/skill.yaml            → defaults/skill.yaml
strategist/protocol.md           → defaults/protocol.md
```

### Manter como estão (config runtime — NÃO embedded)

```
.strategist/active.yaml          ← gerado por install, editado pelo usuário
.strategist/roles/default.yaml   ← idem
.strategist/knowledge.index.yaml ← customizável pelo usuário
.strategist/memory/              ← runtime: outcomes.tmp, outcomes.jsonl
.strategist/.compiled/           ← gerado por strategist compile
```

---

## Estrutura Go proposta

```
strategist-skill/
├── cmd/
│   └── strategist/
│       ├── main.go
│       ├── root.go              ← cobra root command
│       ├── install.go           ← strategist install [--silent|--wizard] [--target]
│       ├── compile.go           ← strategist compile [--root]
│       └── check_stale.go       ← strategist check-stale <artifact.gz>
│
├── pkg/
│   ├── compile/
│   │   ├── config.go            ← compile-config.sh
│   │   ├── domain.go            ← compile-domain.sh
│   │   ├── index.go             ← compile-knowledge-index.sh
│   │   ├── all.go               ← compile-all.sh (orquestra os 3)
│   │   └── manifest.go          ← escreve .manifest.gz com SHA256
│   ├── stale/
│   │   └── check.go             ← check-stale.sh
│   ├── install/
│   │   ├── installer.go         ← copia defaults/ para .strategist/
│   │   ├── wizard.go            ← modo interativo
│   │   ├── gitignore.go         ← adiciona .strategist/.compiled/ ao .gitignore
│   │   └── shim.go              ← cria .claude/skills/strategist/skill.yaml
│   └── embed/
│       └── defaults.go          ← //go:embed defaults/**
│
├── defaults/                    ← embedded no binário
│   ├── SKILL.md
│   ├── skill.yaml
│   ├── protocol.md
│   ├── knowledge.index.yaml
│   ├── personas/
│   │   ├── pragmatic.yaml
│   │   └── epic.yaml
│   ├── roles/
│   │   ├── default.yaml
│   │   ├── mission.yaml
│   │   └── spec-driven.yaml
│   ├── contracts/               ← todos os 10 contracts
│   ├── schemas/
│   ├── templates/               ← domain templates
│   ├── skills/                  ← sub-skills (archivist, etc.)
│   └── memory/
│       └── source-hints.yaml
│
├── tests/
│   ├── compile_test.go
│   ├── stale_test.go
│   ├── install_test.go
│   ├── fixtures/                ← YAML fixtures (mantidos)
│   └── specs/                   ← BDD specs (mantidos)
│
├── go.mod
├── go.sum
├── .goreleaser.yaml
├── bootstrap.sh                 ← reescrito: baixa binário
├── Makefile                     ← build, test, release
└── readme.md
```

---

## O que muda em `.strategist/` no cliente

Antes (hoje):
```
.strategist/
├── scripts/          ← shell scripts copiados pelo install.sh
├── SKILL.md
├── active.yaml
├── personas/
├── ...
```

Depois (Go):
```
.strategist/
├── SKILL.md          ← extraído do binário pelo strategist install
├── active.yaml       ← gerado pelo wizard
├── personas/         ← extraídos do binário
├── roles/
├── contracts/
├── schemas/
├── templates/
├── skills/
├── .compiled/        ← gerado por strategist compile (gitignored)
└── memory/
```

Sem `scripts/`. O agente chama `strategist <cmd>` diretamente.  
**Pressuposto:** `strategist` está no PATH do ambiente onde Claude Code executa.

---

## SKILL.md — únicas linhas que mudam

```diff
# §1 Bootstrap fast path
- sh .strategist/scripts/check-stale.sh .strategist/.compiled/.config.gz
+ strategist check-stale .strategist/.compiled/.config.gz

# §2a Preflight fast path
- sh .strategist/scripts/check-stale.sh .strategist/.compiled/.domain.gz
+ strategist check-stale .strategist/.compiled/.domain.gz

# §4 Context Enrichment fast path
- sh .strategist/scripts/check-stale.sh .strategist/.compiled/.index.gz
+ strategist check-stale .strategist/.compiled/.index.gz

# Compile (install / re-compile)
- sh .strategist/scripts/compile-all.sh .strategist .strategist/knowledge.index.yaml
+ strategist compile --root .strategist
```

Todo o resto de SKILL.md (approval gate, forbidden behaviors, pipeline, etc.) fica intacto.

---

## bootstrap.sh reescrito (big bang simplification)

Sem preocupação com retrocompat, o novo bootstrap.sh pode ser mínimo:

```bash
#!/usr/bin/env bash
set -euo pipefail

REPO="SergioLacerda/strategist-skill"
VERSION="${STRATEGIST_VERSION:-latest}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
BIN_NAME="strategist-${OS}-${ARCH}"

# Resolver versão
if [[ "$VERSION" == "latest" ]]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
fi

BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

# Download + verify
curl -fsSL "${BASE_URL}/${BIN_NAME}" -o /tmp/strategist
curl -fsSL "${BASE_URL}/SHA256SUMS" -o /tmp/SHA256SUMS
(cd /tmp && sha256sum --check --ignore-missing SHA256SUMS)

# Install binary
install -m 755 /tmp/strategist ~/.local/bin/strategist  # ou /usr/local/bin

# Install skill
strategist install
```

65 linhas → ~30 linhas. Sem resolve_ref complexo, sem tarball, sem extração.

---

## goreleaser config

```yaml
# .goreleaser.yaml
version: 2

builds:
  - id: strategist
    main: ./cmd/strategist
    binary: strategist
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    env: [CGO_ENABLED=0]
    ldflags: ["-s -w -X main.version={{.Version}}"]

archives:
  - format: binary   # binário direto, sem tarball

checksum:
  name_template: SHA256SUMS
  algorithm: sha256

release:
  github:
    owner: SergioLacerda
    name: strategist-skill
```

---

## Decisão sobre PATH do binário

**Opção 1 — PATH global** (`/usr/local/bin` ou `~/.local/bin`):
- Simples para SKILL.md: `strategist check-stale`
- Requer que install coloque binário em PATH

**Opção 2 — PATH local ao projeto** (`.strategist/bin/strategist`):
- Sem poluir PATH do sistema
- SKILL.md chama `.strategist/bin/strategist check-stale`
- Mais isolado, mas path mais longo

**Recomendação:** Opção 1 para developer experience. `strategist install` coloca o binário em `~/.local/bin` e sugere adicionar ao PATH se não estiver.

---

## Complexidade de implementação por pacote

| Pacote | Complexidade | Motivo |
|--------|-------------|--------|
| `pkg/stale` | Baixa | Lógica simples: mtime + manifest |
| `pkg/compile/config` | Média | YAML merge + gzip write |
| `pkg/compile/domain` | Média | YAML + index parsing |
| `pkg/compile/index` | Média | Tag index building |
| `pkg/compile/all` | Baixa | Orquestra os 3, escreve manifest |
| `pkg/embed` | Baixa | `//go:embed` + extract |
| `pkg/install` (silent) | Média | Copy embed → .strategist/ |
| `pkg/install` (wizard) | Alta | Interactive prompts, validation |
| `cmd/*` | Baixa | Cobra wrappers |
| `tests/*` | Média | go test, fixtures |

**Maior complexidade:** wizard interativo. Biblioteca recomendada: `github.com/charmbracelet/huh` (modern, no heavy deps).
