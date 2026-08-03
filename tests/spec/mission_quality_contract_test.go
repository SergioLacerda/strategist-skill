//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMissionQualityContractWording(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	machine := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "mission-quality.yaml"))
	gate := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "approval-gate.yaml"))

	for _, needle := range []string{
		"unsupported_claims: forbidden",
		"fact_inference_separation: required",
		"traceable_findings: required",
		"acceptance_criteria: required",
		"unresolved_questions_preserved: required",
		"source_scope_respected: required",
		"advisory only",
		"decisions:/evidence: sections are optional in every mission — never required",
	} {
		if !strings.Contains(machine, needle) {
			t.Fatalf("mission-quality contract missing %q", needle)
		}
	}

	if !strings.Contains(gate, "mission_quality_display") {
		t.Fatal("approval-gate contract missing mission_quality_display block")
	}
	if !strings.Contains(gate, "on_no_decisions_or_evidence: omit_line") {
		t.Fatal("approval-gate contract's mission_quality_display must omit the line, not block, when no decisions/evidence are present")
	}
}

func TestDecisionEvidenceRecordingIsOptionalInNarrativeContracts(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	discovery := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "narrative", "03-discovery.md"))
	refinement := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "narrative", "04-refinement.md"))

	if !strings.Contains(discovery, "### Optional Evidence Recording") {
		t.Fatal("03-discovery.md missing Optional Evidence Recording section")
	}
	if !strings.Contains(discovery, "Ranger MAY record") {
		t.Fatal("03-discovery.md must state evidence recording is optional (MAY), not required")
	}

	if !strings.Contains(refinement, "### Optional Decision Ledger") {
		t.Fatal("04-refinement.md missing Optional Decision Ledger section")
	}
	if !strings.Contains(refinement, "Archivist MAY consolidate") {
		t.Fatal("04-refinement.md must state decision consolidation is optional (MAY), not required")
	}
}

func TestMissionQualityContractIsAdvisoryNotBlocking(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	machine := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "mission-quality.yaml"))

	for _, needle := range []string{
		"mission_quality is advisory only",
		"never blocks a mission",
		"never substitutes",
		"this contract does not define a CLI command or a Go call site",
	} {
		if !strings.Contains(machine, needle) {
			t.Fatalf("mission-quality contract missing invariant wording %q", needle)
		}
	}
}
