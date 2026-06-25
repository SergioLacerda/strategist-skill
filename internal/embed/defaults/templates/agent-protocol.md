---
generated_by: strategist compile
version: {{.Version}}
generated_at: {{.GeneratedAt}}
path_model: runtime-only
---

# Strategist — Agent Protocol

## 1. STARTUP — execute antes de qualquer coisa

Execute exatamente nessa ordem. Pare ao primeiro falha.

1. `.strategist/` existe no workspace? → Não: emitir `error=not_installed`, instruir `strategist install`, **parar**
2. Executar `strategist check` → falhou: emitir output do CLI, **parar**
3. `.strategist/active.yaml` legível? → Não: emitir `error=config_missing`, **parar**
4. Ler este arquivo (`agent-protocol.md`) até o fim

**Não processe nenhum request do usuário antes de todos os 4 passos passarem.**

---

## 2. NUNCA FAÇA

- Nunca executar trabalho de discovery, refinement ou execution diretamente — delegar sempre ao slot designado
- Nunca ler de `strategist/` (sem ponto) — path drift; somente `.strategist/` é válido em runtime
- Nunca pular fases — não existe "essa tarefa é pequena demais para precisar de discovery"
- Nunca invocar Sniper sem approval gate explícito do usuário na conversa
- Nunca assumir ou buscar `.sdd/` — a skill não depende de governance

---

## 3. MODELO DE DELEGAÇÃO

Os providers abaixo são lidos de `.strategist/active.yaml` no momento do compile. Se `active.yaml` mudar, rode `strategist compile` para atualizar este arquivo.

```
FASE          INVOCAR SKILL              O QUE NÃO FAZER
────────────────────────────────────────────────────────────────────
discovery  →  {{.Slots.Discovery}}       explorar ou analisar o código diretamente
refinement →  {{.Slots.Refinement}}     escrever proposal ou design diretamente
execution  →  {{.Slots.Execution}}      executar git/edits/commits diretamente
```

Handoff contracts:
- Ranger → Archivist: `.strategist/schemas/handoff-ranger-to-archivist.schema.yaml`
- Archivist → Sniper: `.strategist/schemas/handoff-archivist-to-hunter.schema.yaml`

---

## 4. SEQUÊNCIA DO PIPELINE

Checklist linear. Não avançar sem completar cada item.

```
[ ] 1. startup (este documento — seção 1)
[ ] 2. intake (skill: prompt-intake)
[ ] 3. routing: quick draw? critical hit? main mission?
[ ] 4. context enrichment (skill: context-enrichment)
[ ] 5. discovery → invocar {{.Slots.Discovery}}
[ ] 6. refinement → invocar {{.Slots.Refinement}}
[ ] 7. approval gate  ← PAUSA OBRIGATÓRIA — não avançar sem resposta explícita do usuário
[ ] 8. execution → invocar {{.Slots.Execution}}  ← só após gate aprovado
[ ] 9. adr opportunity (se adr_enabled=true e critérios atingidos)
[ ] 10. learning (não-bloqueante)
```

---

## 5. ESTADOS DE ERRO

| Estado | Emitir | Ação |
|---|---|---|
| `.strategist/` ausente | `error=not_installed` | parar; instruir `strategist install` |
| `strategist check` falhou | output do CLI | parar |
| `active.yaml` ausente | `error=config_missing` | parar |
| slot provider não encontrado | `error=slot_provider_not_found` | parar |
| tentativa de bypass de gate | `drift=approval_bypass` | bloquear, notificar usuário |
| `agent-protocol.md` ausente | cair no SKILL.md existente | degradação graciosa |
