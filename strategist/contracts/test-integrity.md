# TEST INTEGRITY
id: test-integrity
severity: high

Testes são a especificação executável do comportamento esperado.
O código se adapta ao teste — nunca o contrário.

## Proibido

- Enfraquecer uma assertion para fazer o teste passar
  (`assert.Equal` → `assert.NotNil` sem justificativa)
- Remover um caso de teste sem documentar o motivo
- Atualizar golden file ou snapshot sem diff explicado no commit
- Escrever um teste que não falha quando o comportamento que testa quebra
- Teste dependente de ordem de execução
- `time.Sleep` arbitrário em teste — usar `testify/assert.Eventually` ou channels
- Mock que torna o teste insensível a mudanças reais de comportamento
- Teste que só valida detalhes de implementação interna (não comportamento observável)

## Obrigatório ao modificar testes

Qualquer modificação em arquivo `*_test.go` que enfraqueça cobertura ou remova
casos de teste requer comentário explicando o motivo no mesmo commit.

## Enforcement

`testifylint` em golangci-lint detecta má-uso de assertions testify.
Coverage gate (90%) detecta regressão de cobertura.
response-critic avalia integridade dos testes após cada execução do Sniper.
