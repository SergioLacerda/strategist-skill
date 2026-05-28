Abaixo estão dois textos prontos para reaproveitar: um **README consolidado da thread** e um **resumo específico do Strategist como skill autônoma**.

# Strategist Skill + SDD Harness

## Runtime Cognitivo, Governança e Convergência de Agentes IA

## Visão Geral

Esta thread consolidou a relação entre dois conceitos complementares:

1. **Strategist** — uma skill autônoma de orquestração de análises, refinamento técnico e preparação de execução governada.
2. **SDD Harness** — um projeto de governança para IA que fornece protocolo, runtime, contratos, mandates, políticas, memória operacional e integração com ferramentas externas.

A ideia central é:

> O LLM não deve receber o prompt bruto nem operar livremente sobre o projeto inteiro.
> O runtime local deve classificar, enriquecer, filtrar, montar um dossiê mínimo e só então acionar a skill ou ferramenta adequada.

Nesse modelo, parte da inteligência operacional sai do LLM e passa para o ambiente governado: `.sdd`, skills, índices, cache, contratos, exemplos, rubricas, feedback humano e protocolos de execução.

---

## Problema

Agentes IA tendem a falhar quando operam sem governança:

* perdem contexto entre iterações;
* carregam documentação demais;
* confundem análise com execução;
* ignoram arquitetura;
* executam sem diagnóstico;
* corrigem sintomas em vez da causa raiz;
* entram em loops de retry;
* geram prompts grandes e pouco densos;
* não aprendem de forma controlada com acertos e erros;
* alteram código antes de haver plano, validação ou aprovação humana.

O problema não é apenas falta de inteligência do modelo.

O problema é ausência de:

```text
governança
+ roteamento
+ memória operacional
+ contexto seletivo
+ validação
+ feedback
+ aprendizado controlado
```

---

## Strategist

O **Strategist** é uma skill autônoma orientada a convergência.

Ela recebe uma demanda crua do usuário e transforma essa demanda em:

* análise contextual;
* refinamento técnico;
* plano de implementação;
* artefatos persistidos;
* checklist de execução;
* critérios de validação;
* ponto explícito de aprovação humana.

O Strategist executa automaticamente as fases de **discovery** e **refinement** quando não encontra conflitos, ambiguidades ou violações de governança.

Ele nunca executa implementação diretamente sem aprovação humana.

Quando a implementação é aprovada, ele delega a execução para um **Hunter configurado**, que pode ser uma skill ou provider especializado, como:

```text
caveman
rewriter
stabilize
codemod
implementer
```

---

## SDD Harness

O **SDD Harness** é o ambiente de governança para IA.

Ele fornece:

* estrutura `.sdd`;
* mandates;
* políticas;
* skill registry;
* plugin registry;
* contratos de execução;
* approval gate;
* progress events;
* runtime state;
* artifact state machine;
* validação de providers;
* memória operacional;
* contexto seletivo;
* dossiês mínimos;
* integração com ferramentas externas.

O SDD Harness não precisa conter toda inteligência internamente.

Ele pode funcionar como um **control plane** que permite plugar ferramentas externas de análise, refinamento, compressão, diagnóstico e execução.

Exemplo:

```text
SDD Harness
  ↓
governance + contracts + runtime + approval gate
  ↓
Strategist
  ↓
discovery provider
refinement provider
hunter / execution provider
```

---

## Relação entre Strategist e SDD Harness

O Strategist é autônomo, mas se beneficia do SDD.

O SDD é genérico, mas se fortalece ao plugar o Strategist.

A relação ideal é:

```text
Strategist = skill autônoma de análise e convergência
SDD Harness = runtime de governança e controle operacional
```

Quando o Strategist está dentro de um ambiente SDD, ele passa a operar com:

* mandates do projeto;
* políticas de segurança;
* práticas recomendadas;
* decisões arquiteturais;
* contratos de execução;
* modelos de aprovação;
* memória operacional;
* padrões de documentação;
* governança de contexto;
* validação de drift.

Isso aumenta a qualidade da análise e reduz o risco de execução solta.

---

## Fluxo do Strategist

Fluxo principal:

```text
Prompt bruto do usuário
  ↓
Awakening / Preflight
  ↓
Mission Intake
  ↓
Discovery automático
  ↓
Refinement automático
  ↓
Conflitos ou ambiguidades?
  ↓
Approval gate
  ↓
Delegação para Hunter configurado
  ↓
Validação + checklist + progresso
  ↓
Done
```

---

## 1. Awakening / Preflight

O Strategist inicia detectando o ambiente.

Ele pode:

* localizar `.sdd`;
* carregar registry;
* resolver roles;
* validar contratos;
* identificar providers disponíveis;
* preparar o mission context;
* verificar se há governança ativa.

