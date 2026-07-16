//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// abilityContractFiles lists paths relative to a layer root that belong to the
// ability contract set — the files most likely to accumulate drift when the
// ability model changes (Quick Draw semantics, Opportunity Attack ownership,
// Side Quest routing).
func abilityContractFiles() []string {
	return []string{
		"skill.yaml",
		"personas/pragmatic.yaml",
		"internal_skills/ranger/SKILL.md",
		"internal_skills/sniper/SKILL.md",
		"contracts/machine/approval-gate.yaml",
		"contracts/machine/compliance-summary.yaml",
		"schemas/progress-contract.yaml",
	}
}

// --- Part A: Stale-string guards ---

// TestAbilityContractNoStaleQuickDrawOutputs ensures the old read/count Quick Draw
// model (total_ideas, similar_ideas) has been fully replaced by the write-only model
// (ideas_added, destination_path) across canonical, embedded defaults, and runtime.
func TestAbilityContractNoStaleQuickDrawOutputs(t *testing.T) {
	t.Parallel()

	roots := []string{
		filepath.Join(repoRoot(t), "strategist"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults"),
		isolatedStrategistDir(t),
	}
	forbidden := []string{"total_ideas", "similar_ideas"}

	for _, root := range roots {
		for _, rel := range abilityContractFiles() {
			path := filepath.Join(root, rel)
			data, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				t.Fatalf("read %s: %v", path, err)
			}
			content := string(data)
			for _, bad := range forbidden {
				if strings.Contains(content, bad) {
					t.Fatalf("%s contains stale Quick Draw output %q; replace with ideas_added/destination_path", path, bad)
				}
			}
		}
	}
}

// TestAbilityContractNoStaleApprovalGateRouting ensures the old adr_opportunity
// routing has been removed from approval-gate contracts in all layers.
func TestAbilityContractNoStaleApprovalGateRouting(t *testing.T) {
	t.Parallel()

	paths := []string{
		filepath.Join(repoRoot(t), "strategist", "contracts", "machine", "approval-gate.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "approval-gate.yaml"),
		filepath.Join(isolatedStrategistDir(t), "contracts", "machine", "approval-gate.yaml"),
	}
	forbidden := "next_phase: adr_opportunity"

	for _, path := range paths {
		content := readFile(t, path)
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s still routes to adr_opportunity on reject; replace with next_phase: closed", path)
		}
	}
}

// TestAbilityContractNoStaleImplementationIntentField ensures implementation_intent
// has been replaced by request_intent in agent-protocol templates and protocol docs.
func TestAbilityContractNoStaleImplementationIntentField(t *testing.T) {
	t.Parallel()

	paths := []string{
		filepath.Join(repoRoot(t), "strategist", "templates", "agent-protocol.md"),
		filepath.Join(repoRoot(t), "strategist", "protocol.md"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "agent-protocol.md"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "protocol.md"),
		filepath.Join(isolatedStrategistDir(t), "agent-protocol.md"),
		filepath.Join(isolatedStrategistDir(t), "templates", "agent-protocol.md"),
		filepath.Join(isolatedStrategistDir(t), "protocol.md"),
	}
	forbidden := "implementation_intent"

	for _, path := range paths {
		content := readFile(t, path)
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s still uses %q; replace with request_intent", path, forbidden)
		}
	}
}

// TestAbilityContractRangerDoesNotProduceOpportunityManifest ensures Ranger SKILL.md
// no longer references opportunity_manifest as a required artifact section.
// Archivist legitimately uses opportunity_manifest — this guard is Ranger-scoped only.
func TestAbilityContractRangerDoesNotProduceOpportunityManifest(t *testing.T) {
	t.Parallel()

	paths := []string{
		filepath.Join(repoRoot(t), "strategist", "internal_skills", "ranger", "SKILL.md"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills", "ranger", "SKILL.md"),
		filepath.Join(isolatedStrategistDir(t), "internal_skills", "ranger", "SKILL.md"),
	}
	forbidden := "opportunity_manifest"

	for _, path := range paths {
		content := readFile(t, path)
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s (Ranger) still references opportunity_manifest; Ranger must use scope_observations — Archivist owns opportunity_manifest", path)
		}
	}
}

