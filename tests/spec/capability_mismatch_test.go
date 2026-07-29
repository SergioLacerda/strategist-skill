//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestRoleLockDoesNotReferenceRemovedCapabilityCheck verifies the parent-agent
// Role Lock in SKILL.md no longer references the removed subtype/weapon
// manifest capability check — discovery always resolves to
// internal_skills/ranger now (see .analysis/refined/20260728-ranger-drift-eval/).
func TestRoleLockDoesNotReferenceRemovedCapabilityCheck(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "SKILL.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"Discovery subtypes are selected by Scout and executed through Ranger",
		"external weapon is ever consulted as a substitute for Ranger",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing native-discovery term %q", path, needle)
		}
	}
	for _, forbidden := range []string{
		"discovery_subtype_support",
		"provider_capability_mismatch",
	} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s still references removed capability-check term %q", path, forbidden)
		}
	}
}

// TestPreflightContractOmitsProviderCapabilityMismatch verifies preflight.yaml
// no longer documents the removed post-route provider/subtype mismatch block —
// discovery never reaches an external weapon, so there is nothing left to check.
func TestPreflightContractOmitsProviderCapabilityMismatch(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "preflight.yaml")
	content := readFile(t, path)
	for _, forbidden := range []string{
		"code: provider_capability_mismatch",
		"reason=provider_capability_mismatch",
	} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s still documents removed provider_capability_mismatch term %q", path, forbidden)
		}
	}
}

// TestDriftPatternsCoverExternalDiscoveryWeaponRegression verifies the normative
// drift-patterns.yaml teaches the successor pattern: never regress to invoking
// an external weapon for discovery, for any subtype.
func TestDriftPatternsCoverExternalDiscoveryWeaponRegression(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "domain", "identity", "drift-patterns.yaml")
	content := readFile(t, path)
	for _, needle := range []string{
		"id: external_discovery_weapon_regression",
		"internal_skills/ranger",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing external_discovery_weapon_regression drift pattern term %q", path, needle)
		}
	}
	if strings.Contains(content, "id: provider_capability_mismatch") {
		t.Fatalf("%s still declares the removed provider_capability_mismatch drift pattern", path)
	}
}

// TestSkillYamlStopConditionsOmitProviderCapabilityMismatch verifies the master
// pipeline no longer declares provider_capability_mismatch as a stop condition —
// discovery has no external-weapon path left to fail that way.
func TestSkillYamlStopConditionsOmitProviderCapabilityMismatch(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skill.yaml")
	content := readFile(t, path)
	if strings.Contains(content, "provider_capability_mismatch") {
		t.Fatalf("%s stop_conditions must not include removed provider_capability_mismatch", path)
	}
}

// TestRoutingContractOmitsPostRouteCapabilityCheck verifies 00-routing.md no
// longer describes a post-route weapon-capability check — discovery always
// resolves to internal_skills/ranger, so there is no weapon invocation left to
// gate.
func TestRoutingContractOmitsPostRouteCapabilityCheck(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "00-routing.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"internal_skills/ranger",
		"kind=native_role",
		"never a live behavior guarantee",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing discovery weapon resolution term %q", path, needle)
		}
	}
	for _, forbidden := range []string{
		"### Post-Route Capability Check",
		"post_route_capability_check",
	} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s still documents removed post-route capability check term %q", path, forbidden)
		}
	}
}

// TestScoutRoutingMachineContractOmitsPostRouteCapabilityCheck verifies
// scout-routing.yaml no longer defines the post_route_capability_check block —
// it has no remaining caller once discovery never reaches an external weapon.
func TestScoutRoutingMachineContractOmitsPostRouteCapabilityCheck(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "scout-routing.yaml")
	content := readFile(t, path)
	if !strings.Contains(content, "discovery_always_resolves_to_native_ranger") {
		t.Fatalf("%s missing discovery_always_resolves_to_native_ranger invariant", path)
	}
	for _, forbidden := range []string{
		"post_route_capability_check:",
		"applies_to_subtypes: [creative]",
	} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s still defines removed post_route_capability_check term %q", path, forbidden)
		}
	}
}