Objetivo:

```text
garantir que a missão comece em modo governado
```

---

## 2. Mission Intake

O Strategist extrai premissas da solicitação do usuário.

Exemplos:

```text
entrega por sprint ou entrega total?
precisa de retrocompatibilidade?
é apenas análise ou também execução?
há prazo?
há restrições técnicas?
há escopo explícito?
```

O usuário pode antecipar essas decisões no prompt:

```text
/strategist analisar XTPO, entrega total, sem retrocompatibilidade
```

Assim, o Strategist não precisa perguntar novamente.

---

## 3. Discovery Automático

O Strategist executa ou aciona a fase de descoberta quando não há conflito.

Objetivo:

```text
entender o contexto antes de propor solução
```

A fase pode inspecionar:

* documentação;
* código;
* módulos relevantes;
* arquitetura;
* testes;
* riscos;
* ambiguidades;
* histórico de decisões.

Artefato esperado:

```text
.sdd/runtime/missions/review/<mission>-scout.md
```

---

## 4. Refinement Automático

Após a descoberta, o Strategist executa ou aciona refinamento técnico.

Objetivo:

```text
transformar análise em plano executável
```

O refinamento deve produzir:

* tarefas;
* subitens;
* detalhes técnicos;
* módulos afetados;
* goals;
* non-goals;
* exemplos de faça / não faça;
* riscos;
* checklist de execução;
* instruções para o Hunter.

Artefato esperado:

```text
.sdd/runtime/missions/reviewed/<mission>-engineer.md
```

---

## 5. Checagem de Conflitos

Antes de seguir para aprovação, o Strategist verifica:

* ambiguidades;
* conflitos de escopo;
* falta de informação;
* violação de mandates;
* provider ausente;
* risco incompatível;
* contrato incompleto.

Se houver conflito:

```text
pausa
solicita esclarecimento ao usuário
não segue para execução
```

---

## 6. Approval Gate

O approval gate é a fronteira entre planejamento e implementação.

Antes dele, o Strategist pode fazer:

```text
análise
descoberta
refinamento
documentação
planejamento
```

Depois dele, somente com aprovação humana:

```text
implementação
alteração de código
execução de tarefas
mudança real no projeto
```

Regra central:

```text
O Strategist nunca implementa sem aprovação humana.
```

Se o usuário não aprovar, o fluxo entrega o plano revisado e encerra.

---

## 7. Hunter Configurado

O Hunter é o slot de execução.

Ele pode apontar para uma skill ou ferramenta especializada.

Exemplos:

```text
caveman
rewriter
stabilize
codemod
implementer
```

Responsabilidades do Hunter:

* ler o plano revisado aprovado;
* executar uma tarefa por vez;
* respeitar escopo;
* atualizar checklist;
* validar resultado;
* parar em caso de drift, falha ou ambiguidade.

Regra:

```text
Implementação é executada pelo Hunter, não pelo Strategist.
```

---

## 8. Progress, Checklist e Feedback

Cada fase deve emitir feedback curto e persistir estado.

Exemplo:

```text
[SDD] phase=discovery status=running
[SDD] phase=discovery status=done artifact=...
[SDD] phase=refinement status=done artifact=...
[SDD] phase=approval status=waiting
[SDD] phase=hunter-execution status=running
```

Arquivos possíveis:

```text
.sdd/runtime/missions/progress/<mission>-progress.md
.sdd/runtime/state/current-mission.json
```

O objetivo é evitar a sensação de que o agente travou e, ao mesmo tempo, gerar rastreabilidade.

---

## Loop de Convergência

O loop do Strategist é um ciclo de redução de ambiguidade.

```text
entrada
  ↓
análise
  ↓
refinamento
  ↓
validação
  ↓
feedback
  ↓
ajuste
  ↓
próxima iteração
```

Cada ciclo deve aumentar a clareza da missão.

O loop existe para impedir que o agente tente resolver tudo de uma vez.

Em vez disso, ele opera progressivamente:

```text
tarefa grande
  ↓
decomposição
  ↓
subtarefas
  ↓
execução uma por vez
  ↓
validação
  ↓
checkpoint
```

---

## Aprendizado Controlado

O aprendizado não deve depender de “memória mágica” do LLM.

O aprendizado deve ser operacional e persistido.

Fluxo:

```text
missão executada
  ↓
resultado avaliado
  ↓
boas práticas registradas
  ↓
maus padrões registrados
  ↓
rubricas ajustadas
  ↓
próxima missão recebe dossiê melhor
```

Esse padrão pode ser chamado de:

```text
Convergence Learning Loop
```

Ou:

