# Proposta: Big Bang — Migração Total para Go
**Mission ID:** bigbang-go-20260529  
**Date:** 2026-05-29 (atualizado 2026-05-29 — rev2: engineering standards)  
**Status:** done — implementado via commit 3561d6e
**Constraint:** sem retrocompatibilidade, sem prazo

---

## Premissas do Big Bang

1. **Dia 1:** `git rm` todos os shell scripts — o repo para de funcionar com shell
2. **Sem migration path:** instalações existentes quebram até reinstalarem com o binário Go
3. **Sem shims:** nenhum wrapper shell que chama Go temporariamente
4. **Bootstrap reescrito do zero:** 30 linhas, sem legado
5. **`bootstrap.ps1` descontinuado:** Go resolve Windows nativamente

---

## O que o binário `strategist` entrega

Um único binário que substitui 764 linhas de shell:

```
strategist install [--silent | --wizard] [--target <path>]
strategist compile  [--root <path>]
strategist check-stale <artifact.gz>
strategist version
```

O binário contém embutidos (via `//go:embed`) todos os arquivos padrão da skill:
SKILL.md, personas, roles, contracts, schemas, templates, sub-skills. Ao rodar `strategist install`, o cliente recebe a skill completa sem precisar de nenhum arquivo YAML separado.

---

## Visão do repositório pós-big-bang

```
strategist-skill/
├── cmd/strategist/          ← CLI (Cobra)
│   ├── main.go
│   ├── root.go
│   ├── install.go
│   ├── compile.go
│   └── check_stale.go
├── pkg/
│   ├── compile/             ← 5 scripts → packages Go
│   ├── stale/               ← check-stale.sh → Go
│   ├── install/             ← install.sh → Go (maior pacote)
│   └── embed/               ← //go:embed defaults/**
├── defaults/                ← source of truth para install
│   ├── SKILL.md
│   ├── personas/
│   ├── roles/
│   ├── contracts/
│   ├── schemas/
│   ├── templates/
│   ├── skills/
│   └── ...
├── tests/                   ← go test ./...
│   ├── *_test.go
│   └── fixtures/*.yaml      ← mantidos
├── go.mod
├── .goreleaser.yaml
├── bootstrap.sh             ← 30 linhas: baixa binário + verifica SHA
├── Makefile
└── readme.md
```

**Removidos completamente:**
- `strategist/scripts/` (5 shell scripts)
- `strategist/install.sh`
- `bootstrap.ps1`
- `strategist/tests/*.sh`
- `strategist/tests/harness/`
- `strategist/tests/validators/*.sh`
- `strategist/tests/unit/*.sh`
- `strategist/tests/integration/*.sh`

**Movidos para `defaults/`:**
- Todo o conteúdo de `strategist/` que não é shell

---

## Impacto em SKILL.md (4 linhas)

Apenas as referências a `sh .strategist/scripts/` mudam. O resto — aprovação, forbidden behaviors, pipeline, drift correction — é intocável.

```diff
- sh .strategist/scripts/check-stale.sh <artifact>
+ strategist check-stale <artifact>

- sh .strategist/scripts/compile-all.sh .strategist .strategist/knowledge.index.yaml
+ strategist compile --root .strategist
```

---

## Padrões de Engenharia (adicionados rev2)

### Clean Architecture
- `internal/domain/` como camada central: tipos + interfaces (Ports), zero imports externos
- Dependências sempre apontam para `domain`; nunca o inverso
- `cmd/` injeta dependências concretas; `internal/` packages nunca importam `cmd/`

### Guardrails de qualidade
- `.golangci.yaml` com: errcheck, gosec, staticcheck, govet, revive, wrapcheck, exhaustive, gocognit (max 15)
- `golangci-lint run ./...` bloqueia CI — nenhum merge sem lint verde
- `//nolint` só com comentário justificando

### Testes Go idiomáticos
- 100% table-driven com `t.Run(tt.name, ...)`
- `t.TempDir()` em todos os testes que precisam de filesystem
- `go test -race ./...` — race detector sempre habilitado em CI
- Cobertura mínima 80% em `internal/compile` e `internal/stale`
- Blackbox tests (`package X_test`) para APIs públicas

### Sniper como referência
O Sniper opera sob governança world-class por construção (forbidden behaviors, contratos, zero side effects não autorizados). Em caso de dúvida sobre decisão de engenharia Go, consultar o Sniper offline — ele aplica os mesmos padrões esperados no código.

## Dependências Go

```
github.com/spf13/cobra              — CLI
gopkg.in/yaml.v3                    — YAML parsing
github.com/charmbracelet/huh        — wizard interativo (opcional; fallback: stdin simples)
github.com/stretchr/testify         — assertions em testes (require + assert)
```

Sem dependências para lógica core (compile, stale, embed): só stdlib Go.

---

## Distribuição

- 5 plataformas via goreleaser: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
- SHA256SUMS por release
- Binário: ~8–12 MB (com defaults embedded)
- `bootstrap.sh`: deteta OS/arch, baixa binário, verifica SHA, executa `strategist install`

---

## O que não muda para o usuário final

```
/strategist <prompt>    ← invocação igual
approval gate           ← igual
missões, artefatos      ← igual
.analysis/ workspace    ← igual
active.yaml, roles      ← igual
```

A experiência do usuário final é idêntica. O que muda é como a skill é instalada e como o agente acessa as ferramentas internas.
