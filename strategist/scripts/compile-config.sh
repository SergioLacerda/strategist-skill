#!/usr/bin/env sh
# compile-config.sh <strategist_root> <output.gz>
# Compiles active.yaml, all personas, and all roles into a single blob.
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

ACTIVE_JSON=$(yq -o json "$ACTIVE_YAML")
add_source "$ACTIVE_YAML"

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
