# ADR-0010 — Contratos ordenados e observabilidade da missão

**Status:** Accepted  
**Data:** 2026-06-18

---

## Contexto

O Strategist acumulou regras operacionais distribuídas entre `SKILL.md`, contratos
soltos, schemas, personas e documentação auxiliar. Esse formato gerou três tipos
de drift:

- dificuldade para identificar a ordem canônica de leitura dos contratos
- ambiguidade no handoff entre discovery e refinement
- observabilidade parcial sobre identidade da missão, artefatos gerados e gates

Também havia um objetivo explícito de tornar a experiência do usuário mais
determinística: a resposta precisa deixar claro que o agente está “dentro da
missão”, qual contrato está sendo seguido e quais artefatos são parte do fluxo.

Por fim, os contratos de input/output precisavam ficar alinhados com baseline de
telemetria compatível com OpenTelemetry, sem obrigar o consumidor a inferir
campos semânticos a partir de texto livre.

## Decisão

O baseline canônico do Strategist passa a ser:

1. `strategist/SKILL.md` atua como shell fino de roteamento
2. contratos operacionais são lidos sob demanda, em ordem explícita e estável
3. o fluxo discovery → refinement usa artefatos canônicos e não ambíguos
4. o envelope de resposta da missão fica externalizado em contrato dedicado
5. eventos e logs estruturados carregam atributos estáveis compatíveis com OTEL

### 1. Contratos ordenados

Os contratos humanos ficam particionados e numerados em `strategist/contracts/`
como sequência canônica de leitura:

1. `00-routing.md`
2. `01-bootstrap.md`
3. `02-intake.md`
4. `03-discovery.md`
5. `04-refinement.md`
6. `05-approval-gate.md`
7. `06-execution.md`
8. `07-adr.md`
9. `08-learning.md`
10. `09-response.md`
11. `10-telemetry.md`

`SKILL.md` não duplica a regra operacional detalhada. Ele apenas roteia para
essa sequência e para skills derivadas específicas.

### 2. Handoff canônico entre Ranger e Archivist

O fluxo oficial entre discovery e refinement passa a ser:

- Ranger gera `<base_path>/pending/<mission_id>-analysis.md`
- Archivist usa esse artefato como base obrigatória
- Archivist gera o pacote:
  - `refined/<mission_id>/proposal.md`
  - `refined/<mission_id>/design.md`
  - `refined/<mission_id>/tasks.md`

`pending/` deixa de ser o artefato principal desse handoff. Qualquer referência
anterior que tratava discovery como produtor de rascunho em `pending/` é drift.

### 3. Contrato de resposta da missão

O contrato narrativo e estrutural da resposta fica externalizado em
`strategist/protocol.md#response-contract` e operacionalizado por
`strategist/contracts/09-response.md`.

Toda resposta final da missão deve:

- identificar explicitamente a missão ou seu contexto operacional
- deixar claro o resultado da missão
- expor resumo de compliance do fluxo quando aplicável
- refletir o estado da missão sem exigir inferência pelo usuário

### 4. Contratos ricos de input/output

Inputs e outputs deixam de ser apenas convenções implícitas em texto corrido.
Eles passam a ser descritos por contratos dedicados e reforçados por schemas de
handoff, envelope de resposta e telemetria.

O objetivo não é maximizar número de campos, e sim produzir contratos ricos o
suficiente para:

- reduzir drift entre persona, runtime e documentação
- permitir validação automatizada
- preservar leitura humana rápida sob consulta

### 5. Baseline de observabilidade compatível com OTEL

Logs, eventos e spans do Strategist passam a privilegiar atributos estruturados
estáveis para missão, componente, gate, artefato e correlação.

O baseline inclui, quando aplicável:

- `mission_id`
- `correlation_id`
- `component`
- `selected_skill`
- `artifact_path`
- `runtime_mode`
- `output_profile`
- `gate_type`
- `gate_status`
- `gate_response`
- `approval_policy`
- `transition_group`
- `checkpoint_path`

Texto livre continua permitido como mensagem de operador, mas não substitui os
atributos estruturados necessários para correlação, auditoria e análise.

## Consequências

**Positivas:**

- reduz custo cognitivo para localizar a regra certa da skill
- elimina ambiguidade no handoff Ranger → Archivist
- torna a narrativa da resposta previsível e auditável
- melhora aderência entre contratos humanos, schemas e runtime
- fortalece compatibilidade com pipelines de observabilidade baseados em OTEL

**Negativas:**

- aumenta a disciplina necessária para manter contratos, schemas e embeds
sincronizados
- qualquer mudança futura no fluxo precisa atualizar múltiplas superfícies
canônicas
- consumidores legados que esperavam artefatos em `pending/` podem precisar de
ajuste

## Regras de manutenção

Mudanças futuras nesse fluxo devem respeitar:

1. ADRs vigentes antes de reinterpretar contratos
2. `strategist/contracts/` como fonte humana ordenada
3. schemas e embeds como espelhos executáveis do mesmo contrato
4. telemetria estruturada como requisito de runtime, não como detalhe opcional
