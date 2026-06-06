# Docs Governance — Editorial Policies Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Bring `docs/` into compliance with the four editorial policies defined in `.analysis/pending/2026-06-06-docs-governance-editorial-design.md` — ADR authority, docs quality, navigation, and language.

**Architecture:** Five independent tasks executed sequentially. Tasks 1–2 repair existing docs (maturity gate fields + dead cross-reference). Tasks 3–4 migrate two informal decision records from `.analysis/done/` into canonical ADRs in `docs/adr/`. Task 5 creates `docs/README.md` as the single navigation entry point and language policy declaration.

**Tech Stack:** Markdown. No build tooling — verification is done with `grep`, `ls`, and manual review.

---

## Task 1: Add Status and Date to existing `docs/` files

Policy 2 requires every file in `docs/` to declare `Status` and `Date` (or `Last Updated`). Several files are missing both. Also, `skill-internals.md` references `../readme_detailed.md`, which is outside `docs/` — a self-containment violation that must be removed.

**Files to modify:**

- Modify: `docs/architecture.md`
- Modify: `docs/cli-reference.md`
- Modify: `docs/configuration.md`
- Modify: `docs/skill-internals.md`
- Modify: `docs/c4-diagrams.md`
- Modify: `docs/performance-baseline.md` (has `Date` but not `Status`)

### Step 1: Add Status + Date to `docs/architecture.md`

Insert after the `# Arquitetura — Strategist Skill` heading (before the first `## Visão Geral`):

```markdown
**Status:** Accepted
**Last Updated:** 2026-06-06
```

### Step 2: Run verification

```bash
head -5 docs/architecture.md
```

Expected: heading, then `**Status:** Accepted`, then `**Last Updated:** 2026-06-06`.

### Step 3: Add Status + Date to remaining docs files

Apply the same insertion pattern to each file:

- `docs/cli-reference.md` — insert after `# Referência CLI — strategist`
- `docs/configuration.md` — insert after `# Referência de Configuração — Strategist Skill`
- `docs/skill-internals.md` — insert after `# Internals da Skill — Sub-skills, Contratos e Schemas`
- `docs/c4-diagrams.md` — insert after `# Diagramas C4 — Strategist Skill`

For `docs/performance-baseline.md` — `Date:` already exists. Insert `**Status:** Accepted` immediately after the existing `**Date:**` line.

### Step 4: Remove external cross-reference from `skill-internals.md`

Locate and remove (or replace inline) the following line in `docs/skill-internals.md`:

```markdown
Para o pipeline geral e comportamento dos slots, veja [readme_detailed.md](../readme_detailed.md).
```

Replace with a standalone sentence that does not reference an external file. Example:

```markdown
Para o pipeline geral e comportamento dos slots, consulte `docs/architecture.md`.
```

### Step 5: Verify no external references remain in `docs/`

```bash
grep -rn "\.\.\/" docs/*.md
```

Expected: no output (no `../` paths pointing outside `docs/`).

### Step 6: Verify all `docs/` files have Status and Date

```bash
for f in docs/*.md; do
  echo "=== $f ==="
  grep -E "Status:|Last Updated:|Date:" "$f" | head -3
done
```

Expected: each file shows at least one `Status:` line and one `Date:` or `Last Updated:` line.

### Step 7: Commit

```bash
git add docs/architecture.md docs/cli-reference.md docs/configuration.md docs/skill-internals.md docs/c4-diagrams.md docs/performance-baseline.md
git commit -m "docs: add Status and Date fields to all docs/ files (Policy 2 compliance)"
```

---

## Task 2: Migrate E2E ADR from `.analysis/done/` to `docs/adr/0006-...`

Source: `.analysis/done/20260602-testes-e2e-skill-adr.md`
Target: `docs/adr/0006-e2e-test-entry-point-full-install-pipeline.md`

The source file is a formal architectural decision record (it records the choice of E2E test entry point via full install pipeline). It belongs in `docs/adr/`, not in `.analysis/done/`. The source file remains in `.analysis/done/` — it is not deleted. The new ADR is an independent standalone document.

### Step 1: Read the source file

