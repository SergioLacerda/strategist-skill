//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"gopkg.in/yaml.v3"
)

// TestConfidenceRolloutBlockingRequiresReviewedEvidence pins the contract's
// enforcement mode to the recorded observe-mode review: advisory needs
// nothing, blocking is only allowed when the referenced review validates.
func TestConfidenceRolloutBlockingRequiresReviewedEvidence(t *testing.T) {
	root := repoRoot(t)
	var contract struct {
		Rollout struct {
			Enforcement        string `yaml:"enforcement"`
			BlockingReviewFile string `yaml:"blocking_review_file"`
			Runbook            string `yaml:"runbook"`
		} `yaml:"rollout"`
	}
	raw := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "confidence-governance.yaml"))
	if err := yaml.Unmarshal([]byte(raw), &contract); err != nil {
		t.Fatal(err)
	}
	r := contract.Rollout
	if r.BlockingReviewFile == "" || r.Runbook == "" {
		t.Fatalf("rollout must name blocking_review_file and runbook: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(root, r.Runbook)); err != nil {
		t.Fatalf("rollout runbook missing: %v", err)
	}
	if r.Enforcement == telemetry.ConfidenceEnforcementAdvisory {
		return
	}
	review, err := telemetry.ReadObserveReview(filepath.Join(root, r.BlockingReviewFile))
	if err != nil {
		t.Fatalf("enforcement %q requires a recorded observe review: %v", r.Enforcement, err)
	}
	if err := telemetry.ValidateEnforcementChange(r.Enforcement, &review); err != nil {
		t.Fatalf("enforcement %q is not admissible: %v", r.Enforcement, err)
	}
}

func TestConfidenceRolloutBaselineRefusesBlocking(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	review, err := telemetry.ReadObserveReview(filepath.Join(root, "docs", "evidence", "confidence-observe-review.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := telemetry.ValidateEnforcementChange(telemetry.ConfidenceEnforcementBlocking, &review); err == nil {
		t.Fatal("the not_ready observe baseline must not authorize blocking")
	}
}
