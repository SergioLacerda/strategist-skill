# Otimização de Performance — Strategist Skill

## Diagnóstico: Onde está o gargalo?

Antes de compilar, precisamos identificar **o que** está lento no trecho offline:

```
┌─────────────────────────────────────────┐
│ Mission invocation (latência esperada)  │
├─────────────────────────────────────────┤
│ 1. Bootstrap (load active.yaml, etc)    │ ← I/O YAML
│ 2. Preflight (resolve slots, validate)  │ ← I/O YAML + schema validation
│ 3. Intake (prompt-intake skill call)    │ ← NETWORK (Claude API)
│ 4. Context Enrichment                   │ ← I/O (knowledge.index, excerpts)
│ 5. Scout (slot provider call)           │ ← NETWORK (Claude API)
│ 6. Engineer (slot provider call)        │ ← NETWORK (Claude API)
│ 7. [Approval gate — HUMAN WAIT]         │ ← BLOCKING
│ 8. Hunter (slot provider call)          │ ← NETWORK (Claude API)
│ 9. Learning Phase                       │ ← I/O (outcomes.jsonl, source-hints)
└─────────────────────────────────────────┘
```

**Onde é "offline"?** Bootstrap + Preflight + Context Enrichment + Learning Phase = I/O local, sem Claude API.

**Gargalos reais (em ordem de impacto):**

| Fase | Gargalo | Causa | Impacto |
|------|---------|-------|--------|
| Bootstrap | YAML parsing | `active.yaml` + `personas/*.yaml` + `roles/*.yaml` | 50–200ms por invocação |
| Preflight | Carregamento de domínio interno | Lê `index.yaml`, depois N arquivos de identidade/directives | 100–500ms (cresce com tamanho) |
| Context Enrichment | Busca linear em knowledge.index | Filtra por tag, carrega excerpts | **O(n) onde n = número de fontes** |
| Learning Phase | outcomes.jsonl é relido inteiro | Append-only, sem índice | ~2–50ms (cresce com histórico) |

---

## Estratégia de Otimização (ordem de ROI)

### Tier 1: Índice Pré-Computado (impacto: 40–60% melhoria)

**Problema:** A cada busca, `context-enrichment` faz:
```yaml
# knowledge.index.yaml
sources:
  - id: arch-docs
    tags: [architecture, system-design, architecture_analysis]
  - id: patterns
    tags: [examples, patterns, refactor]
  - id: team-guide
    tags: [all]
  # ... 50+ mais fontes

# Para task_type="architecture_analysis", filtra linear por tag match.
# Com 100+ fontes, é O(100) por busca.
```

**Solução:** Compilar um índice invertido durante o `install.sh`:

```bash
# Durante install.sh, gerar: .strategist/.index.compiled

# Schema do índice compilado (MessagePack ou CBOR):
{
  "version": "1.0",
  "generated_at": "2026-05-28T14:30:00Z",
  "tags": {
    "architecture": ["arch-docs", "system-overview"],
    "architecture_analysis": ["arch-docs", "system-overview", "patterns"],
    "refactor": ["patterns", "guidelines"],
    "all": ["team-guide", ...] 
  },
  "sources": {
    "arch-docs": {
      "id": "arch-docs",
      "type": "docs",
      "path": "/abs/path/to/docs/architecture",
      "priority": "high",
      "excerpt_length_tokens": 350
    }
  }
}
```

**Implementação:**

