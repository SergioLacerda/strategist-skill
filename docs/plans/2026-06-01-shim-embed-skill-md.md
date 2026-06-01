# Shim Embed SKILL.md Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the 7-line stub shim with the full SKILL.md content so the Strategist pipeline actually executes when invoked.

**Architecture:** After `installGlobalRuntime()` populates `~/.strategist/`, `Install()` reads `~/.strategist/SKILL.md` and passes it to `installShimFor()`, which prepends YAML frontmatter and writes the full content to `~/.claude/skills/strategist/SKILL.md`. No embedded content accessed twice — the just-extracted file is the source.

**Tech Stack:** Go 1.22+, `os.ReadFile`, `testify/assert`, `testify/require`. All changes in `internal/install/`.

---

### Task 1: Update `shim.go` — signatures and content generation

**Files:**
- Modify: `internal/install/shim.go`

**Step 1: Read the current file**

Open `internal/install/shim.go` and understand the existing `shimContent` constant and `installShimTo(homeDir string)` function.

**Step 2: Replace the file content**

Replace the entire file with:

```go
package install

import (
	"fmt"
	"os"
	"path/filepath"
)

const shimFrontmatter = `---
name: strategist
description: "Multi-phase mission orchestrator. Coordinates discovery, refinement, and execution through three pluggable slots."
---

`

func generateShimContent(skillContent string) string {
	return shimFrontmatter + skillContent
}

// installShim creates ~/.claude/skills/strategist/SKILL.md containing the full
// pipeline instructions so Claude receives them inline at skill invocation time.
func installShim(target string) error {
	_ = target // reserved for future multi-root shim support
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	return installShimTo(home, "")
}

// installShimTo writes the shim under homeDir with the given skill content.
// skillContent is the raw content of ~/.strategist/SKILL.md. An empty string
// produces a shim with frontmatter only (used in error-path tests).
func installShimTo(homeDir, skillContent string) error {
	shimDir := filepath.Join(homeDir, ".claude", "skills", "strategist")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		return fmt.Errorf("mkdir shim dir: %w", err)
	}
	shimPath := filepath.Join(shimDir, "SKILL.md")
	content := []byte(generateShimContent(skillContent))
	if err := os.WriteFile(shimPath, content, 0o644); err != nil {
		return fmt.Errorf("write shim: %w", err)
	}
	return nil
}
```

**Step 3: Verify it compiles**

```bash
go build ./internal/install/...
```
Expected: no errors.

**Step 4: Commit**

```bash
git add internal/install/shim.go
git commit -m "refactor(install): replace stub shim with full SKILL.md content generation"
```

---

### Task 2: Update `installer.go` — read SKILL.md and pass to shim

**Files:**
- Modify: `internal/install/installer.go`

**Step 1: Add `readGlobalSKILLMD` helper**

Insert this method on `Service` after `installGlobalRuntime`:

```go
// readGlobalSKILLMD reads the SKILL.md that installGlobalRuntime just extracted.
// Returns a fatal error if the file is absent — installGlobalRuntime must have created it.
func (s Service) readGlobalSKILLMD(ctx context.Context) (string, error) {
	homeDir := s.ShimHomeDir
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home dir: %w", err)
		}
	}
	path := filepath.Join(homeDir, ".strategist", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read global SKILL.md: %w", err)
	}
	slog.InfoContext(ctx, "[Strategist] SKILL.md read for shim", "path", path)
	return string(data), nil
}
```

**Step 2: Update `installShimFor` signature**

Change:
```go
func (s Service) installShimFor(target string) error {
	if s.ShimHomeDir != "" {
		return installShimTo(s.ShimHomeDir)
	}
	return installShim(target)
}
```

To:
```go
func (s Service) installShimFor(target, skillContent string) error {
	if s.ShimHomeDir != "" {
		return installShimTo(s.ShimHomeDir, skillContent)
	}
	return installShim(target)
}
```

Note: `installShim(target)` still compiles because its signature hasn't changed — but it now passes `""` as skillContent internally. That's a stub path (used only when `ShimHomeDir` is empty and no content is passed). Fix it later in step 3.

**Step 3: Update the call site in `Install()`**

Find the block in `Install()` that reads:
```go
shimPath, err := s.resolveShimPath()
if err != nil {
    return fmt.Errorf("install: resolve shim path: %w", err)
}
if err := s.installShimFor(cfg.Target); err != nil {
    return fmt.Errorf("install: shim: %w", err)
}
```

