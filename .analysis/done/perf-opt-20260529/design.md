# Design — Strategist Performance Optimization
**Mission ID:** perf-opt-20260529
**Artifact:** design.md (how)
**Date:** 2026-05-29

---

## 1. Architecture Overview

```
INSTALL TIME (once)                    MISSION TIME (every invocation)
────────────────────                   ────────────────────────────────

  YAML Sources                           Agent reads SKILL.md
  ────────────                           │
  active.yaml       ─┐                  ├─ Bootstrap (§1)
  personas/*.yaml    ├─► compile-config.sh ► .compiled/.config.gz
  roles/*.yaml      ─┘                  │     └─ check-stale.sh → if fresh: jq parse
                                        │     └─ if stale/absent: load YAML directly
  index.yaml                            │
  identity/*.yaml   ─┐                  ├─ Preflight (§2)
  directives/*.yaml  ├─► compile-domain.sh ► .compiled/.domain.gz
  rubrics/*.yaml    ─┘                  │     └─ check-stale.sh → if fresh: jq parse
                                        │     └─ if stale/absent: load YAML directly
  knowledge.index.yaml ──► compile-knowledge-index.sh ► .compiled/.index.gz
                                        │     └─ check-stale.sh → if fresh: jq parse
                                        │     └─ if stale/absent: load YAML directly
                                        │
  compile-all.sh (orchestrator)         ├─ Context Enrichment (§4)
  └─► compile-knowledge-index.sh        │
  └─► compile-domain.sh                 └─ Learning Phase (§8)
  └─► compile-config.sh                       └─ LearningBuffer (shell)
  └─► write .compiled/.manifest.gz                └─ outcomes.tmp → flush → outcomes.jsonl

  install.sh calls compile-all.sh
  .gitignore: .strategist/.compiled/
```

---

## 2. Compiled Artifact Schemas

All artifacts are gzipped JSON files. All contain a `sources` block for stale detection.

### 2.1 `.compiled/.index.gz` — Knowledge Index

```json
{
  "schema": "strategist-compiled-index/1.0",
  "compiled_at": 1748476800,
  "sources": {
    ".strategist/knowledge.index.yaml": 1748470000
  },
  "tags": {
    "architecture": ["arch-docs", "system-overview"],
    "architecture_analysis": ["arch-docs", "system-overview", "patterns"],
    "refactor": ["patterns", "guidelines"],
    "all": ["team-guide"]
  },
  "source_meta": {
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

**Query pattern (shell):**
```sh
# Get sources for task_type "architecture_analysis":
gunzip -c .compiled/.index.gz | jq -r '.tags["architecture_analysis"][]'
```

### 2.2 `.compiled/.domain.gz` — Internal Domain

```json
{
  "schema": "strategist-compiled-domain/1.0",
  "compiled_at": 1748476800,
  "sources": {
    ".strategist/index.yaml": 1748470000,
    ".strategist/identity/what-i-am.yaml": 1748470001,
    ".strategist/identity/drift-patterns.yaml": 1748470002,
    ".strategist/directives/core.yaml": 1748469000,
    ".strategist/directives/by-task/architecture-analysis.yaml": 1748469001,
    ".strategist/rubrics/architecture-analysis.yaml": 1748469002
  },
  "load_always": {
    "identity/what-i-am.yaml": { "...": "full parsed YAML content" },
    "identity/drift-patterns.yaml": { "...": "full parsed YAML content" },
    "directives/core.yaml": { "...": "full parsed YAML content" }
  },
  "load_by_task_type": {
    "architecture_analysis": {
      "directives/by-task/architecture-analysis.yaml": { "...": "full parsed YAML content" },
      "rubrics/architecture-analysis.yaml": { "...": "full parsed YAML content" }
    },
    "refactor": {
      "directives/by-task/architecture-analysis.yaml": { "...": "full parsed YAML content" },
      "rubrics/architecture-analysis.yaml": { "...": "full parsed YAML content" }
    }
  }
}
```

**Query pattern (shell):**
```sh
# Get load_always content:
gunzip -c .compiled/.domain.gz | jq '.load_always'

