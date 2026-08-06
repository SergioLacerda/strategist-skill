//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestCriticalHitClosureE2E_PlainMove and its siblings validate
// e2e-critical-hit-closure.feature at the Gherkin scenario granularity,
// distinct from critical_hit_closure_test.go (contract-text assertions)
// and tests/evals/contracts/critical_hit_closure_report_shape_valid_test.go
// (completion-report.md shape). See the feature file's own "Scope note".

func criticalHitClosureFeature(t *testing.T) string {
	t.Helper()
	return readFile(t, filepath.Join(testDir(t), "specs", "e2e-critical-hit-closure.feature"))
}

func TestCriticalHitClosureE2E_PlainMoveNeedsNoEvidence(t *testing.T) {
	t.Parallel()
	content := criticalHitClosureFeature(t)
	for _, needle := range []string{
		"Scenario: plain move requires no evaluation and no evidence",
		"no completion or validation claim is present",
		"Strategist relocates the artifact without requiring evidence",
		"no completion-report.md is written",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("e2e-critical-hit-closure.feature missing plain-move needle %q", needle)
		}
	}
}

func TestCriticalHitClosureE2E_ClosureMoveRequiresClaimAndEvidence(t *testing.T) {
	t.Parallel()
	content := criticalHitClosureFeature(t)
	for _, needle := range []string{
		"Scenario: closure move requires an explicit claim and supplied evidence",
		"the user supplies an explicit completion/validation claim",
		"the user supplies an evidence summary",
		"Strategist writes completion-report.md inside the source package",
		"the package is moved to done/",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("e2e-critical-hit-closure.feature missing closure-move needle %q", needle)
		}
	}
}

func TestCriticalHitClosureE2E_NotInferredFromDocumentationAppliedAlone(t *testing.T) {
	t.Parallel()
	content := criticalHitClosureFeature(t)
	for _, needle := range []string{
		"Scenario: closure move is never inferred from documentation_applied alone",
		"no explicit completion/validation claim was supplied",
		"Strategist does not treat the package as a closure candidate",
		"the package remains in refined/ as the normal terminal state",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("e2e-critical-hit-closure.feature missing non-inference needle %q", needle)
		}
	}
}
