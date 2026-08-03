//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSlotContractVocabularyConsistentAcrossSurfaces(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	for _, path := range []string{
		filepath.Join(root, "internal", "embed", "defaults", "skill.yaml"),
		filepath.Join(root, ".strategist", "skill.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"discovery:",
			"contract: write_analysis",
			"refinement:",
			"execution:",
			"contract: controlled",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing slot contract term %q", path, needle)
			}
		}
	}

	assertFileContains(t, filepath.Join(root, "docs", "adr", "0005-slot-write-contracts.md"), "The `write_pending` contract was discontinued.")
	assertFileContains(t, filepath.Join(root, "docs", "configuration.md"), "| `discovery` | `write_analysis` |")
	assertFileContains(t, filepath.Join(root, "docs", "configuration.md"), "| `execution` | `controlled` |")
	assertFileContains(t, filepath.Join(root, "docs", "c4-diagrams.md"), "| Slot `discovery` (Ranger) | pluggable | `write_analysis` |")
	assertFileContains(t, filepath.Join(root, "tests", "spec", "specs", "slot-contracts.feature"), "Roles: Ranger=write_analysis, Archivist=write_analysis, Sniper=controlled")
}

func TestGeneratedStatusBadgesAreNotStaticClaims(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "docs", "onboarding", "readme-en.md")
	content := readFile(t, path)
	for _, forbidden := range []string{"CI-passing", "version-1.0"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s contains static status badge claim %q", path, forbidden)
		}
	}
	for _, needle := range []string{
		"actions/workflows/test.yml/badge.svg",
		"img.shields.io/github/v/release/SergioLacerda/strategist-skill",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing dynamic badge %q", path, needle)
		}
	}
}

func TestArchitectureDocumentsCurrentInternalPackages(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "docs", "architecture.md")
	content := readFile(t, path)
	for _, pkg := range []string{"dojo", "governance", "integrity", "i18n", "runtimefs", "treasure", "validate"} {
		if !strings.Contains(content, "  "+pkg+"/") {
			t.Fatalf("%s missing internal package map entry for %s", path, pkg)
		}
	}
}

func assertFileContains(t *testing.T, path, needle string) {
	t.Helper()
	if content := readFile(t, path); !strings.Contains(content, needle) {
		t.Fatalf("%s missing %q", path, needle)
	}
}