# Get content for specific task_type:
gunzip -c .compiled/.domain.gz | jq '.load_by_task_type["architecture_analysis"]'
```

### 2.3 `.compiled/.config.gz` — Active Configuration

```json
{
  "schema": "strategist-compiled-config/1.0",
  "compiled_at": 1748476800,
  "sources": {
    ".strategist/active.yaml": 1748470000,
    ".strategist/personas/pragmatic.yaml": 1748470001,
    ".strategist/roles/default.yaml": 1748469000
  },
  "active": {
    "mode": "pragmatic",
    "base_path": ".analysis",
    "roles_config": "default",
    "knowledge_index_path": "knowledge.index.yaml"
  },
  "personas": {
    "pragmatic": { "...": "full persona content" }
  },
  "roles": {
    "default": {
      "discovery": "brainstorming",
      "refinement": "openspec-explore",
      "execution": "sdd-ask"
    }
  }
}
```

**Query pattern (shell):**
```sh
# Get active mode:
gunzip -c .compiled/.config.gz | jq -r '.active.mode'

# Get persona for active mode:
MODE=$(gunzip -c .compiled/.config.gz | jq -r '.active.mode')
gunzip -c .compiled/.config.gz | jq --arg mode "$MODE" '.personas[$mode]'
```

### 2.4 `.compiled/.manifest.gz` — Artifact Checksums

```json
{
  "schema": "strategist-compiled-manifest/1.0",
  "generated_at": 1748476800,
  "artifacts": {
    ".index.gz": "sha256:abcdef1234...",
    ".domain.gz": "sha256:abcdef5678...",
    ".config.gz": "sha256:abcdef9abc..."
  }
}
```

The manifest is written last by `compile-all.sh`. Its presence signals a complete compilation run. An incomplete run (where `compile-all.sh` was killed mid-way) leaves no manifest, and all compiled artifacts are treated as absent.

---

## 3. Shell Scripts

### 3.1 `check-stale.sh`

**Purpose:** Determine if a compiled artifact is fresh relative to its source files.  
**Contract:**
- Input: `$1` = path to compiled artifact (`.gz` file)
- Output: exit code `0` = fresh, `1` = stale or absent
- Side effects: none (read-only)

```sh
#!/usr/bin/env sh
# check-stale.sh <artifact.gz>
# Returns 0 if fresh, 1 if stale or absent.
set -eu

ARTIFACT="${1:?Usage: check-stale.sh <artifact.gz>}"

# Absent → stale
[ -f "$ARTIFACT" ] || exit 1

# Manifest absent → treat all as stale
MANIFEST_DIR=$(dirname "$ARTIFACT")
[ -f "$MANIFEST_DIR/.manifest.gz" ] || exit 1

# Cross-platform mtime helper
get_mtime() {
  stat -c %Y "$1" 2>/dev/null || stat -f %m "$1" 2>/dev/null || { echo 0; }
}

# Read sources block from artifact
SOURCES=$(gunzip -c "$ARTIFACT" | jq -r '.sources | to_entries[] | "\(.key)\t\(.value)"') || exit 1

# Compare each source's current mtime to recorded mtime
OLD_IFS="$IFS"
IFS='
'
for entry in $SOURCES; do
  file=$(printf '%s' "$entry" | cut -f1)
  recorded=$(printf '%s' "$entry" | cut -f2)
  [ -f "$file" ] || exit 1                          # source gone → stale
  actual=$(get_mtime "$file")
  [ "$actual" -le "$recorded" ] || exit 1           # source newer → stale
done
IFS="$OLD_IFS"

