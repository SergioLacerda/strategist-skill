# NO HACK WITHOUT EVIDENCE
id: no-hack-without-evidence
severity: high

Hacks são proibidos por padrão. Qualquer exceção exige os 5 itens obrigatórios:

1. **diagnosis** — o que foi investigado antes de escolher o hack
2. **evidence** — por que a abordagem correta não funciona neste caso
3. **explicit trade-off** — o que se perde com o hack
4. **temporary marker** — `// HACK: <reason>` com issue ou task associada no mesmo comentário
5. **follow-up task** — issue registrada para resolver a causa raiz

> Hack pode existir como exceção. Nunca invisível.

## Comportamentos proibidos sem evidência

- Suprimir erros sem diagnose
- Enfraquecer testes para fazer o código passar
- `recover()` ou error handling silencioso genérico
- `_ = err` sem comentário explicando por que o erro é seguro de ignorar
- Alterar contratos públicos sem escalação
- Adicionar abstrações sem evidência de necessidade
- Bypassar camadas arquiteturais
- Desabilitar linters, testes ou checks
- Introduzir estado global mutável
- Adicionar dependências sem aprovação

## Enforcement

Este mandate é verificado pelo Archivist durante refinamento e pelo response-critic
após execução. Violações detectadas são reportadas como bloqueadores no learning loop.
