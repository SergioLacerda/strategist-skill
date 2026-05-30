# ADR-0002 — Defaults embutidos no binário via embed.FS

**Status:** Accepted  
**Data:** 2026-05-29  
**Contexto:** Migração para Go (bigbang-go-20260529)

---

## Contexto

O comando `strategist install` precisa copiar ~60 arquivos (SKILL.md, personas, roles, schemas, contratos, templates) para o diretório `.strategist/` do repositório-alvo. Esses arquivos precisam estar disponíveis em qualquer ambiente onde o binário é executado.

Alternativas consideradas:
- **Fetch da rede** — baixar os arquivos do GitHub no momento do install
- **Bundling externo** — distribuir um tarball junto com o binário
- **embed.FS** — embutir os arquivos no próprio binário em tempo de compilação

## Decisão

Embutir todos os defaults em `internal/embed/defaults/` usando `//go:embed all:defaults`. O pacote `embed.Extractor` implementa `domain.FileExtractor` e copia a FS embutida para disco via `fs.WalkDir`, preservando a estrutura de diretórios.

```go
//go:embed all:defaults
var defaultsFS embed.FS
```

`internal/embed/defaults/` é uma cópia exata de `strategist/` — qualquer mudança no runtime da skill precisa ser refletida em ambos.

## Consequências

**Positivas:**
- `strategist install` funciona **offline** e sem dependências externas (sem curl, jq, git)
- Bootstrap via `curl | bash` baixa um único binário self-contained — sem assets separados
- Sem falhas de rede silenciosas no momento do install
- Versão dos defaults está fixada à versão do binário — sem drift entre binário e runtime

**Negativas:**
- `strategist/` e `internal/embed/defaults/` precisam ser mantidos em sincronia — drift é detectável via diff, mas não automaticamente bloqueado em CI
- Binário cresce com os defaults embutidos (~alguns KB de YAML comprimido)
- Editar defaults requer recompilar o binário — não é possível atualizar só os arquivos YAML em produção sem um novo release