exit 0
```

### 3.2 `compile-knowledge-index.sh`

**Purpose:** Build an inverted tag index from `knowledge.index.yaml`.  
**Contract:**
- Input: `$1` = path to `knowledge.index.yaml`, `$2` = output path (`.compiled/.index.gz`)
- Output: gzipped JSON at `$2`; exit code `0` on success, non-zero on failure
- Idempotent: yes (re-running overwrites with identical content if sources unchanged)
- Requires: `yq` (YAML→JSON), `jq`, `gzip`, POSIX `stat`

```sh
#!/usr/bin/env sh
# compile-knowledge-index.sh <knowledge_index_yaml> <output.gz>
set -eu

KNOWLEDGE_INDEX="${1:?Usage: compile-knowledge-index.sh <knowledge.index.yaml> <output.gz>}"
OUTPUT="${2:?Usage: compile-knowledge-index.sh <knowledge.index.yaml> <output.gz>}"

[ -f "$KNOWLEDGE_INDEX" ] || { echo "ERROR: $KNOWLEDGE_INDEX not found" >&2; exit 1; }

get_mtime() {
  stat -c %Y "$1" 2>/dev/null || stat -f %m "$1" 2>/dev/null || echo 0
}

COMPILED_AT=$(date +%s)
SOURCE_MTIME=$(get_mtime "$KNOWLEDGE_INDEX")
ABS_SOURCE=$(cd "$(dirname "$KNOWLEDGE_INDEX")" && pwd)/$(basename "$KNOWLEDGE_INDEX")

mkdir -p "$(dirname "$OUTPUT")"