// TestAbilityContractNoMandatoryOpportunityAttackForRangerOrSniper ensures neither
// Ranger nor Sniper SKILL.md requires opportunity_attack as a mandatory routine.
// Archivist is the sole owner of ADR Opportunity Attack evaluation.
func TestAbilityContractNoMandatoryOpportunityAttackForRangerOrSniper(t *testing.T) {
	t.Parallel()

	type roleCheck struct {
		path string
		role string
	}
	checks := []roleCheck{
		{filepath.Join(repoRoot(t), "strategist", "internal_skills", "ranger", "SKILL.md"), "ranger"},
		{filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills", "ranger", "SKILL.md"), "ranger"},
		{filepath.Join(isolatedStrategistDir(t), "internal_skills", "ranger", "SKILL.md"), "ranger"},
		{filepath.Join(repoRoot(t), "strategist", "internal_skills", "sniper", "SKILL.md"), "sniper"},
		{filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills", "sniper", "SKILL.md"), "sniper"},
		{filepath.Join(isolatedStrategistDir(t), "internal_skills", "sniper", "SKILL.md"), "sniper"},
	}
	forbidden := "Run opportunity_attack as a mandatory routine"

	for _, c := range checks {
		content := readFile(t, c.path)
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s (%s) still mandates opportunity_attack; Archivist is the sole owner — update to scope observation language", c.path, c.role)
		}
	}
}

// TestAbilityContractNoStaleOpportunityAttackCrossRoleRule ensures the old invariant
// requiring all three roles to each invoke opportunity_attack is not present in test files.
func TestAbilityContractNoStaleOpportunityAttackCrossRoleRule(t *testing.T) {
	t.Parallel()

	paths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "tests", "opportunity-attack.test.yaml"),
		filepath.Join(isolatedStrategistDir(t), "contracts", "tests", "opportunity-attack.test.yaml"),
	}
	forbidden := "roles [ranger, archivist, sniper] must each invoke opportunity_attack"

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("%s still requires all roles to invoke opportunity_attack; update to Archivist-only ownership model", path)
		}
	}
}

// --- Part B: Parity check ---

// TestAbilityContractParityCanonicalVsEmbedded ensures the ability contract files
// are byte-for-byte identical between canonical (strategist/) and embedded defaults
// (internal/embed/defaults/). Runtime (.strategist/) is generated/installed and
// may differ at the leaf level — it is not included in this parity check.
func TestAbilityContractParityCanonicalVsEmbedded(t *testing.T) {
	t.Parallel()

	canonical := filepath.Join(repoRoot(t), "strategist")
	embedded := filepath.Join(repoRoot(t), "internal", "embed", "defaults")

	for _, rel := range abilityContractFiles() {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			t.Parallel()

			canonicalPath := filepath.Join(canonical, rel)
			embeddedPath := filepath.Join(embedded, rel)

			canonicalData, err := os.ReadFile(canonicalPath)
			if err != nil {
				if os.IsNotExist(err) {
					t.Skipf("ability contract file %s absent in canonical — skipping parity", rel)
				}
				t.Fatalf("read canonical %s: %v", canonicalPath, err)
			}

			embeddedData, err := os.ReadFile(embeddedPath)
			if err != nil {
				if os.IsNotExist(err) {
					t.Fatalf("ability contract file %s exists in canonical but is missing from embedded defaults (%s)", rel, embeddedPath)
				}
				t.Fatalf("read embedded %s: %v", embeddedPath, err)
			}

			if string(canonicalData) != string(embeddedData) {
				t.Fatalf(
					"ability contract parity failure for %s:\n  canonical: %s\n  embedded:  %s\nSync embedded defaults from canonical or run `strategist compile`.",
					rel, canonicalPath, embeddedPath,
				)
			}
		})
	}
}