```bash
cat .analysis/done/20260602-testes-e2e-skill-adr.md
```

### Step 2: Create the canonical ADR

Create `docs/adr/0006-e2e-test-entry-point-full-install-pipeline.md`.

The new file must:
- Use the canonical header format matching ADRs 0001–0005:
  ```markdown
  # ADR-0006 — E2E Test Entry Point via Full Install Pipeline

  **Status:** Accepted
  **Date:** 2026-06-02
  ```
- Include context, decision, and consequences sections with content transferred and self-contained (no links to files outside `docs/`)
- Remove or rewrite any references to `.analysis/` paths or external mission artifacts
- Be fully readable without consulting any other file — inline context that the source file currently cross-references

### Step 3: Verify canonical format

```bash
head -6 docs/adr/0006-e2e-test-entry-point-full-install-pipeline.md
```

Expected: `# ADR-0006 —` on line 1, `**Status:** Accepted` on line 3, `**Date:** 2026-06-02` on line 4.

### Step 4: Verify no external cross-references

```bash
grep -n "\.\.\/" docs/adr/0006-e2e-test-entry-point-full-install-pipeline.md
grep -n "\.analysis\/" docs/adr/0006-e2e-test-entry-point-full-install-pipeline.md
```

Expected: no output.

### Step 5: Commit

```bash
git add docs/adr/0006-e2e-test-entry-point-full-install-pipeline.md
git commit -m "docs: add ADR-0006 — E2E test entry point via full install pipeline"
```

---

## Task 3: Migrate Structural Compression ADR from `.analysis/done/` to `docs/adr/0007-...`

Source: `.analysis/done/token-economy-phase4-decision-adr.md`
Target: `docs/adr/0007-structural-compression-agent-contract.md`

Same rationale as Task 2: formal architectural decision (deferral of CompressionProvider to agent contract) belongs in `docs/adr/`. Source file stays in `.analysis/done/`.

### Step 1: Read the source file

```bash
cat .analysis/done/token-economy-phase4-decision-adr.md
```

### Step 2: Create the canonical ADR

Create `docs/adr/0007-structural-compression-agent-contract.md`.

Requirements identical to Task 2:
- Canonical header:
  ```markdown
  # ADR-0007 — Structural Compression Pipeline — Agent Contract vs Go Runtime

  **Status:** Accepted
  **Date:** 2026-06-02
  ```
- Context, decision, and consequences sections — self-contained
- No links to `.analysis/` paths, `refined/` files, or mission artifacts

> Note: The source references `.analysis/refined/token_economy/2026-06-01-token-economy-design.md`. Replace this with an inline summary of the relevant context so the ADR is self-contained.

### Step 3: Verify canonical format

```bash
head -6 docs/adr/0007-structural-compression-agent-contract.md
```

Expected: `# ADR-0007 —` on line 1, `**Status:** Accepted` on line 3, `**Date:** 2026-06-02` on line 4.

### Step 4: Verify no external cross-references

```bash
grep -n "\.\.\/" docs/adr/0007-structural-compression-agent-contract.md
grep -n "\.analysis\/" docs/adr/0007-structural-compression-agent-contract.md
```

Expected: no output.

### Step 5: Commit

```bash
git add docs/adr/0007-structural-compression-agent-contract.md
git commit -m "docs: add ADR-0007 — structural compression pipeline agent contract"
```

---

## Task 4: Audit all `docs/adr/` ADRs for self-containment (0001–0007)

Verify that no ADR in `docs/adr/` references files outside `docs/` or uses external paths.

### Step 1: Check for external references in all ADRs

```bash
grep -rn "\.\.\/" docs/adr/*.md
grep -rn "\.analysis\/" docs/adr/*.md
grep -rn "\.strategist\/" docs/adr/*.md
```

Expected: no output from any command.

### Step 2: Verify required fields in all ADRs

```bash
for f in docs/adr/*.md; do
  echo "=== $f ==="
  grep -E "^\*\*Status:\*\*|^\*\*Date:\*\*|^\*\*Data:\*\*" "$f"
done
```

Expected: each file shows exactly one `**Status:**` line and one `**Date:**` (or `**Data:**`) line.