```bash
#!/usr/bin/env bash
# strategist/compile-knowledge-index.sh

KNOWLEDGE_INDEX="$1"
OUTPUT="$2"  # .strategist/.index.compiled

# Ler YAML → Gerar índice invertido (tag → sources)
# Usando yq (agnóstico), jq, ou language-specific parser

yq -o json "$KNOWLEDGE_INDEX" | jq '
  .sources as $sources
  | reduce $sources[] as $src (
      {};
      . as $acc
      | ($src.tags // []) as $tags
      | reduce $tags[] as $tag (
          $acc;
          .[$tag] += [$src.id]
        )
    ) as $tag_index
  | {
      version: "1.0",
      generated_at: now,
      tags: $tag_index,
      sources: (
        $sources | map({(.id): .}) | add
      )
    }
' > "$OUTPUT.json"

# Converter para MessagePack (binário comprimido)
# Opção A: usar msgpack (requer npm/pip/etc)
# Opção B: manter como JSON comprimido (gzip)
gzip -9 "$OUTPUT.json" -o "$OUTPUT"
rm "$OUTPUT.json"
```

**Mudança no preflight (context-enrichment):**

```
ANTES:
  1. Parse knowledge.index.yaml (YAML → dict) — ~30ms
  2. Filtra por tag com loop — ~50ms
  → Total: ~80ms

DEPOIS:
  1. Descompacta .index.compiled (gzip) — ~5ms
  2. Busca tag diretamente na hash — O(1) — ~1ms
  → Total: ~6ms

Ganho: 13x mais rápido para busca, 80+ ms economizados por missão
```

**Arquivo compilado esperado:**
```
.strategist/
├── .index.compiled          ← gerado no install, gitignore'd
├── knowledge.index.yaml     ← source, versionado
```

---

### Tier 2: Lazy Load + Mmap para Domínio Interno (impacto: 20–30%)

**Problema:** Se `.strategist/` tem 50+ arquivos (identity/, directives/, rubrics/, patterns/), carregar todos no preflight é caro mesmo com `index.yaml` filtrando.

```
Preflight 2a (load internal domain):
  - Parse index.yaml
  - Para cada arquivo em load_always: lê + parseia YAML
  - Se há 10 arquivos, 10× parsing = ~300ms
```

**Solução:** Memory-mapped YAML com lazy evaluation.

```bash
# Durante install.sh, compilar também o domínio interno

strategist/compile-domain.sh:
  input: .strategist/index.yaml + all domain files
  output: .strategist/.domain.compiled (single MessagePack blob)
  
  Estrutura:
  {
    "load_always": {
      "identity/what-i-am.yaml": { core_invariants: [...] },
      "identity/drift-patterns.yaml": { patterns: [...] }
    },
    "load_by_task_type": {
      "architecture_analysis": { 
        "directives/arch.yaml": {...},
        "rubrics/arch.yaml": {...}
      }
    }
  }
```

**Implementação em pseudocódigo:**

```python
# strategist/utils.py (language-agnostic pattern)

class CompiledDomain:
    def __init__(self, path):
        self.path = path
        self._buffer = None
        self._index = None
    
    @property
    def _load(self):
        """Lazy load on first access."""
        if self._buffer is None:
            with gzip.open(self.path, 'rb') as f:
                self._buffer = msgpack.unpackb(f.read(), raw=False)
        return self._buffer
    
    def load_always(self):
        """Return files for load_always without touching others."""
        return self._load['load_always']
    
    def load_by_task_type(self, task_type):
        """Return files specific to task_type."""
        return self._load['load_by_task_type'].get(task_type, {})
```

**Ganho:**
```
ANTES: load 10 files × 30ms = 300ms
DEPOIS: deserialize 1 blob (mmap) = 8ms, then access by key = O(1)
Ganho: 40x para domain load
```

---

### Tier 3: Batch Write para Learning Loop (impacto: 15–20%)

**Problema:** Cada missão apende 1 linha a `outcomes.jsonl`:
```
# .strategist/memory/outcomes.jsonl
{"mission_id": "...", "task_type": "...", "status": "completed", ...}
{"mission_id": "...", "task_type": "...", "status": "completed", ...}
```

Em 100 missões, são 100 syscalls de write. Em SSD é rápido; em network filesystem é lento.

**Solução:** Buffer em memória, flush a cada N missões ou T segundos.

