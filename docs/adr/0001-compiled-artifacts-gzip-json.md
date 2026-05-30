# ADR-0001 — Artefatos compilados em gzip+JSON com fast path

**Status:** Accepted  
**Data:** 2026-05-29  
**Contexto:** Migração para Go (bigbang-go-20260529)

---

## Contexto

O agente precisa carregar a configuração da skill (active.yaml, personas, roles, domain templates) no início de **cada missão**. Com arquivos YAML separados, esse carregamento envolve múltiplas leituras de disco, parse de YAML e merge de estruturas — operação repetida a cada invocação.

A alternativa mais simples seria ler os YAMLs diretamente sempre, sem pré-processamento.

## Decisão

Compilar as fontes YAML em artefatos **gzip+JSON** armazenados em `.strategist/.compiled/`:

| Artefato | Fontes |
|----------|--------|
| `.config.gz` | `active.yaml` + `personas/` + `roles/` |
| `.domain.gz` | `templates/domain/` |
| `.index.gz` | `knowledge.index.yaml` |
| `.manifest.gz` | SHA256 dos 3 artefatos acima |

Cada artefato inclui um campo `sources: map[path → mtime]` que permite detectar staleness sem recompilar.

O agente implementa um **fast path**: se o artefato existe e não está stale (`check-stale`), faz `gunzip + JSON parse` em vez de ler e parsear múltiplos YAMLs. O **standard path** (ler YAMLs direto) funciona como fallback em caso de corrupção.

## Consequências

**Positivas:**
- Carregamento de configuração em uma única operação (decompress + decode JSON)
- Artefato corrompido tem fallback automático para standard path — sem parada de missão
- `strategist check-stale` permite CI verificar se recompilação é necessária sem carregar nada
- Formato único para todos os tipos de config — sem lógica de merge espalhada

**Negativas:**
- Requer `strategist compile` após edições manuais nos YAMLs (processo extra que pode ser esquecido)
- `.strategist/.compiled/` precisa estar no `.gitignore` — instalação garante isso via `ensureGitignore`
- Dois caminhos de carregamento para manter sincronizados (fast path e standard path)
