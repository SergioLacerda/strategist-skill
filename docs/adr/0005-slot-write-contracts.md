# ADR-0005 — Contratos de escrita por slot (write_pending / write_analysis / controlled)

**Status:** Accepted  
**Data:** 2026-05-28  
**Contexto:** Slot write contracts design (2026-05-28-slot-write-contracts-design.md)

---

## Contexto

Os slots de discovery (Ranger) e refinement (Archivist) precisam escrever artefatos em `.analysis/` como parte do fluxo normal da missão. Com o modelo original onde todos os slots eram `read_only` exceto Sniper (`controlled`), **qualquer escrita de artefato — mesmo um `.md` local em `pending/` — precisava passar pelo approval gate**.

Isso tornava o fluxo excessivamente interativo: criar `.analysis/pending/discovery.md` exigia um "yes" explícito do usuário, mesmo sendo uma operação de baixo risco sem impacto em código.

A questão: como diferenciar escritas de baixo risco (artefatos locais de análise) de escritas de alto risco (código, configs, arquivos do sistema)?

Alternativas consideradas:
- **Gate para tudo** — manter todos os slots como `read_only` + gate universal
- **Sem gate** — confiar nos providers para não escrever em lugares errados
- **Contratos de escopo** — declarar escopo e tipo de escrita permitidos por slot, validados em preflight

## Decisão

Três níveis de contrato de escrita, declarados no `skill.yaml` e validados pelo Strategist em preflight:

| Contrato | Escopo de escrita | Tipos permitidos | Approval gate |
|----------|-------------------|-----------------|---------------|
| `read_only` | nenhum | — | não aplicável |
| `write_pending` | `<base_path>/pending/` apenas | `.md` | não |
| `write_analysis` | `<base_path>/` e `<base_path>/refined/` | `.md` | não |
| `controlled` | qualquer lugar | qualquer tipo | **obrigatório** |

O `risk_score` do provider (declarado em `skill.yaml` do provider) deve corresponder ao contrato exigido pelo slot. Mismatch bloqueia em preflight com `slot_risk_mismatch`.

Violações em runtime:
- Escrita de tipo não-`.md` por `write_pending` ou `write_analysis` → `slot_write_type_violation`
- Escrita fora do escopo declarado → `slot_write_scope_violation`

## Consequências

**Positivas:**
- Discovery e refinement escrevem artefatos silenciosamente — fluxo natural sem interrupções desnecessárias
- Approval gate preservado apenas onde realmente importa (Sniper / slot `controlled`)
- Contratos verificados em preflight — erros de configuração detectados antes do início da missão, não no meio
- Modelo extensível: novos contratos podem ser adicionados sem alterar o pipeline principal

**Negativas:**
- Dois pontos de verificação de escopo: preflight (risk_score) e runtime (escrita real) — podem divergir se o provider não respeitar seu contrato declarado
- Provider malicioso com `risk_score: write_pending` poderia tentar escrever fora do escopo; a validação em runtime é necessária mas depende do orchestrador detectar a violação
- `known-providers.yaml` precisa ser mantido atualizado para providers que não declaram `risk_score` em seu `skill.yaml` — caso contrário, preflight não consegue validar
