#!/usr/bin/env sh
# compile-knowledge-index.sh <knowledge_index_yaml> <output.gz>
# Builds an inverted tag index from knowledge.index.yaml.
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
