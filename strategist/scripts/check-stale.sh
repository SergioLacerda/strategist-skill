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
  [ -f "$file" ] || exit 1                        # source gone → stale
  actual=$(get_mtime "$file")
  [ "$actual" -le "$recorded" ] || exit 1         # source newer → stale
done
IFS="$OLD_IFS"

exit 0
