# Proposal: Go Testing World-Class — Gaps Restantes
**Mission ID:** critique-tests2-20260530
**Date:** 2026-05-30
**Status:** refined

---

## O quê e por quê

O projeto já atende ~80% das recomendações de `critique_tests2.md`. Os 4 gaps
restantes são de manutenção e confiança — não são bloqueantes, mas causam dívida
técnica acumulada:

1. **Helpers duplicados** — `writeGzJSON`, `readGzJSON` e `minimalStrategistRoot`
   estão definidos em 4 arquivos diferentes. Qualquer mudança no formato de artifact
   (ex: schema do gzip) exige updates paralelos em múltiplos locais.

2. **testdata/ ausente** — Cases complexos de YAML usam strings inline. Dificulta
   leitura, diff em PR, e reutilização entre testes.

3. **Integração sem build tag** — `tests/` é integração real (instala em disco,
   compila pipeline completo), mas roda no `go test ./...` padrão misturado com
   unit tests rápidos.

4. **Sem benchmarks para hot paths** — `compile.CompileAll` e `stale.IsStale` são
   chamados em cada bootstrap do strategist. Regressões de performance são invisíveis.

## Abordagem

**Sprint 1:** Criar `internal/testutil/` e migrar todos os usos duplicados.
**Sprint 2:** `testdata/` para fixtures + build-tag em `tests/`.
**Sprint 3:** Benchmarks para `compile` e `stale`.

Escopo é estritamente o que está em `critique_tests2.md` — nenhuma refatoração além dos 4 gaps.
