#!/usr/bin/env sh
# compile-all.sh <strategist_root> <knowledge_index_yaml>
# Orchestrates all compile scripts; writes .manifest.gz only on full success.
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

# Write manifest last — signals a complete compilation run
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
