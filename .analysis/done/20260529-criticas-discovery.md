# Discovery — Avaliação de Críticas do Projeto
**Mission ID:** 20260529-criticas
**Date:** 2026-05-29
**Source:** `.analysis/todo/criticas_projeto.md`
**Task type:** architecture_analysis

---

## Executive Summary

O arquivo de críticas cobre 5 domínios: segurança, testes, consistência, robustez e shell script. As críticas são marcadas com severidade explícita: `!` = crítico, `~` = moderado, `i` = informacional, `✓` = positivo. Há 4 itens críticos (`!`), 9 moderados (`~`), 2 informacionais (`i`) e 2 positivos (`✓`).

**Recomendação de split:** As críticas devem ser divididas em **duas análises separadas** com base na urgência e no tipo de intervenção necessária:

| Análise | Foco | Itens | Tipo de trabalho |
|---------|------|-------|-----------------|
| `criticas-seguranca-testes` | Itens críticos e de testabilidade | 4 `!` + 1 `~` relacionado | Mudanças de código e CI — requerem execução |
| `criticas-qualidade` | Itens moderados e informacionais | 8 `~` + 2 `i` | Documentação, spec, normalização — menor risco |

---

## Justificativa do Split

Os 4 itens críticos (`!`) envolvem **vetores de segurança reais** (supply chain, schema bypass) e **ausência de garantias de contrato** (zero tests para approval gate e forbidden behaviors). São independentes dos itens de qualidade e exigem atenção imediata antes de qualquer adoção em produção.

Os itens moderados e informacionais são melhorias de qualidade que não bloqueiam uso atual: normalização de vocabulário, documentação incompleta, estratégia de retry, definição de mission_id. Podem ser tratados como backlog estruturado após a correção dos itens críticos.

Misturar os dois grupos num único plano dilui prioridade e cria planos com tarefas de risco muito diferente (patch de segurança ao lado de "adicionar CHANGELOG").

---

## Análise 1 — Segurança e Testes (Críticos)

### Módulos/Documentos Impactados

| Arquivo | Tipo de impacto |
|---------|----------------|
| `bootstrap.sh` | Correção de segurança — adicionar verificação SHA256 |
| `strategist/install.sh` | Correção de segurança — rollback do wizard |
| `.strategist/SKILL.md` (preflight) | Adição de YAML schema validation step |
| `.strategist/schemas/` | Ativação dos schemas existentes no preflight |
| `.github/workflows/` | Extensão do CI para rodar test harness |
| `README.md` | Aviso explícito sobre risco do curl pipe |

### Itens Críticos

**[!] bootstrap.sh — curl | bash sem verificação de integridade**
- O script baixa um tarball via HTTPS mas não verifica checksum.
- Ausência de pinning de versão no happy path (cai no `main` quando não há release).
- Vetor: supply chain attack ou MITM pode executar código arbitrário.
- Arquivos: `bootstrap.sh` (linhas do bloco `resolve_ref` e `download`).
- Recomendação: SHA256 do tarball contra `checksums.txt` assinado; `--ref=vX.Y.Z` como padrão seguro; aviso no README.

**[!] Nenhuma validação YAML no preflight**
- `active.yaml` e `roles/default.yaml` são carregados sem schema validation.
- Um campo `null` silencioso pode bypassar contratos de slot (ex: `risk_score: null` passa o check de `write_pending`).
- Arquivos: `.strategist/SKILL.md` (seção Preflight 2c/2d), `.strategist/schemas/intake.schema.yaml`.
- Recomendação: adicionar step `2a.validate` no preflight usando os schemas já existentes em `.strategist/schemas/`.

**[!] Zero testes automatizados para contratos críticos**
- Approval gate, drift self-correction e forbidden behaviors existem apenas como prosa.
- `.github/workflows/` atual só verifica shellcheck/lint.
- Recomendação: test harness com fixtures YAML simulando estados de missão (`approval_bypassed`, `slot_risk_mismatch`, `discovery_failed`).

