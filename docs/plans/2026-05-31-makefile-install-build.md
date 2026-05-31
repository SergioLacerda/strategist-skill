# Makefile — make install / make build Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace `install-local` with `install` (build → copy binary → run `strategist install`) and confirm `build` is correct as-is.

**Architecture:** Single-file change in `Makefile`. No new files, no tests (Makefile targets are integration-level; verified by running them). The `install` target depends on `build`, installs the binary to `~/.local/bin`, then runs `~/.local/bin/strategist install` to set up the skill in the current directory.

**Tech Stack:** GNU Make, Go toolchain, `install` POSIX utility.

---

### Task 1: Update .PHONY and replace install-local with install

**Files:**
- Modify: `Makefile`

**Step 1: Open the Makefile and locate the .PHONY line**

Current `.PHONY` line:
```
.PHONY: build test lint vuln bench cover cover-gate cover-html analysis-structure-gate install-local release snapshot clean
```

Replace `install-local` with `install`:
```
.PHONY: build test lint vuln bench cover cover-gate cover-html analysis-structure-gate install release snapshot clean
```

**Step 2: Replace the install-local target**

Current target:
```makefile
install-local: build
	install -m 755 bin/strategist ~/.local/bin/strategist
```

Replace with:
```makefile
install: build
	install -m 755 bin/strategist ~/.local/bin/strategist
	~/.local/bin/strategist install
```

> Note: `~/.local/bin/strategist install` uses the freshly installed binary explicitly, so the skill setup always reflects what was just compiled.

**Step 3: Verify the diff looks correct**

Run:
```bash
git diff Makefile
```

Expected: `.PHONY` has `install` not `install-local`; new `install:` target with 2 recipe lines; no other changes.

**Step 4: Smoke-test make build**

Run:
```bash
make build
```

Expected: compiles without errors, `bin/strategist` is updated.

**Step 5: Smoke-test make install**

Run:
```bash
make install
```

Expected:
- Binary copied to `~/.local/bin/strategist`
- `strategist install` runs and exits 0 (skill installed/updated in current directory)

**Step 6: Commit**

```bash
git add Makefile
git commit -m "feat: add make install target (build + install binary + skill setup)"
```