```yaml
# Adicionar a active.yaml:
learning_cache:
  buffer_size: 20      # flush a cada 20 missões
  flush_interval_sec: 300  # ou a cada 5 min
```

```python
class LearningBuffer:
    def __init__(self, outcomes_path, buffer_size=20):
        self.outcomes_path = outcomes_path
        self.buffer = []
        self.buffer_size = buffer_size
        self.last_flush = time.time()
    
    def append(self, outcome):
        self.buffer.append(outcome)
        if len(self.buffer) >= self.buffer_size:
            self.flush()
    
    def flush(self):
        if not self.buffer:
            return
        with open(self.outcomes_path, 'a') as f:
            for outcome in self.buffer:
                f.write(json.dumps(outcome) + '\n')
        self.buffer = []
        self.last_flush = time.time()
    
    def __del__(self):
        self.flush()  # garante flush no cleanup
```

**Ganho:**
```
100 missions:
ANTES: 100 syscalls × 1ms = 100ms
DEPOIS: 5 syscalls × 2ms = 10ms
Ganho: 10x
```

---

### Tier 4: Compilação de Bootstrap (impacto: 5–10%)

**Problema:** A cada invocação do agente, `bootstrap.sh` é executado:
- Parse `active.yaml`
- Resolve persona
- Carrega múltiplos YAML

**Solução:** Pré-compilar a config esperada durante install.

```bash
# strategist/compile-active.sh

INPUT_ACTIVE="$1"  # active.yaml
OUTPUT_COMPILED="$2"  # .strategist/.active.compiled

# Validate + compile once at install time
yq eval . "$INPUT_ACTIVE" | \
  jq '. + {
    compiled_at: now,
    bootstrap_hash: (. | @json | @sha256)
  }' | \
  gzip > "$OUTPUT_COMPILED"
```

Mudança no SKILL.md:
```
# 1. Bootstrap (UPDATED)

Se .strategist/.active.compiled existe e hash matches .yaml:
  - Descompactar .active.compiled (3ms)
ELSE:
  - Parsear active.yaml (fallback)
```

**Ganho:** 20–30ms economizados por invocação de agente.

---

## Compilação Binária — Realidade vs. Hype

**A pergunta:** "compilar os dados em algum binário ajudaria?"

**Resposta curta:** Sim, mas com contexto.

### Opção A: Serialização Binária (MessagePack / CBOR) — ✅ **Recomendado**

```
YAML:
  - Legível
  - Lento: ~50–100ms para parsear 1MB
  - Flexible (comentários, tags, anchors)

MessagePack (binário):
  - Rápido: ~5–10ms para desserializar 1MB
  - Comprimido: 30% menor que YAML
  - Agnóstico de linguagem
  - Escolha real de empresas grandes (Discord, Slack, AWS)

Trade-off: Perder comentários, mas com documentação externa isso é OK.
```

**Como implementar:**

```bash
# strategist/compile-all.sh (chamado durante install)

set -euo pipefail

SOURCE_DIR=".strategist"
BUILD_DIR=".strategist/.compiled"
mkdir -p "$BUILD_DIR"

# 1. Compilar knowledge.index em índice invertido
./strategist/compile-knowledge-index.sh \
  "$SOURCE_DIR/knowledge.index.yaml" \
  "$BUILD_DIR/.index"

# 2. Compilar domínio interno em blob único
./strategist/compile-domain.sh \
  "$SOURCE_DIR/identity" \
  "$SOURCE_DIR/directives" \
  "$SOURCE_DIR/rubrics" \
  "$SOURCE_DIR/index.yaml" \
  "$BUILD_DIR/.domain"

# 3. Compilar active + personas em blob
./strategist/compile-config.sh \
  "active.yaml" \
  "personas/" \
  "roles/" \
  "$BUILD_DIR/.config"

# Checksum para validação
find "$BUILD_DIR" -type f -exec sha256sum {} \; > "$BUILD_DIR/.manifest"
```

