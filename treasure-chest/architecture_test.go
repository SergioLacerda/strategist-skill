//go:build integration

package treasure_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestTreasureChestImportBoundary verifies that only a known, deliberately
// reviewed set of packages import treasure-chest or treasure-chest/domain.
// This is the isolation guarantee ADR-0040 (Treasure Chest In-Repo Isolation
// Staging) asks for: the package boundary is the seam a future Jewelcrafter
// skill implementation can mediate through without every existing caller
// needing to change. A new, unlisted importer showing up here means someone
// started depending on treasure-chest directly — review whether that
// dependency belongs in allowedTreasureChestImporters (tooling/build/eval
// infrastructure) or should instead go through Jewelcrafter once that skill
// has a Go implementation (see docs/adr/0040-treasure-chest-in-repo-isolation-staging.md).
//
// Mirrors the go-list-deps pattern already used by
// internal/domain/architecture_test.go, applied in reverse: that test
// asserts what one package must NOT depend on; this test asserts who is
// allowed to depend on treasure-chest.
func TestTreasureChestImportBoundary(t *testing.T) {
	t.Parallel()

	// allowedTreasureChestImporters are packages outside treasure-chest/**
	// with a reviewed, legitimate reason to import it directly today:
	//   - cmd/strategist: the `eval harvest` CLI command
	//     (eval_harvest_select.go) reads treasure-chest mission scan data to
	//     build eval fixtures — this is build/eval tooling, not agent-facing
	//     mission behavior.
	//   - internal/eval: the eval harness's scope-filter and policy-scenario
	//     dispatchers (scope_filter.go, harness_policy_scenarios.go) exercise
	//     treasure-chest's own validation functions as part of the
	//     deterministic eval suite — also tooling, not mission-time chest
	//     consultation (which stays agent-narrative, reading files directly,
	//     independent of this Go boundary).
	// Mission-time chest consultation (Ranger's consult_treasure_chests
	// ability) does not import this package at all — it reads
	// .strategist/treasure-chests.yaml and jewel/potion files as plain text.
	allowedTreasureChestImporters := map[string]bool{
		"github.com/SergioLacerda/strategist-skill/cmd/strategist": true,
		"github.com/SergioLacerda/strategist-skill/internal/eval":  true,
		// treasure-chest/cli is part of treasure-chest itself, not an outside importer.
		"github.com/SergioLacerda/strategist-skill/treasure-chest/cli": true,
	}

	for _, pkgOut := range []string{
		"github.com/SergioLacerda/strategist-skill/treasure-chest",
		"github.com/SergioLacerda/strategist-skill/treasure-chest/domain",
	} {
		assertAllowedImporters(t, pkgOut, reverseDeps(t, pkgOut), allowedTreasureChestImporters)
	}
}

func assertAllowedImporters(t *testing.T, target string, importers []string, allowed map[string]bool) {
	t.Helper()
	for _, importer := range importers {
		if importer == target || strings.HasPrefix(importer, "github.com/SergioLacerda/strategist-skill/treasure-chest") {
			continue
		}
		if !allowed[importer] {
			t.Errorf("%s imports %s — not in allowedTreasureChestImporters; if this is deliberate, add it there with a documented reason, otherwise route through Jewelcrafter instead", importer, target)
		}
	}
}

// reverseDeps returns every package in this module whose own `go list -deps`
// output includes target.
func reverseDeps(t *testing.T, target string) []string {
	t.Helper()

	out, err := exec.Command(
		"go", "list", "-f", "{{.ImportPath}}\t{{join .Deps \",\"}}", "./...",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps failed: %v\n%s", err, out)
	}

	var importers []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if importer, ok := reverseDepLine(line, target); ok {
			importers = append(importers, importer)
		}
	}
	return importers
}

func reverseDepLine(line, target string) (string, bool) {
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
