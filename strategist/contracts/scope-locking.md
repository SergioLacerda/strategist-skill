# SCOPE LOCKING
id: scope-locking
severity: medium

Toda mudança declara escopo antes de iniciar. Expansões durante execução
requerem pausa e nova aprovação — nunca são executadas inline.

## Regras

- Sniper executa apenas o que está em `tasks.md` aprovado no approval gate
- Qualquer arquivo fora do escopo declarado requer pausa + mini approval
- Melhorias de oportunidade descobertas durante execução vão para um novo item
  em `.analysis/todo/`, não são executadas no mesmo Sniper run
- "Enquanto estou aqui vou também..." é scope expansion — requer gate
- Refatorações adjacentes ao escopo aprovado são scope expansion — requer gate

## Quando pausar

O Sniper deve pausar e sinalizar ao Strategist quando:
- Um arquivo não listado em `tasks.md` precisaria ser modificado para completar a task
- Uma task revela uma dependência não mapeada no design
- A implementação exigiria mudança de contrato público

## Enforcement

Strategist verifica `tasks.md` antes de invocar o Sniper.
O approval gate inclui aviso explícito quando tasks.md contém writes fora de `<base_path>/`.
response-critic sinaliza scope drift detectado após execução.
