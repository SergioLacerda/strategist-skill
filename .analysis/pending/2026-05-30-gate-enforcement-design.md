# Gate Enforcement — Design
**Data:** 2026-05-30
**Status:** pending
**Mission ID:** 20260530-gate-enforcement

---

## Problema

O orquestrador Strategist bypass o gate de aprovação e invoca Sniper diretamente após
o Ranger, sem passar pelo Archivist. O log de execução confirma:

```
[Strategist] phase=ranger
[Strategist] phase=sniper     ← Archivist e gate ausentes
```

**Causa raiz:** o pipeline depende exclusivamente de auto-controle textual do orquestrador
(Claude). Quando o Ranger produz saída concreta e acionável, o padrão de conclusão do
modelo infere "já sei o que fazer" e avança direto para execução. Não há mecanismo de
parada forçada entre as fases.

**Agravante:** o pipeline atual tem dois gates separados (mini gate para side quests +
gate principal), criando superfície de ambiguidade. O orquestrador pode usar um como
substituto do outro, ou ignorar ambos.

---

## Pipeline corrigido

```
Ranger → [HARD-GATE A] → Archivist → [HARD-GATE B / Gate único] → Sniper
```

Side quests não têm gate separado. São catalogados pelo Ranger, resumidos pelo
Archivist e apresentados no gate único junto com a missão principal.

---

## Seção 1 — Estrutura do pipeline

| Fase | Antes | Depois |
|------|-------|--------|
| Ranger | descobre missão + side quests | igual — entrega tudo em artefato único com seções separadas |
| Mini gate | gate separado para side quests | **eliminado** |
| HARD-GATE A | não existia | novo — parada incondicional entre Ranger e Archivist |
| Archivist | refinamento só da missão principal | refinamento da missão + catalogação leve de side quests |
| HARD-GATE B | gate informal com lógica de `tasks.md` ausente = `plan_only` silencioso | gate único obrigatório, apresenta missão + side quests juntos |
| Sniper | invocado se aprovado | igual — executa missão principal + side quests aprovados |

**HARD-GATE A** (instrução no SKILL.md, entre Ranger e Archivist):
```
<HARD-GATE>
Ranger concluiu. PROIBIDO invocar Archivist ou qualquer outro slot agora.
PROIBIDO executar qualquer tarefa identificada pelo Ranger.
Ação permitida: emitir evento done do Ranger. Depois: invocar Archivist.
Esta parada não tem exceção — nem para missões simples, nem para side quests.
</HARD-GATE>
```

**HARD-GATE B** (instrução no SKILL.md, entre Archivist e Sniper):
```
<HARD-GATE>
Archivist concluiu. PROIBIDO invocar Sniper agora.
PROIBIDO executar qualquer tarefa do plano refinado.
Ação permitida: apresentar o gate de aprovação ao usuário. Aguardar resposta explícita.
Esta parada não tem exceção — nem se o plano parece simples ou óbvio.
</HARD-GATE>
```

**Gate único apresentado ao usuário:**
```
[Strategist] Refinamento concluído.
Plano: <base_path>/refined/<mission_id>/

Missão principal:
  <resumo de tasks produzido pelo Archivist>

Side quests identificados pelo Ranger:
  [1] <item> — <motivo>
  [2] ...    (ou "nenhum" se vazio)

Aprovar execução? (yes / no / review)
```

Respostas aceitas:
- `yes / approve / authorize` → invocar Sniper
- `no / decline / stop` → retornar `status: plan_only`
- `review` → exibir conteúdo do artefato refinado e re-apresentar o gate

---

## Seção 2 — Contrato do Archivist e Mission ID

### Contrato de output (agnóstico à skill)

Strategist não conhece a estrutura interna do que o Archivist escreve. O contrato é:

- **Input recebido:** `mission_id`, `base_path`, discovery artifact path, mission contract
- **Escrita em:** `<base_path>/refined/<mission_id>/` (diretório)
- **Sinalização de conclusão:** `archivist complete. artifact=<base_path>/refined/<mission_id>/`

O gate verifica apenas se o diretório existe e tem conteúdo. A estrutura interna é
responsabilidade da skill que ocupa o papel de Archivist.

**Implicação:** se o usuário substituir a skill do papel de Archivist, o contrato de
output permanece — escrever em `refined/<mission_id>/` — mas a estrutura interna pode mudar.

### Mission ID único

Gerado no início da missão (fase de intake), propagado para todos os papéis:

```
mission_id = <YYYYMMDD>-<slug-do-prompt>
```

Artefatos por fase:
```
pending/<mission_id>-discovery.md     ← Ranger
refined/<mission_id>/                 ← Archivist (estrutura livre)
done/<mission_id>-report.md           ← Sniper
```

### Dois níveis de contexto offline

#### Nível 1 — Contexto do projeto (passivo, leitura)

Declarado em `active.yaml` ou `mission_contract` como `mission_docs_dir`. Pode conter:
- integrações com ferramentas externas
- base de conhecimento interna do projeto
- documentação técnica, ADRs, guias de estilo
- qualquer material que ajude a convergir a análise para a realidade do projeto

