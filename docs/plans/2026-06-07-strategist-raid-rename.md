# Strategist-Raid Rename Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rename the scaffolded skill `raid` to `strategist-raid` across active surfaces and historical analysis artifacts, with no compatibility layer.

**Architecture:** This change is a strict identity and path rename. Update the canonical contract path first, then rename all skill directories and internal ids, then rewrite active docs/tests, then normalize historical `.analysis/archived/` artifacts, and finally validate with targeted text search plus spec/governance checks.

**Tech Stack:** Markdown, YAML, Go spec tests, SDD governance CLI

---

### Task 1: Rename the canonical contract file

**Files:**
- Modify: `strategist/contracts/raid.yaml`
- Modify: `.strategist/skills/raid/skill.yaml`
- Modify: `internal/embed/defaults/skills/raid/skill.yaml`
- Modify: `strategist/skills/raid/skill.yaml`

**Step 1: Write the failing check**

Run:

```bash
rg -n "contracts/raid.yaml|module: raid" strategist .strategist internal/embed/defaults -S
```

Expected: matches in the current contract and skill metadata.

**Step 2: Rename the contract file**

Move:

```bash
mv strategist/contracts/raid.yaml strategist/contracts/strategist-raid.yaml
```

Expected: old path gone, new path present.

**Step 3: Update the contract content**

Change in `strategist/contracts/strategist-raid.yaml`:
- `module: raid` → `module: strategist-raid`
- `Skill /raid` → `Skill /strategist-raid`

**Step 4: Update skill entry points**

Change in:
- `strategist/skills/raid/skill.yaml`
- `internal/embed/defaults/skills/raid/skill.yaml`
- `.strategist/skills/raid/skill.yaml`

From:

```yaml
entry_point:
  contract: strategist/contracts/raid.yaml
```

To:

```yaml
entry_point:
  contract: strategist/contracts/strategist-raid.yaml
```

**Step 5: Run the check again**

Run:

```bash
rg -n "contracts/raid.yaml|module: raid" strategist .strategist internal/embed/defaults -S
```

Expected: no matches.

**Step 6: Commit**

```bash
git add strategist/contracts/strategist-raid.yaml strategist/skills/raid/skill.yaml internal/embed/defaults/skills/raid/skill.yaml .strategist/skills/raid/skill.yaml
git commit -m "refactor: rename raid contract to strategist-raid"
```

---

### Task 2: Rename active skill directories and identities

**Files:**
- Rename: `strategist/skills/raid/` → `strategist/skills/strategist-raid/`
- Rename: `internal/embed/defaults/skills/raid/` → `internal/embed/defaults/skills/strategist-raid/`
- Rename: `.strategist/skills/raid/` → `.strategist/skills/strategist-raid/`
- Modify: `strategist/skills/strategist-raid/skill.yaml`
- Modify: `strategist/skills/strategist-raid/SKILL.md`
- Modify: `internal/embed/defaults/skills/strategist-raid/skill.yaml`
- Modify: `internal/embed/defaults/skills/strategist-raid/SKILL.md`
- Modify: `.strategist/skills/strategist-raid/skill.yaml`
- Modify: `.strategist/skills/strategist-raid/SKILL.md`

**Step 1: Verify current paths**

Run:

```bash
find strategist/skills internal/embed/defaults/skills .strategist/skills -maxdepth 2 -type d | rg "/raid$"
```

Expected: three `raid` directories.

**Step 2: Rename the directories**

Run:

```bash
mv strategist/skills/raid strategist/skills/strategist-raid
mv internal/embed/defaults/skills/raid internal/embed/defaults/skills/strategist-raid
mv .strategist/skills/raid .strategist/skills/strategist-raid
```

Expected: old directories gone, new directories present.

**Step 3: Update ids and headings**

In all three `skill.yaml` files:

```yaml
id: strategist-raid
```

In all three `SKILL.md` files:

```markdown
# strategist-raid — Agent Instructions
```

And replace:
- `You are \`raid\`` → `You are \`strategist-raid\``
- `Never invoke execution as part of /raid.` → `/strategist-raid.`

**Step 4: Verify**

Run:

```bash
rg -n "id: raid|# raid —|`raid`|/raid\\b" strategist/skills internal/embed/defaults/skills .strategist/skills -S
```

Expected: no active-skill matches.

**Step 5: Commit**

```bash
git add strategist/skills/strategist-raid internal/embed/defaults/skills/strategist-raid .strategist/skills/strategist-raid
git commit -m "refactor: rename raid skill directories and ids"
```

---

### Task 3: Update active routing and top-level docs

**Files:**
- Modify: `strategist/SKILL.md`
- Modify: `internal/embed/defaults/SKILL.md`
- Modify: `.strategist/SKILL.md`

**Step 1: Write the failing search**

Run:

```bash
rg -n "For `/raid`|contracts/raid.yaml|/raid \\(batch refinement" strategist/SKILL.md internal/embed/defaults/SKILL.md .strategist/SKILL.md -S
```

Expected: matches in all active `SKILL.md` surfaces.

**Step 2: Update the references**

Replace:

```markdown
For `/raid` (batch refinement of captured ideas), see `contracts/raid.yaml`.
```

With:

