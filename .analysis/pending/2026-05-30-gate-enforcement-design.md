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

### Consulta offline de Ranger e Sniper

Quando Archivist é invocado, recebe como contexto adicional (leitura, sem invocar):
- `skill.yaml` + `SKILL.md` da skill que ocupa o papel de **Ranger**
- `skill.yaml` + `SKILL.md` da skill que ocupa o papel de **Sniper**

Archivist usa essas definições para calibrar o nível de detalhe das tarefas e produzir
instruções compatíveis com o que Sniper consegue executar.

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
| `.strategist/skills/archivist/SKILL.md` | Atualizar path de saída, adicionar seção side quests |
| `.strategist/personas/pragmatic.yaml` | Atualizar `approval_prompt` para o formato do gate único |

---

## Não-objetivos

- Não alterar a estrutura interna do output do Archivist atual (openspec)
- Não mudar o comportamento do Ranger
- Não adicionar gates adicionais além dos dois HARD-GATEs
- Não modificar o mecanismo de aprendizado (learning phase)
