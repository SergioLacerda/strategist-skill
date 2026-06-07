//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStrategistRaidSkillArtifactsPresent(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	paths := []string{
		filepath.Join(root, "strategist", "contracts", "strategist-raid.yaml"),
		filepath.Join(root, "strategist", "skills", "strategist-raid", "skill.yaml"),
		filepath.Join(root, "strategist", "skills", "strategist-raid", "SKILL.md"),
		filepath.Join(root, "internal", "embed", "defaults", "skills", "strategist-raid", "skill.yaml"),
		filepath.Join(root, "internal", "embed", "defaults", "skills", "strategist-raid", "SKILL.md"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing strategist-raid artifact %s: %v", path, err)
		}
	}
}

func TestStrategistSkillReferencesStrategistRaidContract(t *testing.T) {
	t.Parallel()

	content := readFile(t, filepath.Join(repoRoot(t), "strategist", "SKILL.md"))
	if !strings.Contains(content, "For `/strategist-raid` (batch refinement of captured ideas), see `contracts/strategist-raid.yaml`.") {
		t.Fatalf("strategist/SKILL.md missing /strategist-raid routing reference")
	}
}
