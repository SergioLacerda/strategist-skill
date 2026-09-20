//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfidenceGovernanceContractDefinesPolicyAndIndependentMetrics(t *testing.T) {
	t.Parallel()
	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "confidence-governance.yaml")
	content := readFile(t, path)
	for _, needle := range []string{
		"version: v1",
		"low: [0, 59]",
		"medium: [60, 84]",
		"high: [85, 100]",
		"question_preservation",
		"assertion_evidence",
		"route_confidence",
		"critic_score",
		"approval_independence",
		"enforcement: advisory",
		"calibration_minimum_sample: 3",
		"denominator_policy:",
		"sample_unit:",
		"missing_records:",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing confidence governance term %q", path, needle)
		}
	}
}

func TestConfidenceGovernanceCanonicalizesEvidenceAndGroundTruth(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	claim := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "schemas", "confidence-claim.schema.yaml"))
	for _, needle := range []string{"agent:", "correlation_key:", "evidence_ids:", "evidence_classes:", "ground_truth_outcome:", "correct", "incorrect"} {
		if !strings.Contains(claim, needle) {
			t.Fatalf("confidence claim schema missing %q", needle)
		}
	}
}

func TestConfidenceHandoffsPreserveQuestionsEvidenceAndCalibration(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, rel := range []string{
		"internal/embed/defaults/schemas/handoff-ranger-to-archivist.schema.yaml",
		"internal/embed/defaults/schemas/handoff-archivist-to-sniper.schema.yaml",
	} {
		path := filepath.Join(root, rel)
		content := readFile(t, path)
		for _, needle := range []string{"confidence_summary:", "required: true", "claim_kind", "evidence_ids", "open_questions:", "sample_size:", "calibration_status:", "missing_record:", "no_sample"} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing handoff confidence term %q", path, needle)
			}
		}
	}
}

func TestApprovalGateConfidenceDoesNotAuthorizeExecution(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	machine := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "approval-gate.yaml"))
	narrative := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "narrative", "05-approval-gate.md"))
	for _, needle := range []string{"confidence_display:", "low_or_unsupported_assertion: review", "confidence summary is advisory input and cannot auto-accept"} {
		if !strings.Contains(machine, needle) {
			t.Fatalf("approval gate machine contract missing %q", needle)
		}
	}
	for _, needle := range []string{"Confidence is a review signal, not an approval", "Confidence cannot invoke Sniper", "implementation_handoff"} {
		if !strings.Contains(narrative, needle) {
			t.Fatalf("approval gate narrative missing %q", needle)
		}
	}
}

func TestConfidenceSchemaAndTelemetryVocabularyArePublished(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, rel := range []string{
		"internal/embed/defaults/schemas/confidence-claim.schema.yaml",
		"internal/embed/defaults/schemas/telemetry-event.schema.yaml",
		"internal/embed/defaults/contracts/narrative/10-telemetry.md",
	} {
		content := readFile(t, filepath.Join(root, rel))
		if !strings.Contains(content, "confidence_percent") || !strings.Contains(content, "calibration_status") {
			t.Fatalf("%s missing shared confidence telemetry vocabulary", rel)
		}
	}
}

func TestConfidenceProducerRoleContractsDeclareCoverageBehavior(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, rel := range []string{"internal/embed/defaults/roles/ranger.yaml", "internal/embed/defaults/roles/archivist.yaml", "internal/embed/defaults/roles/sniper.yaml"} {
		content := readFile(t, filepath.Join(root, rel))
		if !strings.Contains(content, "confidence_summary") {
			t.Fatalf("%s missing confidence_summary role behavior", rel)
		}
	}
}

// confidenceProducerContracts are the boundary contracts that must publish the
// `strategist metrics record` producer command.
var confidenceProducerContracts = []string{
	"roles/ranger.yaml",
	"roles/archivist.yaml",
	"roles/sniper.yaml",
	"internal_skills/response-critic/skill.yaml",
	"contracts/machine/scout-routing.yaml",
	"contracts/machine/mission-quality.yaml",
}

func TestConfidenceProducerContractsPublishCLI(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, rel := range confidenceProducerContracts {
		defaults := readFile(t, filepath.Join(root, "internal", "embed", "defaults", rel))
		if !strings.Contains(defaults, "strategist metrics record") {
			t.Fatalf("%s does not publish the CLI producer", rel)
		}
	}
}

// TestConfidenceProducerContractsRuntimeParityWhenInstalled checks the local
// .strategist mirror only when it exists: .strategist is git-ignored, so CI
// runners do not have it (same rule as the other WhenPresent parity tests).
func TestConfidenceProducerContractsRuntimeParityWhenInstalled(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	if _, err := os.Stat(filepath.Join(root, ".strategist")); os.IsNotExist(err) {
		t.Skip(".strategist runtime not installed in this workspace")
	}
	for _, rel := range confidenceProducerContracts {
		defaults := readFile(t, filepath.Join(root, "internal", "embed", "defaults", rel))
		mirror := readFile(t, filepath.Join(root, ".strategist", rel))
		if defaults != mirror {
			t.Fatalf("confidence producer mirror drift for %s", rel)
		}
	}
}
