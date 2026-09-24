#!/usr/bin/env bash
# Pre-commit step: fail when the embedded plugin catalog has drifted from its
# sources. Any Go edit changes the catalog's host_api_digest, so a commit that
# touches Go, external-skills-source/ or the embedded defaults without running
# `make embed-skills` would fail the drift test later, in CI. The check only
# runs when such a file is staged, so docs-only commits stay fast.
#
# Test knobs: STAGED_FILES (newline separated; default: the staged files) and
# EMBED_CHECK_CMD (default: the real drift check).
set -uo pipefail
cd "$(dirname "$0")/../.." || exit 1

staged="${STAGED_FILES-}"
if [[ -z "${STAGED_FILES+x}" ]]; then
  staged="$(git diff --cached --name-only 2>/dev/null || true)"
fi

if ! grep -qE '^(cmd/.*\.go|internal/.*\.go|external-skills-source/|internal/embed/defaults/)' <<<"${staged}"; then
  echo "pre-commit: embedded catalog drift check skipped (no Go or embedded files staged)"
  exit 0
fi

check="${EMBED_CHECK_CMD:-go run ./cmd/strategist plugins prepare-embedded --check}"
if ! bash -c "${check}" >/dev/null 2>&1; then
  echo "pre-commit: the embedded catalog has drifted from its sources (a Go or embedded-file change alters host_api_digest)." >&2
  echo "  run: make embed-skills   and stage the regenerated catalog, mirrors and external-skills-source.lock.yaml" >&2
  exit 1
fi
echo "pre-commit: embedded catalog is current"
