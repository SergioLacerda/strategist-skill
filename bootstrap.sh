#!/usr/bin/env bash
# Strategist curl installer — Linux / Mac / WSL
#
# Wizard runs by default. Use --silent to skip interactive setup.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash -s -- --silent
#   curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash -s -- --target=/my/project
#   curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash -s -- --ref=v1.0.0

set -euo pipefail

REPO="SergioLacerda/strategist-skill"
DEFAULT_REF="main"

# ── arg parsing ───────────────────────────────────────────────────────────────

INSTALL_ARGS=(--wizard)   # wizard by default; pass --silent to override
REF=""

for arg in "$@"; do
  case "$arg" in
    --ref=*) REF="${arg#--ref=}" ;;
    --silent) INSTALL_ARGS=() ;;              # opt-out of wizard
    *) INSTALL_ARGS+=("$arg") ;;
  esac
done

# ── resolve version ───────────────────────────────────────────────────────────

resolve_ref() {
  if [ -n "$REF" ]; then
    echo "$REF"
    return
  fi

  local latest
  latest="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
    | grep '"tag_name"' \
    | head -1 \
    | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')" || true

  if [ -n "$latest" ]; then
    echo "$latest"
  else
    echo "[Strategist] No release found, using branch: ${DEFAULT_REF}" >&2
    echo "$DEFAULT_REF"
  fi
}

# ── download and extract ──────────────────────────────────────────────────────

TMPDIR_INSTALL="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_INSTALL"' EXIT

REF="$(resolve_ref)"

# Tag refs get archive from /tags/; branch refs from /heads/
if [[ "$REF" == v* ]]; then
  ARCHIVE_URL="https://github.com/${REPO}/archive/refs/tags/${REF}.tar.gz"
else
  ARCHIVE_URL="https://github.com/${REPO}/archive/refs/heads/${REF}.tar.gz"
fi

echo "[Strategist] Downloading from ${ARCHIVE_URL} ..."
curl -fsSL "$ARCHIVE_URL" -o "${TMPDIR_INSTALL}/strategist.tar.gz"

mkdir -p "${TMPDIR_INSTALL}/extracted"
tar -xzf "${TMPDIR_INSTALL}/strategist.tar.gz" \
  --strip-components=1 \
  -C "${TMPDIR_INSTALL}/extracted"

# ── run install ───────────────────────────────────────────────────────────────

INSTALL_SCRIPT="${TMPDIR_INSTALL}/extracted/strategist/install.sh"

if [ ! -f "$INSTALL_SCRIPT" ]; then
  echo "Error: install.sh not found in downloaded archive." >&2
  exit 1
fi

bash "$INSTALL_SCRIPT" "${INSTALL_ARGS[@]+"${INSTALL_ARGS[@]}"}"