### Step 3: If any ADR fails — fix inline

For each ADR that fails either check:
- External reference: replace with inline context or a reference within `docs/`
- Missing field: add `**Status:** Accepted` and `**Date:** YYYY-MM-DD` after the first heading

### Step 4: Commit any fixes

```bash
git add docs/adr/
git commit -m "docs: fix ADR self-containment and required fields (audit pass)"
```

Skip if no changes were needed.

---

## Task 5: Create `docs/README.md`

Policy 3 requires `docs/README.md` as the single navigation entry point. It does not exist yet. Policy 4 requires the language policy to be declared once in `docs/README.md`.

### Step 1: Confirm `docs/README.md` does not exist

```bash
ls docs/README.md 2>&1
```

Expected: `No such file or directory`.

### Step 2: Create `docs/README.md`

The file language follows the skill's configured `language.docs`. The existing `docs/` content is in Portuguese (`pt-BR`), so `docs/README.md` is written in Portuguese.

```markdown
# Documentação — Strategist Skill

**Status:** Accepted
**Last Updated:** 2026-06-06

Este é o ponto de entrada para toda a documentação da skill. Use a tabela abaixo para navegar pela intenção.

---

## Navegação por Intenção

| Quero... | Começar por |
|----------|-------------|
| Entender a arquitetura geral | [`docs/architecture.md`](architecture.md) |
| Revisar decisões de design | [`docs/adr/`](adr/) |
| Configurar a skill | [`docs/configuration.md`](configuration.md) |
| Entender os internals | [`docs/skill-internals.md`](skill-internals.md) |
| Usar o CLI | [`docs/cli-reference.md`](cli-reference.md) |
| Ver diagramas C4 | [`docs/c4-diagrams.md`](c4-diagrams.md) |
| Ver baseline de performance | [`docs/performance-baseline.md`](performance-baseline.md) |
| Navegar análises concluídas | `<base_path>/done/` |

---

## Política de Idioma

A documentação segue a configuração de idioma da skill (`language.docs`). Todos os arquivos em `docs/` são escritos nesse idioma. Artefatos de interação seguem `language.chat`. Internals de código seguem `language.code`.

---

## Regra de Manutenção

Todo pull request que adicionar um novo arquivo em `docs/` deve atualizar a tabela de navegação acima.
```

### Step 3: Verify the file

```bash
grep -c "Quero\|Start\|Navegação\|Navigation" docs/README.md
```

Expected: output ≥ 1.

```bash
grep "Política de Idioma\|Language Policy" docs/README.md
```

Expected: one match.

### Step 4: Verify `docs/README.md` only references files within `docs/`

```bash
grep -n "\.\.\/" docs/README.md
grep -n "http" docs/README.md
```

Expected: no output from either command (no external paths, no absolute URLs).

### Step 5: Commit

```bash
git add docs/README.md
git commit -m "docs: create docs/README.md — navigation entry point and language policy (Policy 3 + 4)"
```

---

## Verification Sweep (post-all-tasks)

After all five tasks are committed, run a final sweep to confirm all policies are satisfied.

```bash
# Policy 1: All ADRs have Status and Date, 0001–0007 exist
ls docs/adr/
for f in docs/adr/*.md; do grep -l "Status:" "$f"; done

# Policy 2: All docs/ files have Status and Date
for f in docs/*.md; do
  s=$(grep -c "Status:" "$f" 2>/dev/null || echo 0)
  d=$(grep -cE "Date:|Last Updated:" "$f" 2>/dev/null || echo 0)
  echo "$f — status=$s date=$d"
done

# Policy 2: No external cross-references in docs/
grep -rn "\.\.\/" docs/*.md docs/adr/*.md

# Policy 3: docs/README.md exists and has navigation table
grep "Navegação\|Navigation" docs/README.md

# Policy 4: Language policy declared in docs/README.md
grep "Política de Idioma\|Language Policy" docs/README.md
```

Expected for all checks:
- `ls docs/adr/` shows 0001–0007
- All docs/ files show `status=1 date=1`
- No `../` paths
- README has navigation table
- README has language policy section
