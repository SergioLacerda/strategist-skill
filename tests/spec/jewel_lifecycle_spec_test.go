//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec tests for the treasure-chest-index-mine-pipeline mission (ADR-0012): the jewel
// lifecycle contract must consistently describe two public commands, four statuses, and a
// proposed-as-hint-only retrieval rule across every runtime mirror of context-enrichment.yaml.

func contextEnrichmentMirrors(t *testing.T) []string {
	t.Helper()
	return []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "context-enrichment.yaml"),
		filepath.Join(isolatedStrategistDir(t), "contracts", "machine", "context-enrichment.yaml"),
	}
}

// TestJewelSpec_NoPublicMultiCommandMiningWorkflow guards the offline organization
// plane's public command surface. Updated by treasure-chest-cli-unification
// (Decisions D2/D5): `mine` was removed and its curation responsibilities absorbed
// into `items` (list/show/accept/verify/deprecate/migrate-status subcommands) —
// see .analysis/refined/treasure-chest-cli-unification/design.md.
func TestJewelSpec_NoPublicMultiCommandMiningWorkflow(t *testing.T) {
	t.Parallel()

	for _, path := range contextEnrichmentMirrors(t) {
		content := readFile(t, path)

		if !strings.Contains(content, `"strategist treasure-chest index"`) {
			t.Errorf("%s: missing public command \"strategist treasure-chest index\"", path)
		}
		if !strings.Contains(content, `"strategist treasure-chest items"`) {
			t.Errorf("%s: missing public command \"strategist treasure-chest items\"", path)
		}
		if !strings.Contains(content, "internal:") || !strings.Contains(content, "- scan") {
			t.Errorf("%s: scan must be listed as an internal phase, not public UX", path)
		}
		for _, forbiddenPublicCmd := range []string{"treasure-chest polish", "treasure-chest pack"} {
			if strings.Contains(content, forbiddenPublicCmd) {
				t.Errorf("%s: %q must not exist as a command — polish/pack are conceptual internal steps or nonexistent, not a multi-command public workflow", path, forbiddenPublicCmd)
			}
		}
	}

	// The Go CLI itself must expose exactly index and items as the non-hidden subcommands
	// under `treasure-chest list/add/remove/index/items/doctor`; scan is folded in as Hidden,
	// and mine/jewel no longer exist as separate commands (renamed/removed, see above).
	scanSource := readFile(t, filepath.Join(repoRoot(t), "internal", "treasurecli", "treasure_chest_scan.go"))
	if !strings.Contains(scanSource, "Hidden: true") {
		t.Error("internal/treasurecli/treasure_chest_scan.go: scan command must be Hidden (internal phase, not public UX)")
	}
	for _, stale := range []string{"treasure_chest_mine.go", "treasure_chest_jewel.go"} {
		if _, err := os.Stat(filepath.Join(repoRoot(t), "internal", "treasurecli", stale)); err == nil {
			t.Errorf("internal/treasurecli/%s must not exist — removed/renamed into treasure_chest_items.go", stale)
		}
	}
}

func TestJewelSpec_NewJewelStatuses(t *testing.T) {
	t.Parallel()

	for _, path := range contextEnrichmentMirrors(t) {
		content := readFile(t, path)
		for _, status := range []string{"proposed", "accepted", "verified", "deprecated"} {
			if !strings.Contains(content, status) {
				t.Errorf("%s: missing jewel status %q", path, status)
			}
		}
	}

	statusFn := readFile(t, filepath.Join(repoRoot(t), "internal", "domain", "jewel_grade.go"))
	for _, status := range []string{`"proposed"`, `"accepted"`, `"verified"`, `"deprecated"`} {
		if !strings.Contains(statusFn, status) {
			t.Errorf("internal/domain/jewel_grade.go: missing jewel status constant %s", status)
		}
	}
}

func TestJewelSpec_ProposedUsedAsHintOnly(t *testing.T) {
	t.Parallel()

	for _, path := range contextEnrichmentMirrors(t) {
		content := readFile(t, path)
		if !strings.Contains(content, "proposed") || !strings.Contains(content, "hint") {
			t.Errorf("%s: jewel_retrieval must document proposed jewels as hint-only", path)
		}
		if !strings.Contains(content, "treating a status:proposed jewel as verified evidence") {
			t.Errorf("%s: jewel_retrieval forbidden list must explicitly forbid treating proposed as verified evidence", path)
		}
	}
}

func TestJewelSpec_EvidencePackRemainsMissionScoped(t *testing.T) {
	t.Parallel()

	for _, path := range contextEnrichmentMirrors(t) {
		content := readFile(t, path)
		if !strings.Contains(content, "<base_path>/<state>/<mission_id>-evidence-pack.md") {
			t.Errorf("%s: evidence_pack artifact_path must remain mission-scoped (<base_path>/<state>/<mission_id>-evidence-pack.md)", path)
		}
		if !strings.Contains(content, "mission-scoped audit") {
			t.Errorf("%s: evidence_pack description must state it is mission-scoped audit evidence, not a generic knowledge base", path)
		}
		if !strings.Contains(content, "never introduces a new retrieval unit") {
			t.Errorf("%s: evidence_pack must still state it never introduces a new retrieval unit (unaffected by jewel lifecycle changes)", path)
		}
	}
}

func TestJewelSpec_LegacyActiveRejectedAfterMigration(t *testing.T) {
	t.Parallel()

	statusFn := readFile(t, filepath.Join(repoRoot(t), "internal", "domain", "jewel_grade.go"))
	if !strings.Contains(statusFn, "jewelStatusLegacyActive") {
		t.Error("internal/domain/jewel_grade.go: must explicitly recognize and reject the legacy \"active\" status")
	}
	if !strings.Contains(statusFn, "migrate-status") {
		t.Error("internal/domain/jewel_grade.go: legacy active rejection must point at the migrate-status command")
	}

	itemsSource := readFile(t, filepath.Join(repoRoot(t), "internal", "treasurecli", "treasure_chest_items.go"))
	if !strings.Contains(itemsSource, "migrate-status") {
		t.Error("internal/treasurecli/treasure_chest_items.go: must expose a migrate-status subcommand for the one-time active -> accepted migration")
	}

	for _, path := range contextEnrichmentMirrors(t) {
		content := readFile(t, path)
		if !strings.Contains(content, `writing the legacy "active" status`) {
			t.Errorf("%s: jewel_generation forbidden list must call out the removed legacy \"active\" status", path)
		}
	}
}
