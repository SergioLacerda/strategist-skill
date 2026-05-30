# Proposal — Segurança e Testes (Críticos)
**Mission ID:** 20260529-criticas
**Analysis group:** seguranca-testes
**Priority:** Alta — bloqueia adoção em produção
**Source:** `.analysis/pending/20260529-criticas-discovery.md` §Análise 1

---

## O Quê

Corrigir quatro vulnerabilidades críticas e um risco de integridade de instalação no projeto Strategist:

1. **bootstrap.sh sem verificação de integridade** — curl pipe baixa e executa um tarball sem checksum. Vetor de supply chain clássico.
2. **Preflight sem validação YAML** — `active.yaml` e `roles/default.yaml` são carregados sem schema validation. Um `null` silencioso pode bypassar contratos de slot.
3. **Zero testes para contratos críticos** — approval gate, drift self-correction e forbidden behaviors existem apenas como prosa; nenhum está coberto por CI.
4. **Nenhum contrato de interface entre slots** — o protocolo Strategist ↔ providers é prosa; nenhum schema valida inputs/outputs reais.
5. **install.sh wizard sem rollback** — falha no meio da configuração deixa o workspace em estado parcial sem recuperação.

## Por Quê

Estes itens foram marcados com severidade `!` (crítico) no arquivo de críticas. Todos criam ou expõem vetores ativos:

- O bootstrap é o ponto de entrada público do projeto. Um atacante que comprometer o CDN ou executar um MITM entrega código arbitrário a qualquer usuário que rode o comando de instalação documentado no README.
- A ausência de validação YAML significa que um arquivo de configuração corrompido (por edição manual, template incompleto ou truncamento de disco) pode resultar em comportamento de agente indefinido — incluindo bypass silencioso dos contratos de slot que são a garantia de segurança central do Strategist.
- Sem testes para approval gate e forbidden behaviors, qualquer refactor pode introduzir um bypass desses contratos sem que o CI detecte. O projeto atual tem um CI badge que só verifica sintaxe shell (`bash -n`).
- O wizard é a segunda forma de instalação mais usada. Estado parcial após falha cria workspaces quebrados que o usuário não sabe como recuperar.

## Escopo

Mudanças de código e CI. Não inclui itens de documentação, vocabulário ou robustez de protocolo (esses vão na análise `criticas-qualidade`).

**Fora de escopo:**
- Suporte a missões paralelas (informacional, não crítico)
- Normalização de vocabulário risk_score (qualidade, não segurança)
- Changelog e documentação (análise separada)