```text
Prompt Intake
→ Context Enrichment
→ Example Retrieval
→ Skill Execution
→ Response Critique
→ Human Verdict
→ Practice Memory Update
```

---

## Token Economy

O objetivo não é simplesmente reduzir arquivos.

O objetivo é entregar ao agente o menor contexto textual suficiente para decidir e executar.

Modelo:

```text
.sdd source
  ↓
runtime cache / indices
  ↓
retrieval seletivo
  ↓
dossiê mínimo
  ↓
LLM / skill
```

O ganho vem de:

* evitar retransmissão de contexto;
* não carregar documentação inteira;
* usar path-based loading;
* montar dossiês mínimos;
* aplicar progressive disclosure;
* separar análise de execução;
* usar skill routing;
* limitar escopo;
* validar por etapa;
* reaproveitar conhecimento operacional.

---

## Convergência

Convergência é sair de:

```text
prompt ambíguo
contexto solto
execução incerta
```

para:

```text
intenção clara
escopo definido
plano revisado
aprovação humana
execução governada
validação rastreável
```

O Strategist aumenta convergência porque força o fluxo:

```text
prompt → intake → discovery → refinement → approval → hunter → validation
```

O SDD Harness aumenta convergência porque fornece:

```text
governança
contratos
mandates
runtime
estado
memória
progress
approval gate
```

---

## Governança

A governança não deve ser apenas documentação.

Ela deve ser parte do runtime.

Exemplo de camadas:

```text
Context Injection
Runtime Awareness
Skill Routing
Execution Contract
Approval Gate
Validation
Progress Events
Scoring
Learning Loop
```

Assim, a governança deixa de ser uma recomendação e vira pressão estrutural.

O caminho correto passa a ser o caminho mais fácil, visível e auditável.

---

## Arquitetura Recomendada

```text
sdd-core
├── governance
├── contracts
├── plugin registry
├── skill registry
├── runtime state
├── progress
├── approval gate
├── artifact state machine
└── provider validation

strategist-skill
├── intake
├── discovery orchestration
├── refinement orchestration
├── conflict detection
├── approval gate integration
├── hunter delegation
├── progress contract
└── learning loop

providers
├── brainstorm
├── diagnose
├── openspec
├── caveman
├── rewriter
├── stabilize
└── codemod
```

---

## Decisão Arquitetural

O Strategist não precisa estar fundido ao core do SDD Harness.

A melhor separação é:

```text
SDD Harness = control plane / runtime de governança
Strategist = skill autônoma plugável
```

Mas, quando instalado dentro de um projeto SDD, o Strategist deve obedecer ao protocolo de governança do SDD.

Assim, os dois evoluem de forma independente, mas se reforçam quando usados juntos.

---

## Resumo Final

O **Strategist** transforma prompts crus em análises e planos confiáveis.

O **SDD Harness** garante que essa transformação ocorra dentro de um modelo governado.

Juntos, eles criam um runtime cognitivo onde:

```text
o agente não improvisa
o contexto é seletivo
a execução é aprovada
o progresso é visível
o aprendizado é controlado
a governança reduz drift
a convergência aumenta a cada ciclo
```

# Strategist Skill — Resumo Conceitual

## O que é

O **Strategist** é uma skill autônoma de orquestração cognitiva para agentes IA.

Sua função é transformar uma solicitação crua do usuário em:

* análise contextual;
* refinamento técnico;
* plano estruturado;
* documentação revisada;
* checklist de execução;
* critérios de validação;
* ponto de aprovação humana;
* delegação segura para execução.

Ele atua como uma camada de convergência entre o prompt do usuário, o conhecimento disponível no projeto e as ferramentas especializadas de análise ou execução.

---

## Objetivo

O objetivo do Strategist é evitar que o agente opere diretamente sobre prompts ambíguos.

Em vez de:

```text
prompt bruto → resposta ou implementação
```

o Strategist cria um fluxo governado:

```text
prompt bruto
  ↓
intake
  ↓
discovery
  ↓
refinement
  ↓
approval gate
  ↓
hunter execution
  ↓
validation
```

Isso reduz drift, retrabalho, excesso de contexto e risco de implementação prematura.

---

## Autonomia

O Strategist é uma skill autônoma.

Ela pode existir fora do SDD Harness, como uma ferramenta independente de análise e refinamento.

Porém, quando instalada dentro de um projeto SDD, ela se beneficia de:

* mandates;
* políticas;
* guardrails;
* decisões arquiteturais;
* padrões de documentação;
* práticas recomendadas;
* exemplos bons e ruins;
* rubricas;
* histórico de execução;
* memória operacional;
* contratos de runtime.

