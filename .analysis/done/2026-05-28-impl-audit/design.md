# Design: Sincronização .strategist/SKILL.md
**Data:** 2026-05-28

## Abordagem
Aplicar as 4 diffs do canônico diretamente no runtime — edições cirúrgicas, não substituição completa do arquivo. Isso preserva qualquer customização que possa existir no runtime.

## Mapeamento de diffs

| # | Localização no runtime | Operação |
|---|----------------------|----------|
| 1 | Após linha 5 (header intro) | Inserir tabela de contratos (6 linhas) |
| 2 | L225-226 | Substituir artifact path flat pela especificação de subdiretório (9 linhas) |
| 3 | L240-248 | Substituir lógica heurística do gate pela lógica de leitura de tasks.md (6 linhas) |
| 4 | Final do arquivo (após última linha do Drift section) | Inserir drift pattern `route_plan_creation_to_sniper` |

## Verificação pós-execução
```bash
diff strategist/SKILL.md .strategist/SKILL.md
```
Expected: zero diff.
