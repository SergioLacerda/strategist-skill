#!/usr/bin/env bash
# Strategist curl installer — Linux / macOS / WSL
#
# Downloads the strategist binary, verifies its SHA256 checksum, and runs
# `strategist install` to set up the skill in the current directory.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash
#   curl -fsSL .../bootstrap.sh | bash -s -- --silent
#   curl -fsSL .../bootstrap.sh | bash -s -- --target=/my/project

set -euo pipefail

REPO="SergioLacerda/strategist-skill"
VERSION="${STRATEGIST_VERSION:-latest}"
INSTALL_DIR="${HOME}/.local/bin"
SILENT=false
TARGET=""

for arg in "$@"; do
  case "$arg" in
    --silent) SILENT=true ;;
    --target=*) TARGET="${arg#--target=}" ;;
    --version=*) VERSION="${arg#--version=}" ;;
  esac
done

# ── detect platform ───────────────────────────────────────────────────────────

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "[Strategist] ERROR: unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# ── resolve version ───────────────────────────────────────────────────────────

if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | cut -d'"' -f4)
fi

BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
BIN_NAME="strategist-${OS}-${ARCH}"

# ── download + verify ─────────────────────────────────────────────────────────

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "[Strategist] Downloading ${BIN_NAME} ${VERSION}..."
curl -fsSL "${BASE_URL}/${BIN_NAME}" -o "${TMP}/strategist"
curl -fsSL "${BASE_URL}/SHA256SUMS" -o "${TMP}/SHA256SUMS"

(cd "$TMP" && grep "$BIN_NAME" SHA256SUMS | sha256sum --check --status)
echo "[Strategist] Checksum verified."

# ── install binary ────────────────────────────────────────────────────────────

mkdir -p "$INSTALL_DIR"
install -m 755 "${TMP}/strategist" "${INSTALL_DIR}/strategist"
echo "[Strategist] Binary installed → ${INSTALL_DIR}/strategist"

export PATH="${INSTALL_DIR}:${PATH}"

# ── run install ───────────────────────────────────────────────────────────────

INSTALL_ARGS="--silent"
[ "$SILENT" = "false" ] && INSTALL_ARGS="--wizard"
[ -n "$TARGET" ] && INSTALL_ARGS="$INSTALL_ARGS --target=${TARGET}"

strategist install $INSTALL_ARGS
