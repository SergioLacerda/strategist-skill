# ADR-0003 — Approval gate obrigatório e jamais contornável

**Status:** Accepted  
**Data:** 2026-05-28  
**Contexto:** Guardrails de segurança do agente (guardrails-20260529)

---

## Contexto

O Strategist orquestra um agente de execução (Sniper) que opera em repositórios de código com potencial de alterar arquivos, executar scripts e modificar configurações. Sem controle, qualquer input do usuário poderia disparar execução imediata.

A questão central: o approval gate deve ser uma **preferência configurável** ou um **invariante do sistema**?

Alternativas consideradas:
- **Gate configurável** — permitir `auto_approve: true` em `active.yaml` para pipelines de CI
- **Gate por risk_score** — gates apenas para execuções de alto risco, livres para baixo risco
- **Gate sempre obrigatório** — nenhuma execução sem aprovação humana explícita, sem exceção

## Decisão

O approval gate é **mandatory_pause** — invariante do sistema, não configuração. Está declarado como `type: mandatory_pause` no `skill.yaml` e como forbidden behavior em `protocol.md`:

```yaml
forbidden_behaviors:
  - invoke_execution_slot_without_approval
```

Qualquer caminho no código que alcança o slot de execution sem resposta afirmativa do usuário é um bug, não um feature. O único resultado válido de um gate negado é `plan_only` — retornar o plano sem executar.

A única exceção prevista é `sdd_injection`, que pode injetar o provider de execução mas não pode remover o gate.

## Consequências

**Positivas:**
- Elimina toda uma classe de bugs de "execução indesejada" — o agente nunca pode agir sem que o humano tenha visto o plano
- Comportamento previsível independente do provider configurado no slot de execution
- Simplifica raciocínio sobre segurança: qualquer path que chega em Sniper sem gate explícito é detectável como violação
- Fixtures de teste (`approval-bypass.yaml`) codificam o invariante como spec executável

**Negativas:**
- CI/CD totalmente automatizado não é possível com a skill em modo padrão — requer intervenção humana em cada missão
- Fluxos de "batch processing" precisam de outra abordagem — o Strategist não é a ferramenta certa para automação sem supervisão
- Pode parecer excessivo para tarefas pequenas, mas o custo de um "yes" é menor que o custo de uma execução indesejada
