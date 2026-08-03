//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestHandoffChallengeSchemaAndPolicy(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	schema := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "schemas", "handoff-archivist-to-sniper.schema.yaml"))
	machine := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "handoff-contract.yaml"))

	for _, needle := range []string{
		"handoff_verification",
		"archivist_to_sniper",
		"[objective, boundary, classification, gate]",
		"return_to_archivist",
	} {
		if !strings.Contains(schema, needle) {
			t.Fatalf("handoff schema missing %q", needle)
		}
	}
	for _, needle := range []string{
		"handoff_verification_policy",
		"semantic handoff challenge",
		"required refs",
		"passing handoff_verification never grants execution by itself",
	} {
		if !strings.Contains(machine, needle) {
			t.Fatalf("handoff contract missing %q", needle)
		}
	}
}

func TestHandoffChallengeBlocksBeforeSniperMaterialization(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	execution := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "narrative", "06-execution.md"))
	sniper := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "internal_skills", "sniper", "SKILL.md"))
	errors := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "errors.yaml"))

	for _, doc := range []struct {
		name    string
		content string
	}{
		{"06-execution.md", execution},
		{"sniper/SKILL.md", sniper},
	} {
		for _, needle := range []string{
			"Handoff Challenge",
			"blocked reason=handoff_challenge_missing",
			"blocked reason=handoff_challenge_failed",
			"blocked reason=handoff_challenge_repair_required",
			"never replaces Approval Gate",
		} {
			if !strings.Contains(doc.content, needle) {
				t.Fatalf("%s missing %q", doc.name, needle)
			}
		}
	}
	for _, token := range []string{
		"token: handoff_challenge_missing",
		"token: handoff_challenge_failed",
		"token: handoff_challenge_repair_required",
	} {
		if !strings.Contains(errors, token) {
			t.Fatalf("errors catalog missing %q", token)
		}
	}
}

func TestHandoffChallengeDoesNotReplaceApprovalGate(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	gate := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "narrative", "05-approval-gate.md"))
	execution := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "narrative", "06-execution.md"))

	for _, needle := range []string{
		"Handoff Challenge Independence",
		"That challenge is not an approval mechanism",
		"Passing the challenge never bypasses this gate",
	} {
		if !strings.Contains(gate, needle) {
			t.Fatalf("approval gate contract missing %q", needle)
		}
	}
	if !strings.Contains(execution, "Sniper still requires `mission_status: gate_analysis_accepted`") {
		t.Fatalf("execution contract must keep Approval Gate status independent")
	}
}

func TestHandoffChallengeTelemetryContract(t *testing.T) {
	t.Parallel()

	telemetry := readFile(t, filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "10-telemetry.md"))
	for _, needle := range []string{
		"`handoff_challenge.status`",
		"`handoff_challenge.critical_failures`",
		"`handoff_challenge.types`",
		"`strategist.handoff_challenge.status`",
		"never imply Approval Gate acceptance",
	} {
		if !strings.Contains(telemetry, needle) {
			t.Fatalf("telemetry contract missing %q", needle)
		}
	}
}

func TestHandoffChallengeFixtures(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"handoff-challenge-pass.yaml",
		"handoff-challenge-missing-ack.yaml",
		"handoff-challenge-misclassified-question.yaml",
		"handoff-challenge-boundary-violation.yaml",
		"handoff-challenge-low-risk-skip.yaml",
	} {
		fixture := readFixture(t, filepath.Join(testDir(t), "fixtures", name))
		if !strings.Contains(fixture.Scenario, "handoff_challenge") {
			t.Fatalf("%s must be a handoff challenge fixture", name)
		}
		if !strings.Contains(fixture.ExpectedEvent, "handoff_challenge") {
			t.Fatalf("%s expected_event must mention handoff_challenge", name)
		}
	}
}
