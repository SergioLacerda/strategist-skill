# Proposal — Qualidade e Consistência
**Mission ID:** 20260529-criticas
**Analysis group:** qualidade
**Priority:** Média — melhora adotabilidade e manutenção; não bloqueia uso atual
**Source:** `.analysis/pending/20260529-criticas-discovery.md` §Análise 2

---

## O Quê

Corrigir inconsistências de vocabulário, lacunas de documentação e pontos de fragilidade operacional do projeto Strategist:

1. **Vocabulário `risk_score` divergente** entre `SKILL.md` e `protocol.md` — dois termos diferentes para o mesmo conceito.
2. **`readme_detailed.md` sem documentação do `housekeeping_scan`** — fase importante ausente do material de referência.
3. **CHANGELOG ausente** — 18 commits, versão 1.0.0, sem registro de mudanças.
4. **Dois READMEs sem hierarquia clara** — ambiguidade de fonte de verdade para novos usuários.
5. **`mission_id` sem formato canônico** — risco de colisão; campo usado em paths de artefatos sem spec.
6. **Sem estratégia de retry para slots** — falha transiente é tratada igual a falha permanente.
7. **`outcomes.jsonl` sem política de rotation** — degradação de performance em projetos longos.
8. **Shell script: sem verificação de dependências** — UX ruim em ambientes minimais.
9. **GitHub API rate limit não tratado** — fallback silencioso para `main` em CI com muitas execuções.

## Por Quê

Estes itens foram marcados com severidade `~` (moderado) ou `i` (informacional) no arquivo de críticas. Nenhum cria um vetor de segurança ativo, mas todos criam fricção para novos usuários, implementadores de slot providers de terceiros, e equipes que tentam adotar o projeto.

O item mais impactante para terceiros é a divergência de vocabulário: um implementador que lê `protocol.md` vai declarar `risk_score: read_only` no seu slot provider e ser bloqueado pelo check de `write_pending` em SKILL.md sem entender por quê.

O `mission_id` sem spec é o único item com risco de corrupção de dados (colisão de IDs em missões concorrentes), mas como a arquitetura é serial e single-user por design, o risco é baixo e tratado com documentação da limitação.

## Escopo

Majoritariamente documentação, spec de esquema e pequenas adições de shell. Nenhuma mudança altera comportamento do agente em execuções normais.

**Fora de escopo:**
- Suporte completo a missões paralelas (risco arquitetural maior, não endereçável como melhoria incremental)
- Migração de `outcomes.jsonl` para outro formato (investigação futura)