```markdown
For `/strategist-raid` (batch refinement of captured ideas), see `contracts/strategist-raid.yaml`.
```

**Step 3: Verify**

Run:

```bash
rg -n "For `/raid`|contracts/raid.yaml" strategist/SKILL.md internal/embed/defaults/SKILL.md .strategist/SKILL.md -S
```

Expected: no matches.

**Step 4: Commit**

```bash
git add strategist/SKILL.md internal/embed/defaults/SKILL.md .strategist/SKILL.md
git commit -m "docs: rename raid routing references to strategist-raid"
```

---

### Task 4: Rename and update the spec alignment test

**Files:**
- Rename: `tests/spec/raid_spec_alignment_test.go` → `tests/spec/strategist_raid_spec_alignment_test.go`
- Modify: `tests/spec/strategist_raid_spec_alignment_test.go`

**Step 1: Rename the test file**

Run:

```bash
mv tests/spec/raid_spec_alignment_test.go tests/spec/strategist_raid_spec_alignment_test.go
```

Expected: old file gone, new file present.

**Step 2: Update asserted paths and strings**

Change expected paths from:
- `strategist/contracts/raid.yaml`
- `strategist/skills/raid/...`
- `internal/embed/defaults/skills/raid/...`

To:
- `strategist/contracts/strategist-raid.yaml`
- `strategist/skills/strategist-raid/...`
- `internal/embed/defaults/skills/strategist-raid/...`

Change the doc assertion from:

```go
"For `/raid` (batch refinement of captured ideas), see `contracts/raid.yaml`."
```

To:

```go
"For `/strategist-raid` (batch refinement of captured ideas), see `contracts/strategist-raid.yaml`."
```

**Step 3: Run the focused test**

Run:

```bash
GOCACHE=/tmp/go-build go test ./tests/spec -tags spec -run StrategistSkillReferencesRaidContract -v
```

Expected: PASS.

**Step 4: Commit**

```bash
git add tests/spec/strategist_raid_spec_alignment_test.go
git commit -m "test: rename raid spec alignment coverage"
```

---

### Task 5: Normalize historical analysis artifacts

**Files:**
- Rename: `.analysis/archived/20260607-skill-raid/` → `.analysis/archived/20260607-strategist-raid/`
- Modify: `.analysis/archived/20260607-strategist-raid/proposal.md`
- Modify: `.analysis/archived/20260607-strategist-raid/design.md`
- Modify: `.analysis/archived/20260607-strategist-raid/tasks.md`

**Step 1: Rename the directory**

Run:

```bash
mv .analysis/archived/20260607-skill-raid .analysis/archived/20260607-strategist-raid
```

Expected: old historical directory gone, new one present.

**Step 2: Rewrite nominal historical references**

Update inside the three files:
- `Skill /raid` → `Skill /strategist-raid`
- `Mission ID: 20260607-skill-raid` → `Mission ID: 20260607-strategist-raid`
- `.claude/skills/raid/...` → repo-native skill paths or `strategist/skills/strategist-raid/...`
- `contracts/raid.yaml` → `contracts/strategist-raid.yaml`
- `/raid <arquivo>` → `/strategist-raid <arquivo>`

Do not rewrite incidental prose that no longer names the entity directly unless it is still misleading.

**Step 3: Verify**

Run:

```bash
rg -n "20260607-skill-raid|Skill /raid|contracts/raid.yaml|/raid\\b" .analysis/archived/20260607-strategist-raid -S
```

Expected: no matches naming the old skill.

**Step 4: Commit**

```bash
git add .analysis/archived/20260607-strategist-raid
git commit -m "docs: normalize archived raid analysis to strategist-raid"
```

---

### Task 6: Global sweep for legacy active references

**Files:**
- Modify as needed based on search hits in:
  - `strategist/`
  - `.strategist/`
  - `internal/embed/defaults/`
  - `tests/spec/`

**Step 1: Run the sweep**

Run:

```bash
rg -n "id: raid|`/raid`|contracts/raid.yaml|skills/raid/|\\bmodule: raid\\b|\\b/raid\\b" strategist .strategist internal/embed/defaults tests/spec -S
```

Expected: no matches.

**Step 2: Fix residuals**

If any matches remain:
- update the exact file
- re-run the same `rg` command

**Step 3: Commit**

```bash
git add strategist .strategist internal/embed/defaults tests/spec
git commit -m "refactor: remove remaining legacy raid references"
```

---

### Task 7: Final validation

**Files:**
- Validate workspace state only

**Step 1: Run spec tests**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/embed ./tests/spec -tags spec
```

Expected: all PASS.

**Step 2: Run governance validation**

Run:

```bash
UV_CACHE_DIR=/tmp/uv-cache uv run sdd governance validate
```

Expected: PASS, with only pre-existing non-blocking language advisories if any.

**Step 3: Run final runtime check**

Run:

```bash
UV_CACHE_DIR=/tmp/uv-cache uv run sdd runtime status
```

Expected: `SDD GOVERNANCE: drift=none | governance=healthy | profile=client`

**Step 4: Final commit**

```bash
git add strategist .strategist internal/embed/defaults tests/spec .analysis/archived
git commit -m "refactor: rename raid skill to strategist-raid"
```