Replace with:
```go
skillContent, err := s.readGlobalSKILLMD(ctx)
if err != nil {
    return fmt.Errorf("install: read SKILL.md: %w", err)
}

shimPath, err := s.resolveShimPath()
if err != nil {
    return fmt.Errorf("install: resolve shim path: %w", err)
}
if err := s.installShimFor(cfg.Target, skillContent); err != nil {
    return fmt.Errorf("install: shim: %w", err)
}
```

**Step 4: Verify it compiles**

```bash
go build ./internal/install/...
```
Expected: no errors.

**Step 5: Commit**

```bash
git add internal/install/installer.go
git commit -m "feat(install): read ~/.strategist/SKILL.md and embed in agent shim"
```

---

### Task 3: Fix tests in `install_test.go`

**Files:**
- Modify: `internal/install/install_test.go`

**Step 1: Run tests to see current failures**

```bash
go test ./internal/install/... 2>&1
```
Expected: compile errors — `installShimTo` calls missing second argument.

**Step 2: Update `TestInstallShimTo_ReadOnlyParent` and `TestInstallShimTo_WriteError`**

Both tests call `installShimTo(home)` — add `""` as second argument:

```go
err := installShimTo(home, "")
```

Do this for both tests.

**Step 3: Update `TestInstall_Silent` to assert shim content**

After the existing `assert.FileExists` checks, add:

```go
shimPath := filepath.Join(svc.ShimHomeDir, ".claude", "skills", "strategist", "SKILL.md")
shimData, err := os.ReadFile(shimPath)
require.NoError(t, err)
shimStr := string(shimData)
assert.Contains(t, shimStr, "name: strategist", "shim must have frontmatter")
assert.Contains(t, shimStr, "# SKILL", "shim must contain SKILL.md content from extractor")
```

(The `mockExtractor` writes `"# SKILL\n"` as the SKILL.md content — that's what the assertion checks.)

**Step 4: Update `TestInstall_GlobalRuntimePopulated` similarly**

Add the same shim content assertions at the end of that test.

**Step 5: Run tests**

```bash
go test ./internal/install/... -v -run "TestInstall_Silent|TestInstall_GlobalRuntime" 2>&1
```
Expected: PASS.

**Step 6: Commit**

```bash
git add internal/install/install_test.go
git commit -m "test(install): assert shim contains frontmatter and SKILL.md content"
```

---

### Task 4: Fix tests in `installer_whitebox_test.go`

**Files:**
- Modify: `internal/install/installer_whitebox_test.go`

**Step 1: Run tests to find failures**

```bash
go test ./internal/install/... 2>&1
```
Expected: compile errors for `installShimTo` calls missing second argument.

**Step 2: Update `TestInstallShimTo_ReadOnlyParent` and `TestInstallShimTo_WriteError`**

Both tests in this file call `installShimTo(home)` — update to pass `""`:

```go
err := installShimTo(home, "")
```

**Step 3: Add `TestReadGlobalSKILLMD_FileAbsent`**

Insert this new test in the whitebox file (it needs package-internal access):

```go
func TestReadGlobalSKILLMD_FileAbsent(t *testing.T) {
	t.Parallel()
	svc := Service{
		Extractor:   minimalExtractor{},
		Compiler:    nopCompiler{},
		ShimHomeDir: t.TempDir(), // .strategist/SKILL.md does not exist here
	}
	_, err := svc.readGlobalSKILLMD(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "read global SKILL.md")
}
```

**Step 4: Run all install tests**

```bash
go test ./internal/install/... -v 2>&1
```
Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/install/installer_whitebox_test.go
git commit -m "test(install): cover readGlobalSKILLMD absent-file error path"
```

---

### Task 5: Run full test suite and verify end-to-end

**Step 1: Run all tests with race detector**

```bash
go test -race ./... 2>&1
```
Expected: all PASS, no race conditions.

**Step 2: Build and smoke-test the binary**

```bash
make build
./bin/strategist version
```
Expected: version printed without error.

**Step 3: Verify shim content after install (manual)**

```bash
make install
strategist install --wizard
cat ~/.claude/skills/strategist/SKILL.md | head -10
```
Expected output starts with:
```
---
name: strategist
description: "Multi-phase mission orchestrator..."
---

## ⚠️ MANDATORY — BEFORE ANY RESPONSE
```

**Step 4: Final commit if any loose ends**

```bash
git status
```
If clean: done. If not: stage and commit remaining changes.
