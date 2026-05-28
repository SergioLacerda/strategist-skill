#!/bin/bash
# SPEC Pre-commit Hook Setup

# This script installs the SPEC compliance pre-commit hook
# Run this once to set up compliance checking on every commit

HOOK_FILE=".git/hooks/pre-commit"

if [ -f "$HOOK_FILE" ]; then
    echo "⚠️  $HOOK_FILE already exists"
    echo "Backing up to $HOOK_FILE.backup"
    cp "$HOOK_FILE" "$HOOK_FILE.backup"
fi

cat > "$HOOK_FILE" << 'HOOK_CONTENT'
#!/bin/bash
# Pre-commit hook for SDD compliance validation
# Prevents commits that violate architectural rules

set -e

# Try to run sdd lint
if command -v sdd >/dev/null 2>&1; then
    echo "🔍 Running SDD Lint (Full Pipeline)..."
    sdd lint run --skip-mypy
else
    echo "⚠️  sdd CLI not found. Falling back to basic checks..."

    # Basic check for legacy paths
    if grep -rE "(docs/specs|/runtime/|/REALITY/|/DEVELOPMENT/|sdd-generated)" docs/spec/canonical/ > /dev/null 2>&1; then
        echo "❌ ERROR: Legacy paths detected in docs/spec/canonical/"
        exit 1
    fi
fi

echo "✅ SDD Compliance Check PASSED"
exit 0
HOOK_CONTENT

chmod +x "$HOOK_FILE"
echo "✅ Pre-commit hook installed at $HOOK_FILE"
echo ""
echo "Hook will validate on every commit:"
echo "  • CANONICAL/ has valid paths"
echo "  • No project names in CANONICAL/"
echo "  • ARCHIVE/ not modified"
echo "  • Python files have valid syntax"
