//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestRoleLockDoesNotReferenceRemovedCapabilityCheck verifies the parent-agent
// Role Lock in SKILL.md no longer references the removed subtype/weapon
// manifest capability check — the configured discovery Weapon is required
// input to the fixed Ranger role, which invokes and normalizes its untrusted
// result without a native fallback.
func TestRoleLockDoesNotReferenceRemovedCapabilityCheck(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "SKILL.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"Discovery subtypes are selected by Scout and executed under the fixed Ranger role",
		"Ranger must invoke the configured discovery Weapon and normalize its untrusted",
		"There is no fallback",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing weapon-resolution term %q", path, needle)
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
// discovery uses the selected Weapon contract, so there is no subtype-specific
// capability gate left to check here.
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
// an incompatible discovery Weapon for any subtype.
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
// longer describes a post-route weapon-capability check — an incompatible
// discovery weapon is now a fatal error handled by Ranger's own normalization
// boundary, so there is no separate manifest-capability gate left to run
// after routing.
func TestRoutingContractOmitsPostRouteCapabilityCheck(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "00-routing.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"internal_skills/ranger",
		"hard error",
		"does not silently substitute another weapon",
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
// scout-routing.yaml has no separate post-route capability gate: the selected
// Weapon is validated by Ranger's invocation and normalization boundary.
func TestScoutRoutingMachineContractOmitsPostRouteCapabilityCheck(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "scout-routing.yaml")
	content := readFile(t, path)
	for _, required := range []string{
		"discovery_resolves_through_native_ranger",
		"active.slots.discovery is required input",
		"Ranger invokes and normalizes the Weapon",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("%s missing selected-weapon Ranger invariant %q", path, required)
		}
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
