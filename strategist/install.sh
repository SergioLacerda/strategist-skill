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

# ── provider validation ───────────────────────────────────────────────────────

# validate_provider <name> <required_risk> <slot_label>
# Returns 0 if valid, 1 if mismatch or not found.
validate_provider() {
  local name="$1" required="$2" label="$3"
  local score=""

  # 1. Check ~/.claude/skills/<name>/skill.yaml
  local skill_yaml="${HOME}/.claude/skills/${name}/skill.yaml"
  if [ -f "$skill_yaml" ]; then
    score=$(grep -m1 '^risk_score:' "$skill_yaml" | awk '{print $2}')
  fi

  # 2. Fall back to templates/known-providers.yaml
  if [ -z "$score" ]; then
    local known="${SKILL_ROOT}/templates/known-providers.yaml"
    if [ -f "$known" ]; then
      score=$(grep -m1 "^  ${name}:" "$known" | awk '{print $2}')
    fi
  fi

  # 3. Not found — prompt user
  if [ -z "$score" ]; then
    echo "  ⚠ risk_score for '${name}' not declared."
    printf "  Enter value [write_pending/write_analysis/controlled/orchestrator]: "
    read -r score
  fi

  # 4. Compare
  if [ "$score" != "$required" ]; then
    echo "  ✗ Slot ${label} requires '${required}', but '${name}' declares '${score}'."
    return 1
  fi

  echo "  ✓ ${name} → ${score}"
  return 0
}

# ── wizard TUI ───────────────────────────────────────────────────────────────

run_wizard() {
  # Redirect stdin from terminal so read works even when script is piped (curl | bash)
  exec < /dev/tty

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

  # Ranger (discovery) provider
  echo ""
  echo "  Ranger: descobre o espaço do problema → escreve discovery em pending/"
  echo "  Provider recomendado: brainstorming (explora antes de decidir)"
  while true; do
    printf "Ranger provider (discovery, write_pending) [brainstorming]: "
    read -r ranger
    ranger="${ranger:-brainstorming}"
    validate_provider "$ranger" "write_pending" "Ranger" && break
    echo "  Enter a different provider name."
  done

  # Archivist (refinement) provider
  echo ""
  echo "  Archivist: refina a descoberta → escreve proposal/design/tasks em refined/"
  echo "  Provider recomendado: openspec-explore (gera estrutura OpenSpec)"
  while true; do
    printf "Archivist provider (refinement, write_analysis) [openspec-explore]: "
    read -r archivist
    archivist="${archivist:-openspec-explore}"
    validate_provider "$archivist" "write_analysis" "Archivist" && break
    echo "  Enter a different provider name."
  done

  # Sniper (execution) provider
  echo ""
  echo "  Sniper: executa o plano refinado → requer approval gate"
  echo "  Provider recomendado: sdd-ask (execução governada)"
  while true; do
    printf "Sniper provider (execution, controlled) [sdd-ask]: "
    read -r sniper
    sniper="${sniper:-sdd-ask}"
    [ -n "$sniper" ] || { echo "  Error: Sniper provider is required."; continue; }
    validate_provider "$sniper" "controlled" "Sniper" && break
    echo "  Enter a different provider name."
  done

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
discovery: ${ranger}
refinement: ${archivist}
execution: ${sniper}
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
