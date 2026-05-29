#!/usr/bin/env sh
# compile-domain.sh <strategist_root> <output.gz>
# Compiles all internal domain files (index.yaml entries) into a single blob.
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

SOURCES_JSON="{}"
add_source() {
  local path="$1"
  local mtime
  mtime=$(get_mtime "$path")
  SOURCES_JSON=$(printf '%s' "$SOURCES_JSON" \
    | jq --arg k "$path" --argjson v "$mtime" '. + {($k): $v}')
}

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
