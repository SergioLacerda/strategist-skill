//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestTelemetryContractDocumentsChestEventDistinction verifies the telemetry contract
// (Track T-D1) documents treasure_chest_loaded vs treasure_chest_found as intentionally
// distinct events rather than leaving them as unexplained naming drift.
func TestTelemetryContractDocumentsChestEventDistinction(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "10-telemetry.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Chest Event Naming",
			"intentionally distinct events",
			"No rename or consolidation without an explicit approval-gate extension",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing chest event naming classification term %q", path, needle)
			}
		}
	}
}

// TestScoutRouteDecisionSchemaExists verifies the Scout route-decision schema
// file exists in both the canonical source tree and the embedded defaults tree.

// TestTelemetryContractDefinesScoutFields verifies 10-telemetry.md's canonical
// event payload lists Scout's route-decision fields and distinguishes them from
// Ranger's discovery-result events.
func TestTelemetryContractDefinesScoutFields(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "10-telemetry.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"`role`",
			"`route`",
			"`route_reason`",
			"`route_confidence`",
			"`evidence_state`",
			"`discovery_subtype`",
			"`provider`",
			"component: scout",
			"component: ranger",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing scout telemetry field %q", path, needle)
			}
		}
	}
}

// TestRoutingContractDefinesPostRouteCapabilityCheck verifies 00-routing.md
// describes the weapon-capability check running immediately after Scout emits
// route_decision, before the discovery weapon is invoked.
