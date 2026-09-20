//go:build integration

package treasure_test

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/conformance"
)

// TestTreasureChestImportBoundary verifies that only a known, deliberately
// reviewed staging exceptions import Treasure Chest directly. The exceptions
// are inventory evidence, never proof that the Jewelcrafter delegate is live.
// A new external importer must either be a mediated delegate path or carry a
// complete, current staging exception under ADR-0049.
func TestTreasureChestImportBoundary(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	exceptions := []conformance.ImportException{
		{
			Importer: "github.com/SergioLacerda/strategist-skill/cmd/strategist",
			Owner:    "evaluation tooling", Reason: "eval harvest fixture generation",
			Scope: "cmd/strategist eval harvest", ReviewBy: time.Date(2026, time.November, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			Importer: "github.com/SergioLacerda/strategist-skill/internal/eval",
			Owner:    "evaluation tooling", Reason: "deterministic Treasure Chest validation",
			Scope: "internal/eval scope and policy scenarios", ReviewBy: time.Date(2026, time.November, 30, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, pkgOut := range []string{
		"github.com/SergioLacerda/strategist-skill/treasure-chest",
		"github.com/SergioLacerda/strategist-skill/treasure-chest/domain",
	} {
		assertReviewedImporters(t, pkgOut, reverseDirectImports(t, pkgOut), exceptions, now)
	}
}

func assertReviewedImporters(t *testing.T, target string, importers []string, exceptions []conformance.ImportException, now time.Time) {
	t.Helper()
	classifications := conformance.ClassifyTreasureChestImporters(importers, exceptions, now)
	for _, classification := range classifications {
		if classification.Disposition == conformance.ImportViolation {
			t.Errorf("%s imports %s — boundary violation: %s; route through Jewelcrafter or add a complete, time-bounded reviewed exception", classification.Importer, target, classification.Reason)
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
