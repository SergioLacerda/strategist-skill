# ADR-0006 — E2E Test Entry Point via Full Install Pipeline

**Status:** Accepted
**Date:** 2026-06-02

---

## Contexto

O pipeline de compilação (`embed.Extract` → `compile.CompileAll` → `stale.Check`) não tinha testes E2E usando os defaults reais do embed. Ao escrever esses testes, duas abordagens foram consideradas para montar o fixture de teste:

**Opção A — Direta:** chamar `embed.Extractor{}.Extract(tmpDir)`, escrever `active.yaml` manualmente a partir do template, e então chamar `compile.Compiler{}.CompileAll(tmpDir, kiPath)`.

**Opção B — Install completo:** chamar `install.Service.Install` com `embed.Extractor{}` e `compile.Compiler{}` reais, e usar o diretório `.strategist/` resultante como fixture de teste.

## Decisão

A Opção B foi escolhida. `install.Service.Install` é o caminho canônico que produz um diretório `.strategist/` pronto para bootstrap em produção. Usá-lo diretamente significa que os testes E2E exercitam o mesmo code path que um usuário real percorre — incluindo geração de `active.yaml` a partir do template e a etapa de compilação embutida no installer. Isso dá mais confiança de que o que passa no CI é o que funciona em campo.

A Opção A foi rejeitada porque duplicaria a lógica de install nos testes, criando um segundo code path que poderia divergir do real — exatamente o tipo de gap que os testes foram projetados para fechar.

## Consequências

- **Mais fácil:** Qualquer regressão em `install.Service.Install` (extractor, template write, compile) é capturada pelos testes E2E, não apenas pelos testes unitários específicos de install.
- **Mais difícil:** Os testes E2E são ligeiramente mais lentos (install completo incluindo criação de shim) e dependem do pacote `install` — uma mudança nos campos de `install.Service` pode exigir atualização do helper `installReal`. Esse acoplamento é intencional: o helper deve acompanhar a API canônica de install.
- **Trade-off aceito:** O `mockExtractor` em `tests/install_test.go` precisou de um stub `ReadFile` porque `domain.FileExtractor` ganhou o método durante a migração para Go. Essa foi uma deriva de interface latente que o trabalho de E2E expôs e corrigiu.
