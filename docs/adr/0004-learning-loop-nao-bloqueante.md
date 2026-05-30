# ADR-0004 — Learning loop não-bloqueante

**Status:** Accepted  
**Data:** 2026-05-28  
**Contexto:** Design da fase de learning e learning-curator

---

## Contexto

Após cada missão, o Strategist propõe registrar o outcome (`memory/outcomes.jsonl`) e ajustar prioridades de fontes de conhecimento (`memory/source-hints.yaml`). Essa fase envolve: executar `response-critic`, executar `learning-curator`, apresentar checkpoint ao usuário e aguardar resposta.

Qualquer uma dessas etapas pode falhar: timeout do LLM, usuário ignora o checkpoint, sub-skill retorna erro, arquivo de memória corrompido.

A questão: se o learning falhar, o que acontece com o resultado da missão?

Alternativas consideradas:
- **Bloqueante** — missão só retorna após learning completo; falha de learning falha a missão
- **Opcional configurável** — usuário escolhe se quer learning via `active.yaml`
- **Não-bloqueante** — learning roda após a missão; falha não altera o resultado entregue ao usuário

## Decisão

A fase de learning é **não-bloqueante**: roda após execution, mas qualquer falha (timeout, erro de sub-skill, usuário recusa o checkpoint) resulta em log e retorno do resultado da missão sem alteração.

Declarado em `protocol.md` e `skill.yaml`:
```yaml
- stage: learning
  skill: learning-curator
  blocking: false
```

`learning-curator` tem forbidden behavior `block_mission_result_on_learning_failure` — a sub-skill não pode reter o resultado da missão aguardando o learning.

O learning buffer (`memory/outcomes.tmp`) tem limite de 20 entradas antes de ser flushed para `outcomes.jsonl` — proteção contra acúmulo infinito se o flush principal nunca ocorrer.

## Consequências

**Positivas:**
- O usuário sempre recebe o resultado da missão — falha de infraestrutura de memória nunca bloqueia o trabalho
- Simplifica raciocínio sobre missões: o resultado é determinístico independente do estado da memória
- Learning é um benefício acumulado, não um requisito para funcionamento — system degraded gracefully
- Falhas de learning são observáveis via log sem impactar o fluxo principal

**Negativas:**
- Se o learning falha sistematicamente (ex: arquivo de memória corrompido), o problema pode passar despercebido por muitas missões sem que o usuário perceba
- O learning buffer é uma segunda via de escrita de outcomes — dois caminhos para o mesmo dado podem causar duplicatas se o flush principal e o buffer flush coincidirem
- Usuário que ignora o checkpoint repetidamente perde o benefício acumulado do knowledge system sem feedback explícito