Essa sinergia permite que o Strategist produza análises mais alinhadas ao projeto.

---

## Capacidade de Melhorar Prompts

O Strategist não trata o prompt do usuário como entrada final.

Ele trata o prompt como matéria-prima.

A skill pode:

* classificar intenção;
* detectar escopo;
* identificar risco;
* extrair premissas;
* inferir restrições;
* buscar exemplos relevantes;
* aplicar templates;
* consultar boas práticas;
* montar dossiê mínimo;
* remover ruído;
* transformar demanda vaga em missão estruturada.

Exemplo:

```text
"quero analisar XTPO sem prazo e sem retrocompatibilidade"
```

Pode virar:

```yaml
mission:
  intent: requirements_analysis
  delivery_strategy: total
  legacy_compatibility: not_required
  execution_intent: plan_then_approval
```

Assim, o LLM ou provider recebe uma entrada muito mais clara.

---

## Discovery e Refinement Automáticos

O Strategist executa automaticamente discovery e refinement quando não encontra conflitos.

Discovery responde:

```text
O que existe?
Quais documentos importam?
Quais módulos são relevantes?
Quais riscos aparecem?
Quais ambiguidades existem?
```

Refinement responde:

```text
Quais tarefas devem ser feitas?
Em qual ordem?
Quais subitens existem?
Quais módulos serão afetados?
Quais são os goals e non-goals?
Quais são os exemplos de faça e não faça?
Como o Hunter deve executar?
```

Essas etapas podem ser executadas pela própria skill ou por providers configurados.

---

## Aprovação Humana Obrigatória

O Strategist nunca implementa sem aprovação humana.

Essa é uma regra central.

Antes da aprovação, ele pode:

```text
analisar
descobrir
refinar
documentar
planejar
```

Depois da aprovação, ele pode:

```text
delegar implementação
acompanhar progresso
validar resultado
atualizar checklist
```

Mas a implementação em si deve ser feita por um Hunter configurado.

---

## Hunter como Slot de Execução

O Hunter não precisa ser uma skill fixa.

Ele é um slot de execução que pode apontar para ferramentas diferentes:

```text
caveman
rewriter
stabilize
codemod
implementer
```

O Strategist mantém:

* aprovação;
* escopo;
* estado;
* feedback;
* checklist;
* artefatos;
* validação;
* governança.

O Hunter executa:

* mudanças;
* tarefas;
* refatorações;
* estabilização;
* codemods;
* implementação.

Essa separação gera confiança porque impede que o orquestrador vire executor solto.

---

## Autoaprendizado

O autoaprendizado do Strategist não deve depender da memória interna do LLM.

Ele deve acontecer no runtime.

O Strategist pode registrar:

* bons exemplos;
* maus exemplos;
* decisões aprovadas;
* decisões rejeitadas;
* padrões de prompt eficazes;
* critérios de avaliação;
* rubricas;
* tipos de ambiguidade recorrentes;
* estratégias que convergiram;
* estratégias que falharam.

Fluxo:

```text
missão
  ↓
resultado
  ↓
crítica
  ↓
feedback humano
  ↓
memória operacional
  ↓
melhor dossiê na próxima missão
```

Isso cria aprendizado controlado, auditável e reutilizável.

---

## Aderência a Modelos de Governança

O Strategist pode operar aderente a modelos de governança porque trabalha com:

* contracts;
* gates;
* manifests;
* roles;
* providers;
* policies;
* stop conditions;
* approval gates;
* progress events;
* checklists;
* validation reports.

Em um projeto SDD, ele pode obedecer aos mandates do projeto e usar o runtime como fonte de verdade operacional.

Isso evita:

* execução sem aprovação;
* escopo implícito;
* quebra de arquitetura;
* implementação fora do plano;
* drift entre análise e desenvolvimento;
* decisões não rastreáveis.

---

## Valor Principal

O valor do Strategist é transformar IA de resposta em IA de processo.

Em vez de apenas responder:

```text
Aqui está uma sugestão.
```

Ele conduz:

```text
Aqui está a análise.
Aqui está o plano refinado.
Aqui estão os riscos.
Aqui está o artefato.
Aguardando aprovação para implementar.
```

---

## Definição Curta

O Strategist é uma skill autônoma de convergência governada que transforma prompts crus em análises e planos estruturados, melhora entradas com conhecimento prévio, aprende operacionalmente com feedback, respeita modelos de governança e só delega implementação após aprovação humana explícita.

---

## Frase de Produto

```text
Strategist converts raw AI prompts into governed analysis, refined plans, and approval-based execution workflows.
```

## Frase em Português

```text
Strategist transforma prompts crus em análises governadas, planos refinados e fluxos de execução com aprovação humana.
```
