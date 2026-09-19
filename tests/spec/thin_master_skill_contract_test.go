//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThinMasterSkillRetainsAuditableControls(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	path := filepath.Join(root, "internal", "embed", "defaults", "SKILL.md")
	content := readFile(t, path)

	for _, marker := range []string{
		"strategist check --json",
		"Role Lock — Parent Agent Contract",
		"error=role_invocation_failed",
		"internal/embed/defaults/",
		".strategist/",
		"base_path",
		"Approval Gate",
		"MUST NOT",
		"contracts/index.yaml",
	} {
		if !strings.Contains(content, marker) {
			t.Fatalf("thin master skill missing retained control %q", marker)
		}
	}

	if lines := strings.Count(content, "\n") + 1; lines > 180 {
		t.Fatalf("master skill is not thin: %d lines", lines)
	}
}

func TestThinMasterSkillRuntimeParityWhenInstalled(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	runtimePath := filepath.Join(root, ".strategist", "SKILL.md")
	runtime, err := os.ReadFile(runtimePath)
	if os.IsNotExist(err) {
		t.Skip(".strategist runtime not installed")
	}
	if err != nil {
		t.Fatalf("read runtime master skill: %v", err)
	}
	authoring := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "SKILL.md"))
	if string(runtime) != authoring {
		t.Fatalf("runtime SKILL.md drifted from canonical authoring source")
	}
}

func TestThinMasterSkillOwnershipMatrixExists(t *testing.T) {
	t.Parallel()
	path := filepath.Join(repoRoot(t), "docs", "design", "phase-a-thin-master-skill-ownership.md")
	content := readFile(t, path)
	for _, marker := range []string{"Disposition", "Canonical owner", "Regression evidence", "rollback", "soft-profile"} {
		if !strings.Contains(content, marker) {
			t.Fatalf("ownership matrix missing %q", marker)
		}
	}
}