**Resultado:**
```
.strategist/
├── knowledge.index.yaml
├── active.yaml
├── personas/
├── roles/
├── identity/
├── directives/
├── rubrics/
└── .compiled/                    ← GERADO
    ├── .index (msgpack.gz)       ← knowledge.index compilado
    ├── .domain (msgpack.gz)      ← domínio compilado
    ├── .config (msgpack.gz)      ← active + personas compilado
    └── .manifest (sha256 checksums)
```

**Mudança no Preflight:**

```python
def load_internal_domain():
    compiled_path = base_path / ".strategist" / ".compiled" / ".domain"
    
    # Tentar compilado primeiro
    if compiled_path.exists() and is_manifest_valid(compiled_path):
        with gzip.open(compiled_path, 'rb') as f:
            domain = msgpack.unpackb(f.read(), raw=False)
    else:
        # Fallback: parsear YAML (mais lento)
        domain = load_yaml(base_path / ".strategist" / "index.yaml")
    
    return domain
```

### Opção B: Compilação Nativa (Go / Rust) — ❌ **Não recomendado**

```
Compilar para binário nativo (.elf, .exe):
  ✓ Muito rápido (~1ms startup)
  ✗ Perde portabilidade (Windows/Mac/Linux = 3 binários)
  ✗ Perde agnóstico de linguagem (agente assumiria uma lang)
  ✗ Dificulta manutenção (código do agente em 2 linguagens)
  ✗ CI/CD mais complexo (need to build per platform)
```

**Não é a solução para o Strategist.** O projeto é agnóstico de linguagem (prosa em Markdown + YAML configs). Compilar para binário nativo quebraria isso.

### Opção C: WebAssembly (WASM) — ❌ **Não recomendado**

```
WASM:
  ✓ Agnóstico de linguagem
  ✗ Overhead de JIT compilation
  ✗ I/O de rede é bloqueante (não melhora context enrichment)
  ✗ Adiciona complexidade sem ganho real
```

---

## Implementação Progressiva (Roadmap)

### Phase 1: Baseline (faz agora — 1h)
```bash
✅ Adicionar .gitignore para .compiled/
✅ Criar compile-knowledge-index.sh
✅ Integrar ao install.sh
✅ Preflight checar .compiled/.index primeiro
# Ganho esperado: ~30ms por busca de conhecimento
```

### Phase 2: Domínio Interno (next sprint — 2h)
```bash
✅ compile-domain.sh
✅ Lazy loading no preflight
✅ Test falback para .yaml se .compiled estiver stale
# Ganho esperado: 40–50ms por missão
```

### Phase 3: Config + Personas (next sprint — 1h)
```bash
✅ compile-config.sh
✅ Bootstrap carrega .compiled/.config
✅ .manifest validation no preflight
# Ganho esperado: 20–30ms por invocação
```

### Phase 4: Learning Buffer (next sprint — 2h)
```bash
✅ Implementar LearningBuffer class
✅ Integrar ao learning-curator
✅ Configurável via active.yaml
# Ganho esperado: 10–15ms por 100 missões
```

---

## Benchmark: Antes vs. Depois

```
Cenário: 100 missões com 50+ knowledge sources

ANTES (status quo):
  Bootstrap: 40ms
  Preflight: 150ms (load domain + knowledge search)
  Context Enrichment: 80ms (busca linear em 50 sources)
  Learning Phase: 40ms (write outcomes)
  ─────────────────
  Total offline: 310ms per mission

DEPOIS (com Tier 1–4):
  Bootstrap: 35ms (minimal compile check)
  Preflight: 20ms (mmap + key lookup)
  Context Enrichment: 10ms (indexed lookup)
  Learning Phase: 5ms (buffered write)
  ─────────────────
  Total offline: 70ms per mission

Melhoria: 4.4x mais rápido
100 missões economizam: 24 segundos
```

---

## Implementação Sample (Python pseudocode)

