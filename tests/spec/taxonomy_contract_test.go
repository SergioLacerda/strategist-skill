//go:build spec

package spec_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalTaxonomyDocumentationDefinesAllFamilies(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	paths := []string{
		filepath.Join(root, "docs", "architecture", "strategist-concepts.md"),
		filepath.Join(root, "README.md"),
		filepath.Join(root, "docs", "adr", "0034-role-and-skill-taxonomy.md"),
		filepath.Join(root, "internal", "embed", "defaults", "SKILL.md"),
	}
	required := []string{
		"Roles",
		"Weapons",
		"Abilities",
		"Pipeline Services",
		"Mechanisms",
		"Routes",
		"Artifacts",
	}
	for _, path := range paths {
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
		filepath.Join(root, "docs", "adr", "0034-role-and-skill-taxonomy.md"),
		filepath.Join(root, "internal", "embed", "defaults", "SKILL.md"),
	}
	required := []string{
		"LEVELING",
		"INITIATIVE",
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
		runtime, err := os.ReadFile(filepath.Join(root, ".strategist", runtimeRel))
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
