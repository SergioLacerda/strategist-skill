package testutil

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// userShimRelPaths are the files an install writes under the invoking user's
// home directory. A test that forgets to isolate HOME (or to set the no-shim
// flag) silently overwrites the developer's real installation, which is how
// TestInstallCmd_PrintsCompletion destroyed a live ~/.claude/skills/strategist.
var userShimRelPaths = []string{
	filepath.Join(".claude", "skills", "strategist", "SKILL.md"),
	filepath.Join(".gemini", "skills", "strategist", "SKILL.md"),
	filepath.Join(".gemini", "antigravity", "skills", "strategist", "SKILL.md"),
	filepath.Join(".codex", "skills", "strategist", "SKILL.md"),
}

// RunWithHomeGuard runs a package's tests and fails the run if any of them
// wrote to the invoking user's real home directory. Packages that install the
// runtime call it from TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(testutil.RunWithHomeGuard(m)) }
//
// It snapshots the user home before the run rather than asserting on $HOME,
// because tests legitimately set, clear and restore HOME while they execute.
func RunWithHomeGuard(m *testing.M) int {
	if m == nil {
		return 0
	}
	return runWithHomeGuardRunner(m.Run)
}

func runWithHomeGuardRunner(runFn func() int) int {
	home, err := realUserHome()
	if err != nil {
		return runFn()
	}
	before := snapshotUserShims(home)
	code := runFn()
	if violations := changedUserShims(home, before); len(violations) > 0 && code == 0 {
		reportHomeViolations(violations)
		return 1
	}
	return code
}

// realUserHome resolves the home directory as it stands before any test runs.
func realUserHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("testutil: resolve user home: %w", err)
	}
	return home, nil
}

func snapshotUserShims(home string) map[string]string {
	snapshot := make(map[string]string, len(userShimRelPaths))
	for _, rel := range userShimRelPaths {
		snapshot[rel] = fingerprint(filepath.Join(home, rel))
	}
	return snapshot
}

func changedUserShims(home string, before map[string]string) []string {
	var changed []string
	for _, rel := range userShimRelPaths {
		if fingerprint(filepath.Join(home, rel)) != before[rel] {
			changed = append(changed, rel)
		}
	}
	return changed
}

// fingerprint returns a content digest, or "absent" when the file does not
// exist, so both a rewrite and a fresh creation are detected.
func fingerprint(path string) string {
	raw, err := os.ReadFile(path) //nolint:gosec // path is derived from the user home directory
	if err != nil {
		return "absent"
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

func reportHomeViolations(violations []string) {
	fmt.Fprintln(os.Stderr, "FAIL: tests wrote to the real user home directory; isolate HOME (setHomeEnv) or disable the shim step:")
	for _, rel := range violations {
		fmt.Fprintf(os.Stderr, "  ~/%s\n", rel)
	}
}
