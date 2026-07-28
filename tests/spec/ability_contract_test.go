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
// ability model changes (Opportunity Attack ownership, Side Quest routing).
func abilityContractFiles() []string {
	return []string{
		"skill.yaml",
		"personas/pragmatic.yaml",
		"internal_skills/ranger/SKILL.md",
		"internal_skills/sniper/SKILL.md",
		"contracts/machine/critical-hit.yaml",
		"contracts/machine/opportunity-attack.yaml",
		"contracts/machine/approval-gate.yaml",
		"contracts/machine/compliance-summary.yaml",
		"contracts/narrative/07-adr.md",
		"contracts/narrative/11-critical-hit.md",
		"schemas/progress-contract.yaml",
	}
}

// --- Part A: Stale-string guards ---

// TestAbilityContractNoStaleReadCountOutputs ensures the old read/count idea-capture
// model (total_ideas, similar_ideas) has been fully replaced by the write-only model
// (ideas_added, destination_path) across canonical, embedded defaults, and runtime.
func TestAbilityContractNoStaleReadCountOutputs(t *testing.T) {
	t.Parallel()

	roots := []string{
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
					t.Fatalf("%s contains stale read/count output %q; replace with ideas_added/destination_path", path, bad)
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
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "agent-protocol.md"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "protocol.md"),
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
		{filepath.Join(repoRoot(t), "internal", "embed", "defaults", "internal_skills", "ranger", "SKILL.md"), "ranger"},
		{filepath.Join(isolatedStrategistDir(t), "internal_skills", "ranger", "SKILL.md"), "ranger"},
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

// TestAbilityContractOpportunityAttackIsADROnly ensures Opportunity Attack does
// not become a second card-closure or implementation-audit mechanism. Since
// strategist-ability-taxonomy-reorg (Decisions D3/D4), Opportunity Attack also
// evaluates runbook- and chest-worthiness alongside ADR-worthiness — three
// independent side quests — but none of the three may evaluate implementation
// completion or move cards to done/; that remains Critical Hit's exclusive remit.
func TestAbilityContractOpportunityAttackIsADROnly(t *testing.T) {
	t.Parallel()

	paths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "opportunity-attack.yaml"),
		filepath.Join(isolatedStrategistDir(t), "contracts", "machine", "opportunity-attack.yaml"),
	}
	required := []string{
		"Opportunity Attack MUST NOT evaluate implementation completion or move cards to done/",
		"Critical Hit is the only route that may close pending/refined cards into done/",
	}

	for _, path := range paths {
		content := readFile(t, path)
		for _, needle := range required {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing Opportunity Attack role boundary %q", path, needle)
			}
		}
	}
}

// TestAbilityContractCriticalHitOwnsCardClosure keeps finalized-card movement
// assigned to Critical Hit rather than ADR Opportunity Attack.
func TestAbilityContractCriticalHitOwnsCardClosure(t *testing.T) {
	t.Parallel()

	paths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "critical-hit.yaml"),
		filepath.Join(isolatedStrategistDir(t), "contracts", "machine", "critical-hit.yaml"),
	}
	required := []string{
		"Critical Hit owns pending/refined card closure into done/",
		"Opportunity Attack does not move analysis cards",
	}

	for _, path := range paths {
		content := readFile(t, path)
		for _, needle := range required {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing Critical Hit closure boundary %q", path, needle)
			}
		}
	}
}

// TestAbilityContractNoLegacyOpportunityFSMNaming ensures the generic side-quest
// gate does not reuse Opportunity Attack naming.
func TestAbilityContractNoLegacyOpportunityFSMNaming(t *testing.T) {
	t.Parallel()

	paths := []string{
		filepath.Join(repoRoot(t), "internal", "domain", "policy.go"),
		filepath.Join(repoRoot(t), "internal", "domain", "state_machine.go"),
		filepath.Join(repoRoot(t), "internal", "domain", "state_machine_test.go"),
	}
	forbidden := []string{
		"StateOpportunityAttack",
		"StateOpportunityGate",
		"StateOpportunityExec",
		"EventSniperOA",
		"sniper_opportunity_attack",
		"OPPORTUNITY_ATTACK",
		"OPPORTUNITY_GATE",
		"OPPORTUNITY_EXEC",
	}

	for _, path := range paths {
		content := readFile(t, path)
		for _, bad := range forbidden {
			if strings.Contains(content, bad) {
				t.Fatalf("%s still contains legacy Opportunity FSM name %q; use SideQuest naming for the generic gate", path, bad)
			}
		}
	}
}

// --- Part B: Parity check ---

// TestAbilityContractParityCanonicalVsEmbedded was removed in W7a (Option B):
// strategist/ was retired, so canonical and embedded defaults are the same tree
// and byte parity is true by construction. Runtime parity is covered by
// TestLocalRuntimeMirrorsCanonicalNormativeFilesWhenPresent in spec_alignment_test.go.
