//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestRoleLockRequiresSubtypeCapabilityCheck verifies the parent-agent Role Lock
// blocks unsupported discovery subtype/weapon pairings before invoking the weapon.
func TestRoleLockRequiresSubtypeCapabilityCheck(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "SKILL.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Discovery subtypes are selected by Scout and executed through Ranger",
			"discovery_subtype_support",
			"provider_capability_mismatch",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing subtype capability-check term %q", path, needle)
			}
		}
	}
}

// TestPreflightContractDefinesProviderCapabilityMismatch verifies preflight.yaml
// documents the post-route provider/subtype mismatch block and honest remediation hint.

// TestPreflightContractDefinesProviderCapabilityMismatch verifies preflight.yaml
// documents the post-route provider/subtype mismatch block and honest remediation hint.
func TestPreflightContractDefinesProviderCapabilityMismatch(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "preflight.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"code: provider_capability_mismatch",
			"reason=provider_capability_mismatch",
			"scout-routing.yaml",
			"editing skill.yaml metadata alone does not change",
			"Verify with a live invocation",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing preflight provider_capability_mismatch term %q", path, needle)
			}
		}
	}
}

// TestDriftPatternsCoverProviderCapabilityMismatch verifies the normative
// drift-patterns.yaml teaches subtype-capability blocking on weapons.

// TestDriftPatternsCoverProviderCapabilityMismatch verifies the normative
// drift-patterns.yaml teaches subtype-capability blocking on weapons.
func TestDriftPatternsCoverProviderCapabilityMismatch(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "domain", "identity", "drift-patterns.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"id: provider_capability_mismatch",
			"editing skill.yaml metadata alone does not change",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing provider_capability_mismatch drift pattern term %q", path, needle)
			}
		}
	}
}

// TestSkillYamlStopConditionsIncludeProviderCapabilityMismatch verifies the
// master pipeline declares provider_capability_mismatch as a stop condition.

// TestSkillYamlStopConditionsIncludeProviderCapabilityMismatch verifies the
// master pipeline declares provider_capability_mismatch as a stop condition.
func TestSkillYamlStopConditionsIncludeProviderCapabilityMismatch(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skill.yaml"),
	} {
		content := readFile(t, path)
		if !strings.Contains(content, "provider_capability_mismatch") {
			t.Fatalf("%s stop_conditions must include provider_capability_mismatch", path)
		}
	}
}

// TestTelemetryContractDefinesScoutFields verifies 10-telemetry.md's canonical
// event payload lists Scout's route-decision fields and distinguishes them from
// Ranger's discovery-result events.

// TestRoutingContractDefinesPostRouteCapabilityCheck verifies 00-routing.md
// describes the weapon-capability check running immediately after Scout emits
// route_decision, before the discovery weapon is invoked.
func TestRoutingContractDefinesPostRouteCapabilityCheck(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "00-routing.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Post-Route Capability Check",
			"post_route_capability_check",
			"before the discovery weapon is invoked",
			"discovery_subtype_support",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing post-route capability check term %q", path, needle)
			}
		}
	}
}

// TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck verifies
// scout-routing.yaml defines the post_route_capability_check block that runs
// before weapon invocation, checking discovery_subtype_support.

// TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck verifies
// scout-routing.yaml defines the post_route_capability_check block that runs
// before weapon invocation, checking discovery_subtype_support.
func TestScoutRoutingMachineContractDefinesPostRouteCapabilityCheck(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "scout-routing.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"post_route_capability_check:",
			"discovery_subtype_support",
			"before_weapon_invocation",
			"applies_to_subtypes: [creative]",
			"this check does not run",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing post_route_capability_check term %q", path, needle)
			}
		}
	}
}

// TestJewelGenerationContractDefinesTrustCeiling verifies the jewel_generation contract
// (Track T-J / SQ-009) caps generation per consultation and enforces the trust-tier ceiling
// that replaced the human pre-approval gate for agent-generated jewels.
