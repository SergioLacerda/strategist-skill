# Referência CLI — strategist

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

## install-global

Instala a skill globalmente em `~/.strategist/`, sem vínculo a um repositório específico.

```
strategist install-global [--target=<dir>]
```

**Flags:**

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `--target` | `$HOME` | Diretório base onde `~/.strategist/` será criado |

**Uso:** quando o agente precisa resolver a skill fora de qualquer diretório de projeto. O shim `~/.claude/skills/strategist/SKILL.md` aponta para `~/.strategist/` como raiz global.

Sempre roda em modo silent (sem wizard).

**Saída em sucesso:**
```
[Strategist] global install complete — skill root: /home/user/.strategist/
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

## Observabilidade (OpenTelemetry)

Todos os comandos (`install`, `compile`, `check-stale`, `sync-governance`) emitem spans OTel quando um collector está configurado. Sem configuração, o binário usa um provider no-op — zero overhead e zero conexões de rede abertas.

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
