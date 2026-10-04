//go:build spec

package spec_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-0064 defines the current seven-family taxonomy for canonical documentation.
// ADR-0053 remains historical evidence and is intentionally outside this contract.
func TestCanonicalTaxonomyDocumentationDefinesAllFamilies(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	requiredByPath := map[string][]string{
		filepath.Join(root, "docs", "architecture", "strategist-concepts.md"): {
			"public vocabulary has seven families",
			"**Role**", "**Weapon**", "**Feat**", "**Tool**", "**Mechanism**", "**Stage**", "**Artifact**",
		},
		filepath.Join(root, "README.md"): {
			"canonical taxonomy has seven families",
			"Roles", "Weapons", "Feats", "Tools", "Mechanisms", "Stages", "Artifacts",
		},
		filepath.Join(root, "internal", "embed", "defaults", "SKILL.md"): {
			"seven public families",
			"Roles", "Weapons", "Feats", "Tools", "Mechanisms", "Stages", "Artifacts",
		},
	}
	for path, required := range requiredByPath {
		content := readFile(t, path)
		for _, term := range required {
			if !strings.Contains(content, term) {
				t.Errorf("%s missing canonical taxonomy family %q", path, term)
			}
		}
	}
}

func TestTaxonomyDocumentsLevelingAndRoleBoundaries(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	paths := []string{
		filepath.Join(root, "docs", "architecture", "strategist-concepts.md"),
		filepath.Join(root, "README.md"),
		filepath.Join(root, "internal", "embed", "defaults", "SKILL.md"),
	}
	required := []string{
		"LEVELING",
		"INITIATIVE",
		"Feat",
		"Tool",
		"Stage",
		"immutable operational resolver",
		"origin",
		"extensibility",
		"Pathfinder",
		"Cartographer",
		"Jeweler",
		"Jewelcrafter",
	}
	for _, path := range paths {
		content := readFile(t, path)
		for _, term := range required {
			if !strings.Contains(content, term) {
				t.Errorf("%s missing taxonomy boundary term %q", path, term)
			}
		}
	}
}

func TestCanonicalTaxonomySourceAndRuntimeMirrorsStayInParity(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	pairs := map[string]string{
		"SKILL.md": "SKILL.md",
		filepath.Join("contracts", "narrative", "00-routing.md"):   filepath.Join("contracts", "narrative", "00-routing.md"),
		filepath.Join("contracts", "narrative", "03-discovery.md"): filepath.Join("contracts", "narrative", "03-discovery.md"),
		filepath.Join("contracts", "narrative", "10-telemetry.md"): filepath.Join("contracts", "narrative", "10-telemetry.md"),
	}
	for sourceRel, runtimeRel := range pairs {
		source, err := os.ReadFile(filepath.Join(root, "internal", "embed", "defaults", sourceRel))
		if err != nil {
			t.Fatalf("read embedded source %s: %v", sourceRel, err)
		}
		runtime, err := os.ReadFile(filepath.Join(isolatedStrategistDir(t), runtimeRel))
		if err != nil {
			t.Fatalf("read runtime mirror %s: %v", runtimeRel, err)
		}
		if !bytes.Equal(source, runtime) {
			t.Errorf("runtime mirror %s differs from embedded source %s", runtimeRel, sourceRel)
		}
	}
}

func TestActiveRoleManifestsExcludeProposedTaxonomyNames(t *testing.T) {
	t.Parallel()

	root := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "roles")
	for _, roleID := range []string{"pathfinder", "cartographer", "jeweler", "jewelcrafter"} {
		if _, err := os.Stat(filepath.Join(root, roleID+".yaml")); err == nil {
			t.Errorf("proposed role %q must not have an active role manifest", roleID)
		} else if !os.IsNotExist(err) {
			t.Errorf("checking proposed role %q: %v", roleID, err)
		}
	}
}