yq -o json "$KNOWLEDGE_INDEX" \
  | jq \
      --argjson compiled_at "$COMPILED_AT" \
      --arg source_key "$ABS_SOURCE" \
      --argjson source_mtime "$SOURCE_MTIME" \
      '
        . as $data |
        reduce ($data.sources // [])[] as $src (
          {};
          . as $acc |
          ($src.tags // []) as $tags |
          reduce $tags[] as $tag (
            $acc;
            .[$tag] += [$src.id]
          )
        ) as $tag_index |
        {
          schema: "strategist-compiled-index/1.0",
          compiled_at: $compiled_at,
          sources: { ($source_key): $source_mtime },
          tags: $tag_index,
          source_meta: (($data.sources // []) | map({(.id): .}) | add // {})
        }
      ' \
  | gzip > "$OUTPUT"

echo "[compile-knowledge-index] done → $OUTPUT"
```

### 3.3 `compile-domain.sh`

**Purpose:** Compile all internal domain files into a single blob.  
**Contract:**
- Input: `$1` = `.strategist/` root dir, `$2` = output path (`.compiled/.domain.gz`)
- Output: gzipped JSON at `$2`
- Idempotent: yes
- Requires: `yq`, `jq`, `gzip`, POSIX `stat`

```sh
#!/usr/bin/env sh
# compile-domain.sh <strategist_root> <output.gz>
set -eu

STRATEGIST_ROOT="${1:?Usage: compile-domain.sh <strategist_root> <output.gz>}"
OUTPUT="${2:?Usage: compile-domain.sh <strategist_root> <output.gz>}"

INDEX="$STRATEGIST_ROOT/index.yaml"
[ -f "$INDEX" ] || { echo "ERROR: $INDEX not found" >&2; exit 1; }

get_mtime() {
  stat -c %Y "$1" 2>/dev/null || stat -f %m "$1" 2>/dev/null || echo 0
}

COMPILED_AT=$(date +%s)
mkdir -p "$(dirname "$OUTPUT")"

# Build sources manifest
SOURCES_JSON="{}"
add_source() {
  local path="$1"
  local mtime
  mtime=$(get_mtime "$path")
  SOURCES_JSON=$(printf '%s' "$SOURCES_JSON" \
    | jq --arg k "$path" --argjson v "$mtime" '. + {($k): $v}')
}

# Read index.yaml to determine which files to load
LOAD_ALWAYS_FILES=$(yq -o json "$INDEX" | jq -r '.load_always[]' 2>/dev/null || true)
TASK_TYPES=$(yq -o json "$INDEX" | jq -r '.load_by_task_type | keys[]' 2>/dev/null || true)

add_source "$INDEX"

# Compile load_always
LOAD_ALWAYS_JSON="{}"
for rel_path in $LOAD_ALWAYS_FILES; do
  full_path="$STRATEGIST_ROOT/$rel_path"
  [ -f "$full_path" ] || { echo "WARN: $full_path not found, skipping" >&2; continue; }
  add_source "$full_path"
  content=$(yq -o json "$full_path")
  LOAD_ALWAYS_JSON=$(printf '%s' "$LOAD_ALWAYS_JSON" \
    | jq --arg k "$rel_path" --argjson v "$content" '. + {($k): $v}')
done

# Compile load_by_task_type
TASK_TYPE_JSON="{}"
for task_type in $TASK_TYPES; do
  FILES=$(yq -o json "$INDEX" | jq -r --arg tt "$task_type" '.load_by_task_type[$tt][]' 2>/dev/null || true)
  TYPE_FILES_JSON="{}"
  for rel_path in $FILES; do
    full_path="$STRATEGIST_ROOT/$rel_path"
    [ -f "$full_path" ] || { echo "WARN: $full_path not found, skipping" >&2; continue; }
    add_source "$full_path"
    content=$(yq -o json "$full_path")
    TYPE_FILES_JSON=$(printf '%s' "$TYPE_FILES_JSON" \
      | jq --arg k "$rel_path" --argjson v "$content" '. + {($k): $v}')
  done
  TASK_TYPE_JSON=$(printf '%s' "$TASK_TYPE_JSON" \
    | jq --arg tt "$task_type" --argjson v "$TYPE_FILES_JSON" '. + {($tt): $v}')
done

# Write final blob
jq -n \
  --argjson compiled_at "$COMPILED_AT" \
  --argjson sources "$SOURCES_JSON" \
  --argjson load_always "$LOAD_ALWAYS_JSON" \
  --argjson load_by_task_type "$TASK_TYPE_JSON" \
  '{
    schema: "strategist-compiled-domain/1.0",
    compiled_at: $compiled_at,
    sources: $sources,
    load_always: $load_always,
    load_by_task_type: $load_by_task_type
  }' \
| gzip > "$OUTPUT"

echo "[compile-domain] done → $OUTPUT"
```

### 3.4 `compile-config.sh`

**Purpose:** Compile `active.yaml`, resolved persona, and resolved roles into a single blob.  
**Contract:**
- Input: `$1` = `.strategist/` root dir, `$2` = output path (`.compiled/.config.gz`)
- Output: gzipped JSON at `$2`
- Idempotent: yes
- Requires: `yq`, `jq`, `gzip`, POSIX `stat`

```sh
#!/usr/bin/env sh
# compile-config.sh <strategist_root> <output.gz>
set -eu

STRATEGIST_ROOT="${1:?Usage: compile-config.sh <strategist_root> <output.gz>}"
OUTPUT="${2:?Usage: compile-config.sh <strategist_root> <output.gz>}"

ACTIVE_YAML="$STRATEGIST_ROOT/active.yaml"
[ -f "$ACTIVE_YAML" ] || { echo "ERROR: $ACTIVE_YAML not found" >&2; exit 1; }

get_mtime() {
  stat -c %Y "$1" 2>/dev/null || stat -f %m "$1" 2>/dev/null || echo 0
}

COMPILED_AT=$(date +%s)
mkdir -p "$(dirname "$OUTPUT")"

SOURCES_JSON="{}"
add_source() {
  local path="$1"
  local mtime
  mtime=$(get_mtime "$path")
  SOURCES_JSON=$(printf '%s' "$SOURCES_JSON" \
    | jq --arg k "$path" --argjson v "$mtime" '. + {($k): $v}')
}

# Parse active.yaml
ACTIVE_JSON=$(yq -o json "$ACTIVE_YAML")
add_source "$ACTIVE_YAML"

MODE=$(printf '%s' "$ACTIVE_JSON" | jq -r '.mode // "pragmatic"')
ROLES_CONFIG=$(printf '%s' "$ACTIVE_JSON" | jq -r '.roles_config // "default"')

# Compile all personas
PERSONAS_JSON="{}"
for persona_file in "$STRATEGIST_ROOT/personas/"*.yaml; do
  [ -f "$persona_file" ] || continue
  add_source "$persona_file"
  persona_name=$(basename "$persona_file" .yaml)
  content=$(yq -o json "$persona_file")
  PERSONAS_JSON=$(printf '%s' "$PERSONAS_JSON" \
    | jq --arg k "$persona_name" --argjson v "$content" '. + {($k): $v}')
done

# Compile all roles
ROLES_JSON="{}"
for role_file in "$STRATEGIST_ROOT/roles/"*.yaml; do
  [ -f "$role_file" ] || continue
  add_source "$role_file"
  role_name=$(basename "$role_file" .yaml)
  content=$(yq -o json "$role_file")
  ROLES_JSON=$(printf '%s' "$ROLES_JSON" \
    | jq --arg k "$role_name" --argjson v "$content" '. + {($k): $v}')
done

jq -n \
  --argjson compiled_at "$COMPILED_AT" \
  --argjson sources "$SOURCES_JSON" \
  --argjson active "$ACTIVE_JSON" \
  --argjson personas "$PERSONAS_JSON" \
  --argjson roles "$ROLES_JSON" \
  '{
    schema: "strategist-compiled-config/1.0",
    compiled_at: $compiled_at,
    sources: $sources,
    active: $active,
    personas: $personas,
    roles: $roles
  }' \
| gzip > "$OUTPUT"

echo "[compile-config] done → $OUTPUT"
```

### 3.5 `compile-all.sh`

**Purpose:** Orchestrate all compile scripts; write `.manifest.gz`; called by `install.sh`.  
**Contract:**
- Input: `$1` = `.strategist/` root dir, `$2` = `knowledge.index.yaml` path
- Output: all `.compiled/*.gz` artifacts + `.manifest.gz`; exit code `0` on full success
- Idempotent: yes (re-running overwrites stale artifacts)
- Error behavior: any compile step failure causes `compile-all.sh` to exit non-zero; no partial manifest is written

```sh
#!/usr/bin/env sh
# compile-all.sh <strategist_root> <knowledge_index_yaml>
set -eu

SKILL_ROOT="${1:?Usage: compile-all.sh <strategist_root> <knowledge_index_yaml>}"
KNOWLEDGE_INDEX="${2:?Usage: compile-all.sh <strategist_root> <knowledge_index_yaml>}"
COMPILED_DIR="$SKILL_ROOT/.compiled"
SCRIPTS_DIR="$(dirname "$0")"

mkdir -p "$COMPILED_DIR"

echo "[compile-all] compiling knowledge index..."
sh "$SCRIPTS_DIR/compile-knowledge-index.sh" \
  "$KNOWLEDGE_INDEX" \
  "$COMPILED_DIR/.index.gz"

echo "[compile-all] compiling internal domain..."
sh "$SCRIPTS_DIR/compile-domain.sh" \
  "$SKILL_ROOT" \
  "$COMPILED_DIR/.domain.gz"

echo "[compile-all] compiling config..."
sh "$SCRIPTS_DIR/compile-config.sh" \
  "$SKILL_ROOT" \
  "$COMPILED_DIR/.config.gz"

# Write manifest (last — signals complete run)
echo "[compile-all] writing manifest..."
GENERATED_AT=$(date +%s)

sha256_file() {
  sha256sum "$1" 2>/dev/null | cut -d' ' -f1 \
  || shasum -a 256 "$1" 2>/dev/null | cut -d' ' -f1 \
  || echo "unavailable"
}

INDEX_SHA=$(sha256_file "$COMPILED_DIR/.index.gz")
DOMAIN_SHA=$(sha256_file "$COMPILED_DIR/.domain.gz")
CONFIG_SHA=$(sha256_file "$COMPILED_DIR/.config.gz")

jq -n \
  --argjson generated_at "$GENERATED_AT" \
  --arg index_sha "sha256:$INDEX_SHA" \
  --arg domain_sha "sha256:$DOMAIN_SHA" \
  --arg config_sha "sha256:$CONFIG_SHA" \
  '{
    schema: "strategist-compiled-manifest/1.0",
    generated_at: $generated_at,
    artifacts: {
      ".index.gz": $index_sha,
      ".domain.gz": $domain_sha,
      ".config.gz": $config_sha
    }
  }' \
| gzip > "$COMPILED_DIR/.manifest.gz"

echo "[compile-all] done. artifacts in $COMPILED_DIR"
```

---

## 4. Agent Read Path — SKILL.md Instructions

The Strategist is an LLM agent, not a compiled program. The "read path" means updated SKILL.md instructions that tell the agent to check for compiled artifacts before loading individual YAML files. The agent uses its `Bash` tool to run shell commands.

### 4.1 Bootstrap §1 Update

Replace the current §1 with:

```
## 1. Bootstrap

On every invocation, before any other action:

**Fast path (if compiled artifacts present and fresh):**

Run the following check:
  sh .strategist/scripts/check-stale.sh .strategist/.compiled/.config.gz

If exit code is 0 (fresh):
  Load configuration by running:
    gunzip -c .strategist/.compiled/.config.gz
  Parse the JSON. Extract:
    - active: use as active.yaml content
    - personas[active.mode]: use as persona content
    - roles[active.roles_config]: use as roles content
  Skip steps 1–4 below. Proceed directly to step 5.

**Standard path (fallback):**

1. Load active.yaml from the skill root.
2. Resolve persona: load personas/<active.yaml.mode>.yaml.
   [... rest of existing §1 unchanged ...]
```

### 4.2 Preflight §2a/2b Update

Replace §2a and §2b with:

```
**2a. Load internal domain**

**Fast path (if compiled artifacts present and fresh):**

Run the following check:
  sh .strategist/scripts/check-stale.sh .strategist/.compiled/.domain.gz

If exit code is 0 (fresh):
  Load domain by running:
    gunzip -c .strategist/.compiled/.domain.gz
  Parse the JSON. Extract:
    - load_always: contains all always-loaded files pre-parsed
    - load_by_task_type[task_type]: contains task-type-specific files pre-parsed
  Skip the individual file reads in §2a and §2b. Proceed to §2c.

**Standard path (fallback):**

Load <base_path>/.strategist/index.yaml. [... existing §2a text ...]

**2b. Load identity files**

[Only executed on standard path — skip if fast path succeeded.]
```

### 4.3 Context Enrichment §4 Update

```
**Context Enrichment — Fast Path:**

Before scanning knowledge.index.yaml:

Run:
  sh .strategist/scripts/check-stale.sh .strategist/.compiled/.index.gz

If exit code is 0 (fresh):
  Query the inverted index directly:
    gunzip -c .strategist/.compiled/.index.gz | jq -r '.tags["<task_type>"][]'
  This returns source IDs matching task_type in O(1). No linear scan needed.
  Retrieve source metadata:
    gunzip -c .strategist/.compiled/.index.gz | jq '.source_meta["<source_id>"]'
  Proceed with enrichment using the retrieved sources.

**Standard path (fallback):**

Load knowledge.index.yaml and filter sources by tag. [... existing §4 text ...]
```

### 4.4 Observability (Agent Log Lines)

Add to each phase transition in SKILL.md:

```
[Strategist] bootstrap=fast_path      # compiled .config.gz used
[Strategist] bootstrap=standard_path  # YAML fallback used
[Strategist] preflight=fast_path      # compiled .domain.gz used
[Strategist] preflight=standard_path  # YAML fallback used
[Strategist] context_enrichment=fast_path   # compiled .index.gz used
[Strategist] context_enrichment=standard_path  # YAML fallback used
```

These log lines allow measuring cache hit rate in production without additional tooling.

---

## 5. install.sh Integration

**Execution order within install.sh:**

```sh
# --- [existing] validate required dependencies ---
command -v yq >/dev/null 2>&1 || { echo "ERROR: yq is required" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "ERROR: jq is required" >&2; exit 1; }
command -v gzip >/dev/null 2>&1 || { echo "ERROR: gzip is required" >&2; exit 1; }

# --- [new] compile artifacts (optional, non-blocking) ---
echo "[install] compiling Strategist artifacts..."
if sh "$SKILL_ROOT/scripts/compile-all.sh" \
     "$TARGET_REPO/.strategist" \
     "$TARGET_REPO/.strategist/knowledge.index.yaml"; then
  echo "[install] compilation complete — fast path enabled"
else
  echo "[install] WARN: compilation failed — YAML fallback will be used (no action needed)"
fi

# --- [new] ensure .gitignore entry ---
GITIGNORE="$TARGET_REPO/.gitignore"
GITIGNORE_ENTRY=".strategist/.compiled/"
if [ -f "$GITIGNORE" ]; then
  grep -qF "$GITIGNORE_ENTRY" "$GITIGNORE" \
    || printf '\n# Strategist compiled artifacts (generated, do not commit)\n%s\n' \
         "$GITIGNORE_ENTRY" >> "$GITIGNORE"
else
  printf '# Strategist compiled artifacts (generated, do not commit)\n%s\n' \
    "$GITIGNORE_ENTRY" > "$GITIGNORE"
fi
echo "[install] .gitignore updated"

# --- [existing install steps continue] ---
```

**Re-install behavior:** `compile-all.sh` is always re-run on install. This is safe because all scripts are idempotent. If sources are unchanged, output is identical.

---

## 6. LearningBuffer — Shell Implementation

Replaces the Python `LearningBuffer` class. No language runtime required.

**Mechanism:** outcomes are written to a temp file. At the START of each mission, if the temp file has ≥ `buffer_size` lines, it is flushed to `outcomes.jsonl`. This ensures flush happens at a predictable, non-crash-sensitive point.

**Files:**
- `.strategist/memory/outcomes.jsonl` — permanent store (append-only)
- `.strategist/memory/outcomes.tmp` — buffer (cleared on flush)

**SKILL.md §8 (Learning Phase) — updated instructions:**

```
## 8. Learning Phase (non-blocking)

After mission completes:

[... existing: invoke response-critic, learning-curator ...]

**LearningBuffer write procedure:**

1. Append the mission outcome JSON line to:
   .strategist/memory/outcomes.tmp

2. At the START of the next mission (before Bootstrap §1):
   a. Count lines in outcomes.tmp:
        sh -c 'wc -l < .strategist/memory/outcomes.tmp 2>/dev/null || echo 0'
   b. If count >= 20 (configurable via active.yaml learning_buffer_size, default 20):
        cat .strategist/memory/outcomes.tmp >> .strategist/memory/outcomes.jsonl
        : > .strategist/memory/outcomes.tmp
        emit: [Strategist] learning_buffer=flushed count=<N>

**Rollback:** To force-flush the buffer manually, run:
  cat .strategist/memory/outcomes.tmp >> .strategist/memory/outcomes.jsonl
  : > .strategist/memory/outcomes.tmp
```

**Why flush at mission START not END:** If the agent crashes mid-mission, the previous mission's outcome is already safe in `outcomes.tmp`. The next mission's START is always reached before any new writes, making flush deterministic and crash-safe.

---

## 7. Windows Compatibility

The compile scripts use POSIX `sh`, `jq`, `gzip`, `yq`, and `stat`. On Windows:

| Tool | Windows availability | Recommended source |
|------|---------------------|-------------------|
| `sh` | Via Git for Windows (Git Bash) or WSL | Install Git for Windows |
| `jq` | Native Windows binary available | [jqlang.github.io/jq](https://jqlang.github.io/jq/) |
| `yq` | Native Windows binary available | [mikefarah/yq](https://github.com/mikefarah/yq/releases) |
| `gzip` | Via Git for Windows or WSL | Install Git for Windows |
| `stat` | POSIX stat available in Git Bash | Install Git for Windows |

**Primary Windows path:** Run `compile-all.sh` via Git Bash (ships with Git for Windows). This is the recommended approach — no additional scripts needed.

**Alternative PowerShell script:** A `compile-all.ps1` should be provided for environments without Git Bash:

```powershell
# compile-all.ps1 — Windows-native alternative
# Requires: PowerShell 7+, jq.exe in PATH, yq.exe in PATH

param(
  [Parameter(Mandatory)][string]$StrategistRoot,
  [Parameter(Mandatory)][string]$KnowledgeIndex
)

$CompiledDir = Join-Path $StrategistRoot ".compiled"
New-Item -ItemType Directory -Force -Path $CompiledDir | Out-Null

function Get-MtimeUnix($path) {
  [DateTimeOffset]::new(
    (Get-Item $path).LastWriteTimeUtc
  ).ToUnixTimeSeconds()
}

function Write-GzJson($obj, $outPath) {
  $json = $obj | ConvertTo-Json -Depth 20 -Compress
  $bytes = [System.Text.Encoding]::UTF8.GetBytes($json)
  $stream = [System.IO.File]::Create($outPath)
  $gz = [System.IO.Compression.GZipStream]::new(
    $stream, [System.IO.Compression.CompressionMode]::Compress)
  $gz.Write($bytes, 0, $bytes.Length)
  $gz.Close(); $stream.Close()
}

# [... knowledge index, domain, config compilation using same JSON schemas ...]
# Full implementation deferred to implementation task.

Write-Host "[compile-all] done (PowerShell)"
```

The `compile-all.ps1` must produce artifacts with identical JSON schemas to the POSIX scripts. The Sniper implementation task for Windows compatibility should write the full PowerShell script.

---

## 8. Rollback Procedure

To revert to the YAML-only path (no compiled artifacts):

```sh
rm -rf .strategist/.compiled/
```

The fallback contract guarantees that all read paths work correctly without `.compiled/`. No code change is required. The agent will emit `standard_path` log lines instead of `fast_path`.

To re-enable after rollback:

```sh
sh .strategist/scripts/compile-all.sh .strategist .strategist/knowledge.index.yaml
```

---

## 9. Stale Detection Summary

| Scenario | Detection | Behavior |
|----------|-----------|----------|
| `.compiled/` absent | `check-stale.sh` → exit 1 (file not found) | Standard path |
| `.manifest.gz` absent (incomplete compile) | `check-stale.sh` → exit 1 | Standard path |
| One YAML source newer than compiled_at | `check-stale.sh` → exit 1 (mtime mismatch) | Standard path |
| New YAML file added (not in sources block) | Not detected by mtime — **requires recompile** | Standard path if manifest absent; otherwise stale data served |
| Corrupt `.gz` file | `jq` parse error → agent falls back | Standard path |

**Known limitation:** Adding a new file to `personas/` or `roles/` without running `compile-all.sh` will not be detected by `check-stale.sh` because the new file is not in the `sources` block of the existing compiled artifact. Documentation in `install.sh` must note: "run install.sh after adding new persona or role files."
