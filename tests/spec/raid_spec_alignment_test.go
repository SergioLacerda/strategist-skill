//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRaidSkillArtifactsPresent(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	paths := []string{
		filepath.Join(root, "strategist", "contracts", "raid.yaml"),
		filepath.Join(root, "strategist", "skills", "raid", "skill.yaml"),
		filepath.Join(root, "strategist", "skills", "raid", "SKILL.md"),
		filepath.Join(root, "internal", "embed", "defaults", "skills", "raid", "skill.yaml"),
		filepath.Join(root, "internal", "embed", "defaults", "skills", "raid", "SKILL.md"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing raid artifact %s: %v", path, err)
		}
	}
}

func TestStrategistSkillReferencesRaidContract(t *testing.T) {
	t.Parallel()

	content := readFile(t, filepath.Join(repoRoot(t), "strategist", "SKILL.md"))
	if !strings.Contains(content, "For `/raid` (batch refinement of captured ideas), see `contracts/raid.yaml`.") {
		t.Fatalf("strategist/SKILL.md missing /raid routing reference")
	}
}
