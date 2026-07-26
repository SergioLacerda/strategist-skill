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
		filepath.Join(root, "internal", "embed", "defaults", "contracts", "strategist-raid.yaml"),
		filepath.Join(root, "internal", "embed", "defaults", "internal_skills", "strategist-raid", "skill.yaml"),
		filepath.Join(root, "internal", "embed", "defaults", "internal_skills", "strategist-raid", "SKILL.md"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing strategist-raid artifact %s: %v", path, err)
		}
	}
}

func TestStrategistSkillReferencesStrategistRaidContract(t *testing.T) {
	t.Parallel()

	// The exact sentence moved when W4 (token-economy) compacted the Contract Loading
	// Order section, but the reference itself must survive in some form.
	content := readFile(t, filepath.Join(repoRoot(t), "internal", "embed", "defaults", "SKILL.md"))
	if !strings.Contains(content, "strategist-raid.yaml") || !strings.Contains(content, "/strategist-raid") {
		t.Fatalf("internal/embed/defaults/SKILL.md missing /strategist-raid routing reference")
	}
}
