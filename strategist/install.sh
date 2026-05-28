#!/usr/bin/env bash
# Strategist install script.
# Silent by default: generates active.yaml from pragmatic-standalone template.
# --wizard flag enables TUI for interactive setup.
# Usage:
#   sh install.sh                    # silent install with pragmatic defaults
#   sh install.sh --wizard           # interactive TUI setup
#   sh install.sh --target /path     # set target repo root (default: current dir)

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
  read_template "$template_name" > "${SKILL_ROOT}/active.yaml"
  echo "[Strategist] active.yaml created from template: $template_name"
}

scaffold_workspace() {
  local base_path="$1"
  local target="${TARGET_REPO}/${base_path}"
  mkdir -p "${target}/todo" "${target}/pending" "${target}/refined" "${target}/done" "${target}/.strategist"

  # Copy internal domain templates into .strategist/ if not already present.
  local domain_src="${SKILL_ROOT}/templates/domain"
  if [ -d "$domain_src" ] && [ ! -f "${target}/.strategist/index.yaml" ]; then
    cp -r "${domain_src}/." "${target}/.strategist/"
    echo "[Strategist] workspace scaffolded at: ${target}"
  else
    echo "[Strategist] workspace directories ensured at: ${target}"
  fi
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
    echo "Error: Hunter provider is required. Re-run the wizard or edit roles/default.yaml."
    exit 1
  fi

  # Knowledge base
  printf "Knowledge base path (leave blank to create at ${base_path}/.strategist/knowledge): "
  read -r knowledge_path

  # Write active.yaml from chosen template then patch values
  write_active_yaml "$TEMPLATE"
  # Patch base_path
  sed -i "s|^base_path:.*|base_path: ${base_path}|" "${SKILL_ROOT}/active.yaml"

  # Write roles/default.yaml
  cat > "${SKILL_ROOT}/roles/default.yaml" <<EOF
scout: ${scout}
engineer: ${engineer}
hunter: ${hunter}
EOF

  # Scaffold workspace
  scaffold_workspace "$base_path"

  # Handle knowledge base
  if [ -n "$knowledge_path" ]; then
    sed -i "s|knowledge_index_path:.*|knowledge_index_path: ${SKILL_ROOT}/knowledge.index.yaml|" "${SKILL_ROOT}/active.yaml"
    echo "[Strategist] knowledge base path recorded: ${knowledge_path}"
  else
    local kb_path="${TARGET_REPO}/${base_path}/.strategist/knowledge"
    mkdir -p "$kb_path"
    echo "[Strategist] empty knowledge base initialized at: ${kb_path}"
  fi

  echo ""
  echo "Setup complete. active.yaml written to: ${SKILL_ROOT}/active.yaml"
  echo "Edit roles/default.yaml to change slot providers."
}

# ── silent install ────────────────────────────────────────────────────────────

run_silent() {
  write_active_yaml "pragmatic-standalone.yaml"
  local base_path
  base_path="$(grep '^base_path:' "${SKILL_ROOT}/active.yaml" | awk '{print $2}')"
  scaffold_workspace "$base_path"
  echo "[Strategist] install complete. Run with --wizard for interactive setup."
}

# ── main ──────────────────────────────────────────────────────────────────────

if [ "$WIZARD" = true ]; then
  run_wizard
else
  run_silent
fi
