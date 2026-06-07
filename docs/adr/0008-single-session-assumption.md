# ADR-0008 — Single-Session Workspace Assumption

**Status:** Accepted
**Date:** 2026-06-06

---

## Contexto

A skill Strategist foi projetada para operar com uma única sessão ativa por workspace. Quando duas sessões Claude rodam simultaneamente contra os mesmos diretórios `.strategist/` e `.analysis/`, cinco classes de falha emergem (ver análise em `.analysis/refined/conflito_multi_thread/design.md`).

Três dessas falhas foram mitigadas:

- **F4** (partial `.config.gz` read): resolvido com atomic write via `os.Rename` em `writeGzJSON`
- **F1** (mission ID collision): mitigado por existence check no intake contract
- **F5** (gate sem identidade de sessão): resolvido adicionando `{mission_id}` ao `approval_prompt` das personas

Uma falha permanece intencionalmente não mitigada:

**F3 — Sniper cross-session blind spot:** Se Mission A e Mission B têm Sniper apontando para o mesmo arquivo, o segundo Sniper não detecta o conflito da outra sessão. O resultado é sobrescrita silenciosa no nível da skill — a colisão aparece como git conflict no momento do commit, não antes.

## Decisão

F3 é aceito como limitação conhecida da arquitetura single-session. A skill não implementa um lock registry cross-session ou coordenação de escrita entre sessões.

**Razão:** A complexidade de um lock registry distribuído (arquivo de lock, timeout, detecção de sessão morta, rollback) excede o benefício para um caso de uso que já tem mitigação adequada via git. O git conflict é visível, recuperável e ocorre antes de qualquer push.

**Recomendação para usuários:** Ao executar múltiplas sessões em paralelo, evitar que dois Snipers editem o mesmo arquivo na mesma janela de tempo. Fazer commits frequentes entre sessões para minimizar a janela de conflito.

## Consequências

- A skill documenta explicitamente que paralelismo de sessões não é garantido como seguro para escrita concorrente no mesmo arquivo.
- Usuários que precisam de paralelismo completo devem usar worktrees git separados.
- F3 pode ser revisitado se casos de uso de paralelismo se tornarem mais frequentes e o custo de git conflicts se tornar inaceitável.
