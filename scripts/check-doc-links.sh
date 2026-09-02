#!/usr/bin/env bash
# check-doc-links.sh — fails when a markdown internal link under docs/ points
# at a file or directory that does not exist. External links (http/https/
# mailto), bare in-page anchors (#foo), and generated files under
# docs/generated/ (their own content is deterministic and already gated by
# docs-generated-gate) are skipped. See tasks.md T3 acceptance check: "broken
# links ... fail validation" and docs/adr/0025-generated-documentation-anti-drift.md
# for the sibling docs/generated/ determinism gate this complements.
set -euo pipefail

DOCS="docs"
violations=0

is_external_or_anchor() {
  case "$1" in
    http://*|https://*|mailto:*|\#*) return 0 ;;
    *) return 1 ;;
  esac
}

while IFS= read -r f; do
  [ -n "$f" ] || continue
  dir=$(dirname "$f")

  # Extract every markdown link target: "](target)" -> "target"
  targets=$(grep -oE '\]\([^)]+\)' "$f" 2>/dev/null | sed -E 's/^\]\((.*)\)$/\1/' || true)
  [ -n "$targets" ] || continue

  while IFS= read -r target; do
    [ -n "$target" ] || continue
    if is_external_or_anchor "$target"; then
      continue
    fi
    # Strip an in-page anchor fragment ("path.md#section" -> "path.md");
    # a link that is only a fragment was already skipped above.
    path="${target%%#*}"
    [ -n "$path" ] || continue

    resolved="$dir/$path"
    if [ ! -e "$resolved" ]; then
      echo "FAIL: $f: broken link to $target (resolved: $resolved)"
      violations=$((violations + 1))
    fi
  done <<EOF
$targets
EOF
done <<EOF
$(find "$DOCS" -name '*.md' | sort)
EOF

if [ "$violations" -ne 0 ]; then
  echo "FAIL: $violations broken doc link(s) found"
  exit 1
fi

echo "OK: no broken doc links"
