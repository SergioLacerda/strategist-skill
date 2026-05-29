O mais urgente é o bootstrap.sh — fazer curl | bash sem verificação de checksum é um vetor de supply chain clássico. Não é específico desse projeto; é um anti-padrão difundido que vale corrigir agora que o projeto pode ser adotado por outros.
O segundo é a ausência de testes para os invariantes de segurança. O approval gate, os forbidden behaviors e os stop conditions são contratos de segurança — e atualmente existem apenas como prosa. Um conjunto de golden-file tests que simula estados de missão daria confiança real em refactors.
Inconsistência que vale corrigir antes de crescer:
O vocabulário de risk_score diverge entre SKILL.md (write_pending, write_analysis) e protocol.md (read_only, controlled_write). Pequeno hoje, mas vai confundir quem implementar slot providers de terceiros.

# seguranca
!bootstrap.sh executa curl | bash sem verificação de integridade. O script baixa um tarball de github.com e executa install.sh sem checksum, HTTPS-only enforcement explícito, ou pinning de versão no happy path. Um ataque de supply chain ou MITM pode executar código arbitrário na máquina do usuário.
Recomendação: Adicionar verificação de SHA256 do tarball contra um arquivo checksums.txt assinado. Alternativa mínima: publicar releases com hash verificável e exigir --ref=vX.Y.Z em ambiente de produção. Adicionar aviso explícito no README sobre o risco de piping curl direto.
!Nenhuma validação YAML no bootstrap/preflight. Se active.yaml ou roles/default.yaml estiver malformado, o comportamento do agente é indefinido. Um YAML inválido pode silenciosamente resultar em um null field que bypassa contratos de slot.
Recomendação: Adicionar step de schema validation no preflight usando jsonschema ou equivalente. Os schemas já existem em schemas/ — usá-los no preflight seria consistente com a arquitetura.
~install.sh —wizard não tem rollback. Se o wizard falhar no meio da configuração, o workspace pode ficar em estado parcial sem mecanismo de recovery.

# tests
!Zero testes automatizados para os contratos críticos. O approval gate, o drift self-correction, e os forbidden behaviors são invariantes de segurança do sistema — nenhum deles tem teste. O CI badge no README aponta para uma workflow que não verifica nada além de shellcheck/lint (inferido pela ausência de test artifacts).
Recomendação: Criar um test harness mínimo com fixtures YAML que simulam estados de missão (approval_bypassed, slot_risk_mismatch, scout_failed) e verificam que o agente emite os eventos bloqueados corretos. Mesmo um conjunto de golden-file tests aumentaria a confiança em refactors.
!Nenhum contrato de interface testado entre slots. O protocolo entre Strategist e Scout/Engineer/Hunter é definido em prosa (SKILL.md). Não há schema validation dos inputs/outputs reais durante execução, nem mocks para testar o pipeline sem slots reais.
~CI não verifica os schemas YAML. Os arquivos em schemas/ existem mas não há evidência de que são validados em CI.

# consistencia
Inconsistência no risk_score entre SKILL.md e protocol.md. O SKILL.md define Scout como write_pending e Engineer como write_analysis. O protocol.md define Scout e Engineer como read_only e Hunter como controlled_write. São vocabulários diferentes para o mesmo conceito — pode confundir implementadores de slot providers.
Recomendação: Normalizar para um vocabulário único. Se write_pending/write_analysis/controlled é o vocab canônico (como em skill.yaml), atualizar protocol.md para refletir isso.
~readme_detailed.md não menciona o housekeeping_scan. Esta fase importante (side quest manifest, mini approval gate) aparece detalhada no SKILL.md mas está ausente da documentação técnica. Um usuário que lê apenas o readme_detailed não sabe que existe.
~CHANGELOG ausente. Com 18 commits e versão 1.0.0, não há registro do que mudou entre iterações. Dificulta adoção em equipes que precisam avaliar impacto de updates.
iDois READMEs (readme.md + readme_detailed.md) cria ambiguidade. Não está claro qual é a fonte de verdade para novos usuários. O GitHub renderiza o lowercase readme.md, que é bom — mas a divisão cria manutenção duplicada.

# robustez
~mission_id não tem definição canônica. O campo aparece em paths de artefatos (<mission_id>-discovery.md), no mission result schema, e nos eventos de progresso — mas não há especificação de como é gerado (timestamp? UUID? slug do prompt?). Colisão de IDs em missões concorrentes é um risco não endereçado.
Recomendação: Definir explicitamente o formato do mission_id em intake.schema.yaml. Sugestão: YYYYMMDD-HHMMSS-<4-char-slug> para ser human-readable e collision-resistant.
~Nenhuma estratégia de retry para slots. Se Scout falha por timeout de rede (não por falha lógica), o protocolo manda parar. Uma distinção entre falha transiente vs. falha permanente permitiria retry automático seguro antes de escalar para o usuário.
~outcomes.jsonl sem tamanho máximo definido. Um projeto de longa duração pode acumular um arquivo de memória muito grande, degradando o context enrichment. Não há política de rotation ou pruning.
iSem suporte a missões paralelas. A arquitetura assume execução serial. Para equipes que rodam múltiplas missões simultaneamente no mesmo repo, o file-system state pode sofrer race conditions.

# shell script

✓set -euo pipefail presente. Boa prática. Garante que erros não sejam silenciados.
✓TMPDIR com trap EXIT. Cleanup correto do diretório temporário mesmo em caso de erro.
~Sem verificação de dependências. O script usa curl, tar, sed, grep — se algum estiver ausente (ex: Alpine Linux minimal), a mensagem de erro é críptica. Uma função require_cmd melhoraria o UX significativamente.
~GitHub API rate limit não é tratado. O resolve_ref() faz uma chamada à API pública sem autenticação. Em CI com muitas execuções, isso pode falhar silenciosamente e cair no main branch em vez de errar.