**[!] Nenhum contrato de interface testado entre slots**
- Protocolo Strategist ↔ slot providers definido em prosa (SKILL.md + protocol.md).
- Não há schema validation dos inputs/outputs reais durante execução.
- Recomendação: mocks de slot para testar o pipeline sem providers reais; contrato de output documentado como schema.

### Item Moderado Incluído

**[~] install.sh --wizard sem rollback**
- Se o wizard falhar no meio, o workspace fica em estado parcial.
- Incluído nesta análise por ser um risco de integridade no mesmo fluxo de instalação.

---

## Análise 2 — Qualidade e Consistência (Moderados/Informacionais)

### Módulos/Documentos Impactados

| Arquivo | Tipo de impacto |
|---------|----------------|
| `strategist/protocol.md` | Normalização de vocabulário risk_score |
| `.strategist/SKILL.md` | Fonte de verdade do vocabulário canônico |
| `readme_detailed.md` | Adicionar seção sobre housekeeping_scan |
| `readme.md` | Consolidar ou estabelecer hierarquia clara |
| `.strategist/schemas/intake.schema.yaml` | Definir formato canônico de mission_id |
| `CHANGELOG.md` | Criar arquivo |
| `bootstrap.sh` (resolve_ref) | Tratar rate limit da GitHub API |
| `strategist/install.sh` | Adicionar require_cmd |

### Itens por Categoria

**[Consistência]**
- `~` Vocabulário de `risk_score` diverge entre `SKILL.md` (`write_pending`, `write_analysis`, `controlled`) e `protocol.md` (`read_only`, `controlled_write`). Confirmado por leitura direta dos dois arquivos. Normalizar para o vocabulário de `skill.yaml` e atualizar `protocol.md`.
- `~` `readme_detailed.md` não menciona a fase `housekeeping_scan` nem o mini approval gate.
- `~` CHANGELOG ausente com 18 commits e versão 1.0.0.
- `i` Dois READMEs criam ambiguidade de fonte de verdade. GitHub renderiza `readme.md` — manter como entry point, referenciar `readme_detailed.md` explicitamente.

**[Robustez]**
- `~` `mission_id` sem definição canônica — risco de colisão em missões concorrentes. Definir em `intake.schema.yaml` o formato `YYYYMMDD-HHMMSS-<4-char-slug>`.
- `~` Sem estratégia de retry para slots — falha transiente vs. permanente não distinguida.
- `~` `outcomes.jsonl` sem política de rotation/pruning — pode degradar context enrichment em projetos longos.
- `i` Sem suporte a missões paralelas — race conditions no filesystem são um risco arquitetural para equipes.

**[Shell Script]**
- `~` Sem verificação de dependências (`curl`, `tar`, `sed`, `grep`). Uma função `require_cmd` melhoraria UX em ambientes minimais (ex: Alpine Linux).
- `~` GitHub API rate limit não tratado no `resolve_ref()` — falha silenciosa cai no branch `main`.
- `✓` `set -euo pipefail` — boa prática confirmada.
- `✓` `TMPDIR` com `trap EXIT` — cleanup correto.

---

## Dependências Entre Análises

As duas análises são **independentes** — nenhuma é pré-requisito da outra para execução. A análise de segurança/testes tem prioridade mais alta. A normalização de vocabulário em `protocol.md` (Análise 2) não bloqueia as correções de segurança (Análise 1).

---

## Boundaries Summary

| Grupo | Escopo de mudança | Risk level |
|-------|-------------------|-----------|
| Análise 1 — Segurança+Testes | `bootstrap.sh`, `install.sh`, preflight em SKILL.md, CI workflows | Alto — mudanças em script de instalação e CI |
| Análise 2 — Qualidade | `protocol.md`, `readme_detailed.md`, `intake.schema.yaml`, novo `CHANGELOG.md` | Baixo — majoritariamente documentação e spec |
