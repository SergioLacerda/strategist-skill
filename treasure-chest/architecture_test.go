//go:build integration

package treasure_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestTreasureChestImportBoundary verifies that no production package imports
// Treasure Chest directly. The external provider is reached through the
// JEWELER-owned Weapon adapter; a new consumer must use that mediated path.
func TestTreasureChestImportBoundary(t *testing.T) {
	t.Parallel()

	for _, pkgOut := range []string{
		"github.com/SergioLacerda/strategist-skill/treasure-chest",
		"github.com/SergioLacerda/strategist-skill/treasure-chest/domain",
	} {
		for _, importer := range reverseDirectImports(t, pkgOut) {
			t.Errorf("%s imports %s directly; route access through JEWELER", importer, pkgOut)
		}
	}
}

// reverseDirectImports returns packages whose own .Imports list contains target.
// It deliberately does not use .Deps: transitive dependencies are not direct
// boundary crossings.
func reverseDirectImports(t *testing.T, target string) []string {
	t.Helper()

	out, err := exec.Command(
		"go", "list", "-f", "{{.ImportPath}}\t{{join .Imports \",\"}}", "./...",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps failed: %v\n%s", err, out)
	}

	var importers []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if importer, ok := reverseDirectImportLine(line, target); ok && !strings.HasPrefix(importer, "github.com/SergioLacerda/strategist-skill/treasure-chest") {
			importers = append(importers, importer)
		}
	}
	return importers
}

func reverseDirectImportLine(line, target string) (string, bool) {
	parts := strings.SplitN(line, "\t", 2)
	if len(parts) != 2 {
		return "", false
	}
	for _, dep := range strings.Split(parts[1], ",") {
		if dep == target {
			return parts[0], true
		}
	}
	return "", false
}
