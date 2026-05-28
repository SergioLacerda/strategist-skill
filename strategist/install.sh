#!/usr/bin/env bash
# Strategist install script.
# Silent by default: generates active.yaml from pragmatic-standalone template.
# --wizard flag enables TUI for interactive setup.
# Usage:
#   sh install.sh                    # silent install with pragmatic defaults
#   sh install.sh --wizard           # interactive TUI setup
#   sh install.sh --target /path     # set target repo root (default: current dir)
#
# To uninstall agent shims: rm -rf ~/.claude/skills/strategist (and equivalent for other agents)

set -euo pipefail

SKILL_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_REPO="${PWD}"
WIZARD=false

for arg in "$@"; do
  case "$arg" in
    --wizard) WIZARD=true ;;
    --target=*) TARGET_REPO="${arg#--target=}" ;;
  esac
done

# ── helpers ──────────────────────────────────────────────────────────────────

read_template() {
  cat "${SKILL_ROOT}/templates/$1"
}

write_active_yaml() {
  local template_name="$1"
  read_template "$template_name" > "${TARGET_REPO}/.strategist/active.yaml"
  echo "[Strategist] active.yaml created from template: $template_name"
}

copy_skill_runtime() {
  local dest="${TARGET_REPO}/.strategist"
  mkdir -p "${dest}/memory"

  for f in SKILL.md knowledge.index.yaml protocol.md; do
    [ -f "${SKILL_ROOT}/${f}" ] && cp "${SKILL_ROOT}/${f}" "${dest}/${f}"
  done

  for d in personas roles schemas; do
    [ -d "${SKILL_ROOT}/${d}" ] && cp -r "${SKILL_ROOT}/${d}" "${dest}/${d}"
  done

  # Copy internal domain templates — only if index.yaml not already present
  local domain_src="${SKILL_ROOT}/templates/domain"
  if [ -d "$domain_src" ] && [ ! -f "${dest}/index.yaml" ]; then
    cp -r "${domain_src}/." "${dest}/"
  fi

  echo "[Strategist] runtime installed at: ${dest}"
}

install_agent_shims() {
  local skill_root_abs
  skill_root_abs="$(cd "${TARGET_REPO}/.strategist" && pwd)"

  local description
  description="$(awk '/^description:/{found=1; next} found && /^  /{sub(/^ +/,""); printf "%s ", $0; next} found{exit}' \
    "${SKILL_ROOT}/skill.yaml" | tr -s ' ' | sed 's/ *$//' | cut -c1-120)"
  [ -z "$description" ] && description="Multi-phase mission orchestrator."

  local shim_content
  shim_content="$(cat <<SHIM
---
name: strategist
description: "${description}"
skill_root: ${skill_root_abs}
---

# Strategist

**SKILL_ROOT:** \`${skill_root_abs}\`

Read full instructions from: \`${skill_root_abs}/SKILL.md\`

All config paths (active.yaml, personas/, roles/, schemas/) resolve from skill_root.
SHIM
)"

  local targets=(
    "${HOME}/.claude/skills"
    "${HOME}/.gemini/skills"
    "${HOME}/.gemini/antigravity/skills"
    "${HOME}/.codex/skills"
  )

  for base in "${targets[@]}"; do
    if [ -d "$base" ]; then
      mkdir -p "${base}/strategist"
      printf '%s\n' "$shim_content" > "${base}/strategist/SKILL.md"
      echo "[Strategist] shim registered: ${base}/strategist/SKILL.md"
    else
      echo "[Strategist] skipped (dir not found): ${base}"
    fi
  done
}

scaffold_workspace() {
  local base_path="$1"
  local target="${TARGET_REPO}/${base_path}"
  mkdir -p "${target}/todo" "${target}/pending" "${target}/refined" "${target}/done"
  echo "[Strategist] workspace directories ensured at: ${target}"
}

# ── wizard TUI ───────────────────────────────────────────────────────────────

run_wizard() {
  echo ""
  echo "Strategist Setup Wizard"
  echo "─────────────────────────────────────────────"

  # Template selection
  echo ""
  echo "Choose a configuration template:"
  echo "  1) pragmatic-standalone  — analytical tone, standalone (default)"
  echo "  2) epic-standalone       — strategic commander tone, standalone"
  echo "  3) epic-sdd              — epic tone, SDD integration"
  echo ""
  printf "Template [1]: "
  read -r template_choice
  template_choice="${template_choice:-1}"

  case "$template_choice" in
    2) TEMPLATE="epic-standalone.yaml" ;;
    3) TEMPLATE="epic-sdd.yaml" ;;
    *) TEMPLATE="pragmatic-standalone.yaml" ;;
  esac

  # Base path
  printf "Mission workspace base path [.analysis]: "
  read -r base_path
  base_path="${base_path:-.analysis}"

  # Scout provider
  printf "Scout (discovery) provider [sdd-diagnose]: "
  read -r scout
  scout="${scout:-sdd-diagnose}"

  # Engineer provider
  printf "Engineer (refinement) provider [engineer]: "
  read -r engineer
  engineer="${engineer:-engineer}"

  # Hunter provider
  printf "Hunter (execution) provider: "
  read -r hunter
  hunter="${hunter:-}"

  if [ -z "$hunter" ]; then
    echo "Error: Hunter provider is required. Re-run the wizard or edit .strategist/roles/default.yaml."
    exit 1
  fi

  # Knowledge base
  printf "Knowledge base path (leave blank to create at .strategist/knowledge): "
  read -r knowledge_path

  # Install runtime files first, then configure
  copy_skill_runtime
  write_active_yaml "$TEMPLATE"

  # Patch base_path in active.yaml
  sed -i "s|^base_path:.*|base_path: ${base_path}|" "${TARGET_REPO}/.strategist/active.yaml"

  # Write roles/default.yaml into .strategist/
  cat > "${TARGET_REPO}/.strategist/roles/default.yaml" <<EOF
scout: ${scout}
engineer: ${engineer}
hunter: ${hunter}
EOF

  # Scaffold workspace
  scaffold_workspace "$base_path"

  # Handle knowledge base
  if [ -n "$knowledge_path" ]; then
    sed -i "s|knowledge_index_path:.*|knowledge_index_path: ${TARGET_REPO}/.strategist/knowledge.index.yaml|" \
      "${TARGET_REPO}/.strategist/active.yaml"
    echo "[Strategist] knowledge base path recorded: ${knowledge_path}"
  else
    local kb_path="${TARGET_REPO}/.strategist/knowledge"
    mkdir -p "$kb_path"
    echo "[Strategist] empty knowledge base initialized at: ${kb_path}"
  fi

  install_agent_shims

  echo ""
  echo "Setup complete. active.yaml written to: ${TARGET_REPO}/.strategist/active.yaml"
  echo "Edit .strategist/roles/default.yaml to change slot providers."
}

# ── silent install ────────────────────────────────────────────────────────────

run_silent() {
  copy_skill_runtime
  write_active_yaml "pragmatic-standalone.yaml"
  local base_path
  base_path="$(grep '^base_path:' "${TARGET_REPO}/.strategist/active.yaml" | awk '{print $2}')"
  scaffold_workspace "$base_path"
  install_agent_shims
  echo "[Strategist] install complete. Run with --wizard for interactive setup."
}

# ── main ──────────────────────────────────────────────────────────────────────

if [ "$WIZARD" = true ]; then
  run_wizard
else
  run_silent
fi
