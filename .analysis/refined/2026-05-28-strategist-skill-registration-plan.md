# Implementation Plan — Strategist Skill Registration
**Date:** 2026-05-28  
**Spec:** `.analysis/pending/2026-05-28-strategist-skill-registration-design.md`  
**Scope:** Modify `install.sh` + `SKILL.md` to produce self-contained `.strategist/` install and register agent shims

---

## Task List

### T1 — Update `SKILL.md` source to declare `skill_root` resolution

**File:** `strategist/SKILL.md`  
**Change:** Add one paragraph at the top of the Bootstrap section (before the numbered list):

```
> **Skill root resolution:** If invoked from an agent shim, `skill_root` is declared in
> the frontmatter of this file. Resolve all relative paths — `active.yaml`, `personas/`,
> `roles/`, `schemas/` — from `skill_root`. If `skill_root` is not present, treat the
> directory containing this file as the skill root.
```

**Acceptance:** The SKILL.md bootstrap section reads `skill_root` before any path resolution step.

---

### T2 — Add `copy_skill_runtime()` to `install.sh`

**File:** `strategist/install.sh`  
**Position:** After the `scaffold_workspace()` function definition, before the wizard/silent functions.

**Logic:**
```bash
copy_skill_runtime() {
  local dest="$1/.strategist"
  mkdir -p "${dest}/memory"

  # Copy top-level files
  for f in SKILL.md knowledge.index.yaml protocol.md; do
    [ -f "${SKILL_ROOT}/${f}" ] && cp "${SKILL_ROOT}/${f}" "${dest}/${f}"
  done

  # Copy directories (overwrite, preserve memory contents)
  for d in personas roles schemas; do
    [ -d "${SKILL_ROOT}/${d}" ] && cp -r "${SKILL_ROOT}/${d}" "${dest}/${d}"
  done

  # Copy internal domain templates into .strategist/
  local domain_src="${SKILL_ROOT}/templates/domain"
  if [ -d "$domain_src" ] && [ ! -f "${dest}/index.yaml" ]; then
    cp -r "${domain_src}/." "${dest}/"
  fi

  echo "[Strategist] runtime installed at: ${dest}"
}
```

**Note:** `memory/` is created but its contents are never overwritten on reinstall. `index.yaml` from `templates/domain` is only copied if not already present (idempotent domain init).

**Acceptance:** After running install.sh, `<target>/.strategist/` contains SKILL.md, active.yaml, personas/, roles/, schemas/, knowledge.index.yaml, memory/, and index.yaml.

---

### T3 — Add `install_agent_shims()` to `install.sh`

**File:** `strategist/install.sh`  
**Position:** After `copy_skill_runtime()`.

**Logic:**
```bash
install_agent_shims() {
  local skill_root_abs="$1"  # absolute path to <target>/.strategist
  local description
  description="$(grep '^description:' "${SKILL_ROOT}/skill.yaml" | sed 's/^description: *//' | tr -d '"' | head -1)"

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
```

**Acceptance:** After install, each existing agent dir has `skills/strategist/SKILL.md` with `skill_root` pointing to the installed `.strategist/` path.

---

### T4 — Update `write_active_yaml()` to write to `.strategist/`

**File:** `strategist/install.sh`  
**Change:** The function currently writes to `${SKILL_ROOT}/active.yaml`. Change the target to `${TARGET_REPO}/.strategist/active.yaml`.

```bash
write_active_yaml() {
  local template_name="$1"
  read_template "$template_name" > "${TARGET_REPO}/.strategist/active.yaml"
  echo "[Strategist] active.yaml created from template: $template_name"
}
```

**Note:** `copy_skill_runtime()` must run before `write_active_yaml()` so the `.strategist/` dir exists. Order: `copy_skill_runtime` → `write_active_yaml` → `scaffold_workspace` → `install_agent_shims`.

**Acceptance:** `<target>/.strategist/active.yaml` exists after install. No `active.yaml` is written to skill root.

---

### T5 — Update `scaffold_workspace()` call site

**File:** `strategist/install.sh`  
**Change:** `scaffold_workspace()` currently reads `base_path` from `${SKILL_ROOT}/active.yaml`. After T4, it must read from `${TARGET_REPO}/.strategist/active.yaml`.

```bash
run_silent() {
  copy_skill_runtime "${TARGET_REPO}"
  write_active_yaml "pragmatic-standalone.yaml"
  local base_path
  base_path="$(grep '^base_path:' "${TARGET_REPO}/.strategist/active.yaml" | awk '{print $2}')"
  scaffold_workspace "$base_path"
  install_agent_shims "$(cd "${TARGET_REPO}/.strategist" && pwd)"
  echo "[Strategist] install complete. Run with --wizard for interactive setup."
}
```

Update `run_wizard()` similarly — patch `${TARGET_REPO}/.strategist/active.yaml` instead of `${SKILL_ROOT}/active.yaml`, and write `roles/default.yaml` to `${TARGET_REPO}/.strategist/roles/default.yaml`.

**Acceptance:** Silent and wizard installs both produce the same artifact layout.

---

### T6 — Update `knowledge_index_path` reference in wizard

**File:** `strategist/install.sh`  
**Context:** In `run_wizard()`, the sed patch for `knowledge_index_path` currently references `${SKILL_ROOT}/knowledge.index.yaml`. After T4, the active.yaml is in `.strategist/` and `knowledge.index.yaml` was copied there by `copy_skill_runtime()`.

**Change:** Patch to point to `.strategist/knowledge.index.yaml` using a relative path or the absolute path derived from `TARGET_REPO`.

```bash
sed -i "s|knowledge_index_path:.*|knowledge_index_path: ${TARGET_REPO}/.strategist/knowledge.index.yaml|" \
  "${TARGET_REPO}/.strategist/active.yaml"
```

**Acceptance:** `active.yaml`'s `knowledge_index_path` resolves to the installed copy, not the skill source.

---

### T7 — Manual smoke test

After all code changes, run:

```bash
cd /tmp && mkdir test-strategist-install && cd test-strategist-install
sh /home/sergio/dev/strategist-skill/strategist/install.sh
```

Verify:
- [ ] `.strategist/` created with all expected files
- [ ] `~/.claude/skills/strategist/SKILL.md` exists with correct `skill_root`
- [ ] `~/.gemini/skills/strategist/SKILL.md` exists (if dir present)
- [ ] `~/.gemini/antigravity/skills/strategist/SKILL.md` exists
- [ ] `~/.codex/skills/strategist/SKILL.md` exists (if dir present)
- [ ] No `active.yaml` written to `strategist/` skill source
- [ ] `/strategist` appears in next Claude Code terminal session

---

## Order of Execution

```
T1  → SKILL.md source update (prerequisite for shim correctness)
T2  → copy_skill_runtime() function
T3  → install_agent_shims() function
T4  → write_active_yaml() path update
T5  → run_silent() / run_wizard() orchestration update
T6  → knowledge_index_path wizard patch
T7  → smoke test
```

T1–T6 are all in two files (`strategist/SKILL.md`, `strategist/install.sh`). No other files change.
