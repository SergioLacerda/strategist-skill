# Referência CLI — strategist

**Status:** Accepted
**Last Updated:** 2026-06-26

O binário `strategist` é construído em Go com [cobra](https://github.com/spf13/cobra). Todos os comandos seguem o padrão:

```
strategist <comando> [flags]
```

---

## install

Instala a skill Strategist em um repositório-alvo.

```
strategist install [--target=<dir>] [--wizard] [--silent]
```

**Flags:**

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `--target` | `.` (diretório atual) | Raiz do repositório onde `.strategist/` será criado |
| `--wizard` | `false` | Modo interativo: coleta mode, base_path e provider via prompts |
| `--silent` | `false` | Instalação sem prompts com defaults pragmatic (comportamento padrão quando nenhum flag é passado) |

**O que faz:**

1. Extrai os defaults embutidos para `<target>/.strategist/`
2. Gera `active.yaml` (wizard ou template pragmatic)
3. Adiciona `.strategist/.compiled/` ao `.gitignore`
4. Instala o shim em `~/.claude/skills/strategist/SKILL.md`
5. Compila todos os artefatos para `.strategist/.compiled/`

**Rollback:** se qualquer etapa falhar, os arquivos criados são removidos e o workspace é restaurado ao estado anterior.

**Exemplos:**

```bash
# Instalar com wizard no diretório atual
strategist install --wizard

# Instalar silenciosamente em outro repositório
strategist install --target=/path/to/project

# Via bootstrap (recomendado para primeira instalação)
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash
```

**Saída em sucesso:**
```
[Strategist] install complete → .
```

---

## compile

Compila todos os artefatos YAML da skill para gzip+JSON.

```
strategist compile [--root=<dir>]
```

**Flags:**

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `--root` | `.strategist` | Caminho para a raiz `.strategist/` |

**Artefatos gerados em `<root>/.compiled/`:**

| Arquivo | Conteúdo |
|---------|----------|
| `.index.gz` | `knowledge.index.yaml` compilado |
| `.domain.gz` | Templates de domínio (`templates/domain/`) compilados |
| `.config.gz` | `active.yaml` + `personas/` + `roles/` compilados |
| `.manifest.gz` | SHA256 dos 3 artefatos acima |

Recompe sempre que arquivos YAML de configuração forem editados manualmente.

**Saída em sucesso:**
```
[Strategist] compile complete → .strategist/.compiled/
```

---

## check-stale

Verifica se um artefato compilado está obsoleto em relação às suas fontes YAML.

```
strategist check-stale <artifact.gz>
```

**Argumento:** caminho para um arquivo `.gz` em `.strategist/.compiled/`.

**Códigos de saída:**

| Código | Significado |
|--------|-------------|
| `0` | Artefato fresco — fontes não foram modificadas |
| `1` | Artefato stale — pelo menos uma fonte foi modificada, ou o artefato/manifest não existe |

**Projetado para uso em CI/scripts:**

```bash
if ! strategist check-stale .strategist/.compiled/.config.gz; then
  strategist compile
fi
```

Um artefato é considerado stale quando:
- O arquivo `.gz` não existe
- `.manifest.gz` não existe no mesmo diretório
- Qualquer fonte listada em `artifact.sources` foi modificada após a compilação
- Qualquer fonte listada não existe mais no disco

---

## validate

Valida a árvore de configuração `.strategist/`.

```
strategist validate [--root=<dir>]
```

**Flags:**

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `--root` | `.strategist` | Caminho para a raiz `.strategist/` |

**Verificações realizadas:**

| Arquivo | O que é verificado |
|---------|-------------------|
| `active.yaml` | Existe, YAML válido, campos `mode` e `roles_config` presentes, `mode` é `pragmatic` ou `epic` |
| `personas/*.yaml` | Cada arquivo tem `tone_directive` e `phase_labels` |
| `roles/*.yaml` | Cada arquivo tem os slots `discovery`, `refinement` e `execution` |
| `knowledge.index.yaml` | Se presente, YAML válido |

**Saída em sucesso:**
```
[Strategist] validate OK — 7 check(s) passed (.strategist)
```

**Saída em falha:**
```
  ✗ active.yaml: invalid mode "custom" (must be pragmatic or epic)
  ✗ roles/custom.yaml: missing slot: execution
validate: 2 error(s) in .strategist
```

Útil em CI para garantir que edições manuais na configuração não introduziram erros de schema.

---

## version

Exibe a versão do binário.

```
strategist version
```

A versão é injetada em tempo de build via `-ldflags "-X main.Version=x.y.z"`. Em builds locais sem ldflags, exibe `strategist dev`.

**Saída:**
```
strategist v1.0.0
```

---

## check

Valida que os slot providers declarados em `active.yaml` estão instalados e satisfazem seus contratos de `risk_score`. Use antes de iniciar uma missão para garantir que o workspace está íntegro.

```
strategist check [--root=<dir>]
```

**Flags:**

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `--root` | `.strategist` | Caminho para a raiz `.strategist/` |

**Verificações realizadas:**

- `active.yaml` presente e parseável
- Para cada slot (`discovery`, `refinement`, `execution`):
  - `skills/<provider>/skill.yaml` existe (provider skill), **ou** `roles/<provider>.yaml` existe com o campo do slot (native role)
  - Providers skill devem declarar o `risk_score` correto: `discovery`/`refinement` → `write_analysis`; `execution` → `controlled`
  - Native roles são aceitos por correspondência de campo; sem verificação de `risk_score`

**Saída em sucesso:**
```
[Strategist] check=ok slots=[discovery:brainstorming, refinement:openspec-explore, execution:sniper] persona=epic root=.strategist
```

**Saída em falha:**
```
[Strategist] check=failed reason=slot_provider_not_found slot=execution
```

---

## initiative

Exibe os slot providers configurados e o estado atual do workspace. Leitura imediata sem chamada ao LLM.

```
strategist initiative [--root=<dir>]
```

**Flags:**

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `--root` | `.strategist` (auto-discovered) | Caminho para a raiz `.strategist/` |

**Saída:**

```
SLOTS                                                  
discovery      brainstorming      Ranger rankeado      ✓ manifest OK
refinement     openspec-explore   Archivist rankeado   ✓ manifest OK
execution      sniper             Sniper (base)        ✓ manifest OK
                                                       
WORKSPACE                                              
mode           epic                                    
base_path      .analysis                               
pending        0 cards                                 
done           49 missões                              
last mission   —                                       
```

A seção **SLOTS** exibe, para cada slot: provider configurado, papel canônico, classe (`rankeado` ou `base`) e status do manifest local em `.strategist/skills/<provider>/skill.yaml`.

A seção **WORKSPACE** exibe: `mode` e `base_path` de `active.yaml`, contagens de cards pendentes e missões concluídas, e o ID da última missão registrada em `memory/outcomes.jsonl` (se presente).

---

## dojo

Sistema de health-check da Strategist skill — valida que a skill está instalada, configurada e operando corretamente.

```
strategist dojo check <scenario> [--root=<dir>] [--files-only]
strategist dojo list
```

**Subcomandos:**

| Subcomando | Descrição |
|------------|-----------|
| `dojo check <scenario>` | Executa checks offline para um cenário |
| `dojo list` | Lista cenários disponíveis |

**Flags de `dojo check`:**

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `--root` | `.strategist` | Caminho para a raiz `.strategist/` |
| `--files-only` | `false` | Pula validação do `emit_log`; verifica apenas arquivos |

**Cenários disponíveis** (via `strategist dojo list`):

| Cenário | O que valida |
|---------|-------------|
| `critical-hit` | Edição de doc via fast path — Ranger e Archivist não invocados, gate inline apresentado, Sniper escreve apenas o arquivo alvo |
| `quick-draw` | Ideia bruta convertida em item pendente no todo, gate apresentado, execução não invocada |
| `ranger-weapons` | Lista providers disponíveis para o slot discovery e valida manifests |
| `treasure-chest` | Treasure chest encontrado e conteúdo incorporado na análise |

**Exemplo:**

```bash
# Validar cenário offline
strategist dojo check quick-draw

# Verificar apenas arquivos (sem emit log)
strategist dojo check quick-draw --files-only

# Listar cenários disponíveis
strategist dojo list
```

Para execução do pipeline completo com input sintético, use o skill `/strategist dojo <scenario>` via Claude Agent. Consulte `docs/strategist-concepts.md#dojo`.

---

## treasure-chest

Exibe o status dos treasure chests configurados, políticas de governance, e saúde do knowledge index compilado.

```
strategist treasure-chest [flags]
```

**Flags:**

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `--root` | `.strategist` (auto-discovered) | Caminho para a raiz `.strategist/` |
| `--scope` | `""` | Filtra saída por escopo de slot (`discovery`, `refinement`, `execution`) |
| `--index` | `false` | Reconstrói o knowledge index compilado a partir das fontes declaradas |
| `--include-historical` | `false` | Inclui fontes T2/T3 históricas na reconstrução (requer `--index`) |
| `--format` | `table` | Formato de saída: `table` ou `json` |

**Fontes consultadas:**

- `.strategist/active.yaml` — chests configurados e seus escopos
- `.strategist/treasure-chests.yaml` — políticas de trust e roteamento
- `.strategist/knowledge.index.yaml` — fontes de retrieval indexadas
- `.strategist/.compiled/.index.gz` — artefato compilado (fast-path)

**Exemplo de saída:**

```
CHESTS                                             
ID       PATH          SCOPE   TRUST   FRESHNESS   DRIFT
source   .sdd/source   all     T1      unknown     none

INDEX                                                       
artifact      .strategist/.compiled/.index.gz               
health        ok                                            
compiled_at   2026-06-26 18:19:47 UTC                       
```

---

## sync-governance

Sincroniza `.strategist/skill.yaml` com os mandates de governança SDD ativos.

```
strategist sync-governance [flags]
```

**Flags:**

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `--root` | `.strategist` | Caminho para a raiz `.strategist/` |
| `--sdd` | `.sdd` | Caminho para o diretório `.sdd/` |
| `--dry-run` | `false` | Exibe as mudanças sem escrever |

**O que faz:**

1. Lê `.sdd/metadata.json` para verificar o fingerprint de governança
2. Lê `.sdd/source/governance-core.json` para extrair mandates ativos
3. Compara mandates ativos contra `compliance.mandates` em `skill.yaml`
4. Aplica campos de governança ausentes (`validation_policy`, `budget_policy`, `telemetry_policy`)
5. Reporta drift antes de aplicar mudanças

**Exemplo:**

```bash
# Verificar drift sem escrever
strategist sync-governance --dry-run

# Aplicar sincronização
strategist sync-governance
```

Requer que `.sdd/` esteja presente no repositório (SDD governance). Sem `.sdd/`, o comando retorna erro.

---

## Observabilidade (OpenTelemetry)

Todos os comandos emitem spans OTel quando um collector está configurado. Sem configuração, o binário usa um provider no-op — zero overhead e zero conexões de rede abertas.

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `""` | Endpoint gRPC do collector (ex: `localhost:4317`). Vazio → no-op. |
| `OTEL_SERVICE_NAME` | `strategist` | Nome do serviço nos traces. |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true` | TLS desabilitado por padrão. Em produção: `false`. |

**Exemplo com collector local:**

```bash
# Subir Jaeger all-in-one (aceita gRPC na porta 4317)
docker run -d -p 16686:16686 -p 4317:4317 jaegertracing/all-in-one

# Executar com OTel habilitado
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_SERVICE_NAME=strategist \
strategist install --target .

# Ver traces em http://localhost:16686
```

**Atributos dos spans:**

| Span | Atributos |
|------|-----------|
| `strategist.install` | `strategist.target` |
| `strategist.compile` | `strategist.target` |
| `strategist.check_stale` | `strategist.artifact`, `strategist.cache.hit` |
| `strategist.sync_governance` | `strategist.mandates.count`, `strategist.mandates.missing` |
| `strategist.check` | `strategist.target` |
| `strategist.initiative` | `strategist.target` |

---

## Exit codes

Todos os comandos retornam um código de saída padronizado. Útil para CI/CD e scripts.

| Código | Significado | Exemplo de causa |
|--------|-------------|-----------------|
| `0` | Sucesso | Comando completou sem erros |
| `1` | Erro genérico / desconhecido | YAML inválido, arquivo não encontrado |
| `2` | Violação de governança / política | Pipeline bypass detectado sem aprovação |
| `3` | Artefato stale ou erro de integridade de config | `.compiled/` desatualizado, manifest ausente |

**Exemplo em script:**

```bash
strategist validate --root .strategist
code=$?

case $code in
  0) echo "OK";;
  2) echo "Governance violation — check pipeline state" >&2; exit 1;;
  3) echo "Config stale — run: strategist compile" >&2; exit 1;;
  *) echo "Error ($code)" >&2; exit 1;;
esac
```

**Exemplo em CI (GitHub Actions):**

```yaml
- name: Validate strategist config
  run: strategist validate --root .strategist
  # Exits 2 if governance bypassed, 3 if compiled artifacts are stale
```

---

## Instalação local (build from source)

```bash
# Clonar e compilar
git clone https://github.com/SergioLacerda/strategist-skill
cd strategist-skill

# Build
make build          # → bin/strategist

# Instalar no PATH (~/.local/bin/)
make install-local  # equivale a: install -m 755 bin/strategist ~/.local/bin/strategist

# Garantir que ~/.local/bin está no PATH
export PATH="$HOME/.local/bin:$PATH"

# Verificar
strategist version
```