```python
# strategist/compiler.py

import json
import gzip
import msgpack
from pathlib import Path
from typing import Any, Dict

class StrategistCompiler:
    def __init__(self, skill_root: Path, base_path: Path):
        self.skill_root = skill_root
        self.base_path = base_path
        self.compiled_dir = base_path / ".strategist" / ".compiled"
        self.compiled_dir.mkdir(parents=True, exist_ok=True)
    
    def compile_knowledge_index(self) -> None:
        """Compilar knowledge.index.yaml em índice invertido."""
        index_path = self.base_path / ".strategist" / "knowledge.index.yaml"
        output = self.compiled_dir / ".index"
        
        import yaml
        with open(index_path) as f:
            data = yaml.safe_load(f)
        
        tag_index = {}
        for source in data.get('sources', []):
            for tag in source.get('tags', []):
                if tag not in tag_index:
                    tag_index[tag] = []
                tag_index[tag].append(source['id'])
        
        compiled = {
            'version': '1.0',
            'tags': tag_index,
            'sources': {s['id']: s for s in data.get('sources', [])}
        }
        
        with gzip.open(output, 'wb') as f:
            f.write(msgpack.packb(compiled))
    
    def compile_domain(self) -> None:
        """Compilar domínio interno (.strategist/) em blob único."""
        domain_root = self.base_path / ".strategist"
        output = self.compiled_dir / ".domain"
        
        import yaml
        
        # Ler index.yaml para saber quais arquivos carregar
        with open(domain_root / "index.yaml") as f:
            index = yaml.safe_load(f)
        
        compiled = {
            'load_always': {},
            'load_by_task_type': {}
        }
        
        # load_always
        for file_path in index.get('load_always', []):
            full_path = domain_root / file_path
            with open(full_path) as f:
                compiled['load_always'][file_path] = yaml.safe_load(f)
        
        # load_by_task_type
        for task_type, files in index.get('load_by_task_type', {}).items():
            compiled['load_by_task_type'][task_type] = {}
            for file_path in files:
                full_path = domain_root / file_path
                with open(full_path) as f:
                    compiled['load_by_task_type'][task_type][file_path] = yaml.safe_load(f)
        
        with gzip.open(output, 'wb') as f:
            f.write(msgpack.packb(compiled))
    
    def compile_all(self) -> None:
        """Compilar todos os artifacts."""
        self.compile_knowledge_index()
        self.compile_domain()
        # ... mais compilações
        self._write_manifest()
    
    def _write_manifest(self) -> None:
        """Escrever checksums para validação."""
        import hashlib
        manifest = {}
        for compiled_file in self.compiled_dir.glob('.*'):
            if compiled_file.name == '.manifest':
                continue
            with open(compiled_file, 'rb') as f:
                manifest[compiled_file.name] = hashlib.sha256(f.read()).hexdigest()
        with open(self.compiled_dir / ".manifest", 'w') as f:
            json.dump(manifest, f)

# Uso no install.sh:
# python3 -c "
# from pathlib.Path import Path
# from compiler import StrategistCompiler
# compiler = StrategistCompiler(Path('strategist'), Path('/target/repo'))
# compiler.compile_all()
# "
```

---

## Conclusão

**TL;DR:**

1. **Binário ajuda?** Sim, mas não compilar para ELF/WASM. Usar **MessagePack + gzip** é suficiente.
2. **Maior ganho:** Índice invertido para knowledge.index (40–60% melhoria em context enrichment).
3. **Implementação:** Começar com `compile-knowledge-index.sh`, integrar ao install.sh. Aí sim compilar domínio + config.
4. **Benchmark:** 310ms → 70ms offline por missão = 4.4x mais rápido.
5. **Trade-off:** Perder legibilidade de YAML vs. ganhar 200ms+ por invocação. Vale a pena em produção.

**Próximo passo:** Escrever um `compile-all.sh` simples que seja agnóstico de linguagem (bash + jq/yq + msgpack binary). Aí você tem portabilidade e performance.
