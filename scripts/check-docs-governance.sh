#!/usr/bin/env sh
set -eu

DOCS="docs"
ADR_DIR="$DOCS/adr"
README="$DOCS/README.md"
violations=0

# --- Check 7: docs/README.md must exist ---
if [ ! -f "$README" ]; then
  echo "FAIL: $README does not exist"
  violations=1
fi

# --- Checks 1-2: every docs/adr/*.md must have Status and Date/Data ---
while IFS= read -r f; do
  [ -n "$f" ] || continue
  if ! grep -qE "^\*\*Status:\*\*" "$f" 2>/dev/null; then
    echo "FAIL: $f missing **Status:** field"
    violations=1
  fi
  if ! grep -qE "^\*\*Date:\*\*|^\*\*Data:\*\*" "$f" 2>/dev/null; then
    echo "FAIL: $f missing **Date:** or **Data:** field"
    violations=1
  fi
done <<EOF
$(find "$ADR_DIR" -maxdepth 1 -name '*.md' 2>/dev/null | sort)
EOF

# --- Checks 3-4: every docs/*.md (excl. plans/) must have Status and Date/Last Updated ---
while IFS= read -r f; do
  [ -n "$f" ] || continue
  if ! grep -qE "^\*\*Status:\*\*" "$f" 2>/dev/null; then
    echo "FAIL: $f missing **Status:** field"
    violations=1
  fi
  if ! grep -qE "^\*\*(Date|Last Updated):\*\*" "$f" 2>/dev/null; then
    echo "FAIL: $f missing **Date:** or **Last Updated:** field"
    violations=1
  fi
done <<EOF
$(find "$DOCS" -maxdepth 1 -name '*.md' 2>/dev/null | sort)
EOF

# --- Check 5: no ../ refs in docs/*.md (excl. plans/) or docs/adr/*.md ---
while IFS= read -r f; do
  [ -n "$f" ] || continue
  echo "FAIL: $f contains external cross-reference (../)"
  violations=1
done <<EOF
$(find "$DOCS" -maxdepth 1 -name '*.md' -exec grep -l "\.\.\/" {} \; 2>/dev/null | sort
find "$ADR_DIR" -maxdepth 1 -name '*.md' -exec grep -l "\.\.\/" {} \; 2>/dev/null | sort)
EOF

# --- Check 6: no placeholders in docs/*.md (excl. plans/) or docs/adr/*.md ---
while IFS= read -r line; do
  [ -n "$line" ] || continue
  echo "FAIL: placeholder found: $line"
  violations=1
done <<EOF
$(find "$DOCS" -maxdepth 1 -name '*.md' -exec grep -nE "\bTBD\b|\bWIP\b|\[TBD\]|\bTODO\b" {} /dev/null \; 2>/dev/null
find "$ADR_DIR" -maxdepth 1 -name '*.md' -exec grep -nE "\bTBD\b|\bWIP\b|\[TBD\]|\bTODO\b" {} /dev/null \; 2>/dev/null)
EOF

# --- Checks 8-10: README navigation and language policy ---
if [ -f "$README" ]; then
  # Check 9: docs/adr/ directory referenced in README
  if ! grep -q "adr/" "$README"; then
    echo "FAIL: $README does not reference docs/adr/"
    violations=1
  fi

  # Check 10: language policy declared in README
  if ! grep -qiE "language|idioma" "$README"; then
    echo "FAIL: $README does not declare language policy (missing 'language' or 'idioma')"
    violations=1
  fi

  # Check 8: every docs/*.md (excl. README.md) is referenced in README
  while IFS= read -r f; do
    [ -n "$f" ] || continue
    base=$(basename "$f")
    [ "$base" = "README.md" ] && continue
    if ! grep -q "$base" "$README" 2>/dev/null; then
      echo "FAIL: $f not referenced in $README"
      violations=1
    fi
  done <<EOF
$(find "$DOCS" -maxdepth 1 -name '*.md' 2>/dev/null | sort)
EOF
fi

if [ "$violations" -ne 0 ]; then
  exit 1
fi

echo "OK: docs governance valid"