**Quem usa:** Ranger é o principal consumidor. Ele deve usar todas as ferramentas
disponíveis — leitura de arquivos, busca no codebase, consulta ao `mission_docs_dir` —
para produzir uma discovery com máxima profundidade antes de passar o artefato adiante.
O conhecimento do Nível 1 chega ao Archivist já "mastigado" dentro do artefato de
discovery — não é necessário recarregar a mesma fonte.

**Instrução explícita ao Ranger:** o `SKILL.md` do Ranger deve conter mandato de usar
todas as ferramentas disponíveis, incluindo o `mission_docs_dir`, antes de concluir a
fase de discovery. A discovery incompleta por falta de consulta ao contexto disponível
é um erro de Ranger, não uma limitação do pipeline.

#### Nível 2 — Peer review do Sniper (ativo, consulta estruturada)

O Ranger e o Sniper são assimétricos nesse nível:

- **Ranger → Archivist**: fluxo normal do pipeline. O conhecimento do Ranger —
  mandates, boas práticas, contexto L1 — já está materializado no artefato de
  discovery. Consultar o Ranger separadamente seria redundante.

- **Archivist → Sniper**: consulta de pré-execução. O Sniper é o único ator
  com **olhos frescos** — nunca viu a análise. Tem mandates, regras de governança
  e restrições operacionais que podem bloquear execução. Consultá-lo antes do gate
  evita surpresas pós-aprovação.

**Como o Archivist consulta o Sniper:**

Após rascunhar o plano, o Archivist apresenta ao Sniper:
- as tasks propostas (resumo estruturado, não o plano completo)
- pergunta explícita: "Você consegue executar essas tasks dentro dos seus mandates?
  Alguma task vai te bloquear por regra de governança ou restrição operacional?"

O Sniper responde com: sim/não por task + motivo quando há bloqueio + sugestão de
reformulação quando aplicável.

O Archivist incorpora o feedback e finaliza o plano. O que chega ao gate já foi
validado pelo executor — o usuário aprova um plano que o Sniper disse "consigo fazer".

**Quem resolve os paths para a consulta:** o orquestrador Strategist, que já resolveu
todos os providers no preflight (seção 2c), injeta os paths de `skill.yaml` e `SKILL.md`
do Sniper como contexto adicional ao invocar o Archivist.

### Correções de contrato no archivist atual

- `risk_score`: corrigir de `read_only` para `write_analysis`
- `output`: atualizar de `reviewed_plan_path: string` para `output_dir: string` (diretório)
- `SKILL.md` do Archivist: atualizar path de saída e adicionar seção de side quests

---

## Seção 3 — Enforcement em camadas

### Camada 1 — HARD-GATE no SKILL.md

Blocos explícitos de parada incondicional (detalhados na Seção 1).

### Camada 2 — Gate único consolidado

Eliminação do mini gate. Gate principal apresenta missão + side quests juntos.

### Camada 3 — `drift-patterns.yaml` — novos padrões

```yaml
- id: ranger_to_sniper_shortcut
  symptom: >
    Estou prestes a invocar o slot de execução logo após o slot de discovery,
    sem ter invocado o slot de refinement (Archivist).
  correction: >
    Parar. Invocar Archivist com o artefato do Ranger. Somente após Archivist
    concluir, apresentar o gate de aprovação.

- id: gate_artifact_absent_silent
  symptom: >
    O artefato do Archivist não foi encontrado no caminho esperado e estou
    prestes a retornar status plan_only silenciosamente.
  correction: >
    Parar. Ausência de artefato do Archivist é um erro de pipeline, não um
    resultado válido. Verificar se Archivist foi invocado. Se não: invocar agora.
    Se sim e falhou: emitir evento blocked com reason=archivist_failed.
```

### Camada 4 — `forbidden_behaviors` no `skill.yaml`

Adicionar:
- `invoke_sniper_before_archivist`
- `skip_archivist_for_simple_missions`
- `silent_plan_only_on_missing_artifact`
- `invoke_side_quest_sniper_without_main_gate`

---

## Arquivos afetados

| Arquivo | Tipo de mudança |
|---------|----------------|
| `.strategist/SKILL.md` | Adicionar HARD-GATEs, eliminar mini gate, atualizar gate único, remover lógica `tasks.md`, corrigir referências de pipeline |
| `.strategist/skill.yaml` | Atualizar `pipeline`, remover `side_quest_approval`, adicionar `forbidden_behaviors` |
| `.strategist/identity/drift-patterns.yaml` | Adicionar `ranger_to_sniper_shortcut` e `gate_artifact_absent_silent` |
| `.strategist/skills/archivist/skill.yaml` | Corrigir `risk_score`, atualizar `output` |
| `.strategist/skills/archivist/SKILL.md` | Atualizar path de saída, adicionar seção side quests, adicionar instrução de peer review com Sniper |
| `.strategist/personas/pragmatic.yaml` | Atualizar `approval_prompt` para o formato do gate único |
| `.strategist/skills/<ranger-skill>/SKILL.md` | Adicionar mandato explícito de usar todas as ferramentas disponíveis + `mission_docs_dir` antes de concluir discovery |

---

## Não-objetivos

- Não alterar a estrutura interna do output do Archivist atual (openspec)
- Não adicionar gates adicionais além dos dois HARD-GATEs
- Não modificar o mecanismo de aprendizado (learning phase)
- Não fazer o Archivist recarregar o `mission_docs_dir` — esse contexto chega via artefato do Ranger
