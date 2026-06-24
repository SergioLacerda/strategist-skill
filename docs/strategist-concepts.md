# Strategist — Conceitos Fundamentais

**Status:** Accepted
**Last Updated:** 2026-06-22

Referência dos cinco conceitos centrais da Strategist skill: papel, papel rankeado, armas, iniciativa e dojo.

---

## Visão Geral do Pipeline

O Strategist orquestra trabalho em três fases sequenciais, cada uma delegada a um **slot**:

```
Ranger (discovery) → Archivist (refinement) → [gate] → Sniper (execution)
```

O Strategist nunca executa trabalho diretamente — ele delega. Cada slot recebe um **provider** (arma) configurado em `active.yaml`. O conjunto provider + slot + contrato define um **papel**.

---

## Papel (Role)

Um papel é a combinação de um slot com seu contrato de comportamento. Existem três papéis canônicos:

| Papel | Slot | Contrato | Escrita autorizada |
|-------|------|----------|--------------------|
| **Ranger** | `discovery` | `write_pending` | `.md` em `<base_path>/pending/` |
| **Archivist** | `refinement` | `write_analysis` | `.md` em `<base_path>/` e `refined/` |
| **Sniper** | `execution` | `controlled` | Somente após approval gate |

Cada papel tem um contrato declarado em `.strategist/roles/<papel>.yaml` com cláusulas `must` e `must_not`. Exemplo (Ranger):

```yaml
must:
  - separar fatos, hipóteses e ambiguidades
  - incluir todos os campos do handoff contract no artefato
  - executar opportunity_attack e surfaçar resultados

must_not:
  - propor plano final como se aprovado
  - executar mudanças
  - passar contexto bruto ao Archivist (comprimir em evidence cards)
```

O Sniper requer aprovação explícita do usuário antes de qualquer execução — sem exceções.

---

## Papel Rankeado

Um papel rankeado é um provider que declara `provider_class: rankeado` em seu `skill.yaml`.

```yaml
# .strategist/brainstorming/skill.yaml
specialization_taxonomy:
  canonical_role: ranger
  provider_class: rankeado     # ← papel rankeado
```

A diferença em relação a um provider `(base)`:

| | Base | Rankeado |
|---|------|---------|
| `provider_class` | ausente ou `base` | `rankeado` |
| `specialization_taxonomy` | não declarado | `canonical_role` + `provider_class` preenchidos |
| Exibição em `initiative` | `(base)` | `rankeado` |
| Significado | Implementação genérica | Provider especializado, alinhado com o papel canônico |

Um provider rankeado não ganha permissões extras — a distinção é semântica e visível no comando `initiative`. Serve para comunicar que o provider foi projetado especificamente para aquele papel, não apenas plugado nele.

Providers rankeados instalados nesta workspace:

| Provider | Slot | Papel canônico |
|----------|------|---------------|
| `brainstorming` | discovery | Ranger |
| `openspec-explore` | refinement | Archivist |

---

## Armas (Weapons)

Armas são os providers concretos configurados em cada slot. A metáfora é: o papel (Ranger, Archivist, Sniper) é o guerreiro; a arma é a skill que ele empunha para executar seu trabalho.

Configuração em `.strategist/active.yaml`:

```yaml
slots:
  discovery: brainstorming       # arma do Ranger
  refinement: openspec-explore   # arma do Archivist
  execution: sdd-ask             # arma do Sniper
```

Cada arma é um skill com seu próprio `skill.yaml` resolvido em preflight pelo Strategist. O contrato de risco (`risk_score`) da arma deve corresponder ao contrato do slot:

| Slot | risk_score esperado |
|------|---------------------|
| discovery | `write_pending` |
| refinement | `write_analysis` |
| execution | `controlled` |

Para trocar uma arma, basta alterar o valor do slot em `active.yaml` e garantir que o `skill.yaml` do novo provider exista em `.strategist/skills/<provider>/skill.yaml`.

---

## Iniciativa

`strategist initiative` é o comando CLI que responde: *"quem está armado em cada slot agora?"*

```bash
$ strategist initiative

discovery    brainstorming      Ranger    rankeado   ✓ manifest OK
refinement   openspec-explore   Archivist rankeado   ✓ manifest OK
execution    sdd-ask            Sniper    (base)     ✓ manifest OK
```

Colunas:

| Coluna | Significado |
|--------|-------------|
| slot | `discovery`, `refinement`, `execution` |
| provider | ID do skill configurado em `active.yaml` |
| canonical_role | Papel canônico declarado no manifest (`ranger`, `archivist`, `sniper`) |
| class | `rankeado` se `provider_class: rankeado`, senão `(base)` |
| manifest status | `✓ manifest OK` se `.strategist/skills/<provider>/skill.yaml` existe e é válido; `⚠ manifest ausente` caso contrário |

O comando não faz chamadas ao LLM — lê apenas `active.yaml` e os arquivos `skill.yaml` locais. Útil para verificar rapidamente se a workspace está íntegra antes de iniciar uma missão.

```bash
# Com root customizado
strategist initiative --root /path/to/.strategist
```

---

## Dojo

O Dojo é o sistema de treino da Strategist skill — um health-check em duas camadas que valida se a skill está instalada, se os papéis estão preenchidos e se o pipeline opera corretamente.

### Camada 1 — Offline (zero LLM)

```bash
strategist dojo check <scenario>            # valida artefatos, emit log e manifests
strategist dojo check <scenario> --files-only  # valida apenas arquivos (sem emit log)
strategist dojo list                        # lista cenários disponíveis
```

Lê o `criteria.yaml` do cenário e verifica:
- **files_created**: arquivos existem, contêm seções obrigatórias e canary strings
- **emit_log**: eventos OTEL esperados presentes/ausentes no `.last-run/<scenario>/emit.log`
- **manifest_checks**: manifests de providers existem com campos obrigatórios

### Camada 2 — LLM (pipeline real com input sintético)

```
/strategist dojo <scenario>
```

Executa o pipeline completo com input de `.analysis/dojo/<scenario>/input.yaml`, escreve artefatos em `.analysis/dojo/run/` (isolado da produção) e ao final chama automaticamente a camada 1.

### Cenários disponíveis

| Cenário | O que valida |
|---------|-------------|
| `quick-draw` | Ideia bruta convertida em item no todo com canary `KATA_RAPIDO`; pipeline para no gate |
| `treasure-chest` | Chest plantado encontrado e canary `TORNEIO_DO_DOJO` incorporado na análise |
| `ranger-weapons` | Manifest do provider de discovery existe com campos `canonical_role` e `provider_class` |

### Estrutura de um cenário

```
.analysis/dojo/<scenario>/
├── input.yaml      # input sintético para a camada LLM
├── criteria.yaml   # contrato de validação (files, emit, manifests)
├── golden/         # artefatos de referência (opcional)
└── chests/         # treasure chests plantados (cenário treasure-chest)
```

### Regra de inocuidade

Todo `input.yaml` do dojo deve ser inócuo: ideia prefixada com `[dojo-fixture]`, targeting paths novos (ex: `docs/dojo/`). Se o Sniper disparar acidentalmente, nenhum código de produção é tocado.

### Adicionando um novo cenário

1. Criar `.analysis/dojo/<nome>/`
2. Escrever `input.yaml` com ideia inócua e canary string única
3. Escrever `criteria.yaml` referenciando o canary em `must_contain`
4. Validar sintaxe: `strategist dojo check <nome> --files-only`
