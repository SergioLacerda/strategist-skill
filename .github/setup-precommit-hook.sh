#!/usr/bin/env bash
# Installs a Go pre-commit hook for strategist-skill.
# Run once from the repo root: bash .github/setup-precommit-hook.sh

set -euo pipefail

HOOK_FILE=".git/hooks/pre-commit"

if [ ! -d ".git" ]; then
    echo "error: must be run from the repository root" >&2
    exit 1
fi

if [ -f "$HOOK_FILE" ]; then
    echo "backing up existing hook to $HOOK_FILE.backup"
    cp "$HOOK_FILE" "$HOOK_FILE.backup"
fi

cat > "$HOOK_FILE" << 'HOOK'
#!/usr/bin/env bash
# Pre-commit hook — strategist-skill
# Gates: gofmt, go vet, go build, golangci-lint
# Tests and govulncheck are intentionally left to CI (too slow for pre-commit).

set -euo pipefail

fail=0

# 1. Format
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
    echo "pre-commit: gofmt issues in:" >&2
    echo "$unformatted" | sed 's/^/  /' >&2
    echo "  run: gofmt -w ." >&2
    fail=1
fi

# 2. Vet
if ! go vet ./... 2>&1; then
    echo "pre-commit: go vet failed" >&2
    fail=1
fi

# 3. Build
if ! go build ./... 2>&1; then
    echo "pre-commit: go build failed" >&2
    fail=1
fi

# 4. Lint (optional — skipped if golangci-lint is not installed)
GOLANGCI_LINT=$(command -v golangci-lint 2>/dev/null \
    || command -v "$(go env GOPATH)/bin/golangci-lint" 2>/dev/null \
    || true)

if [ -n "$GOLANGCI_LINT" ]; then
    if ! "$GOLANGCI_LINT" run ./... 2>&1; then
        echo "pre-commit: golangci-lint failed" >&2
        fail=1
    fi
else
    echo "pre-commit: golangci-lint not found — skipping lint (install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest)"
fi

if [ "$fail" -ne 0 ]; then
    echo "pre-commit: commit blocked — fix the issues above" >&2
    exit 1
fi

echo "pre-commit: all checks passed"
HOOK

chmod +x "$HOOK_FILE"
echo "pre-commit hook installed at $HOOK_FILE"
echo ""
echo "Gates:"
echo "  • gofmt -l ."
echo "  • go vet ./..."
echo "  • go build ./..."
echo "  • golangci-lint run ./...  (skipped if not installed)"
echo ""
echo "To skip in an emergency: git commit --no-verify"
