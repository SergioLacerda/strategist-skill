package telemetry

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Confidence enforcement modes. Advisory is the default; blocking is only
// admissible with a reviewed observe-mode record (confidence-governance.yaml
// #rollout.blocking_requires).
const (
	ConfidenceEnforcementAdvisory = "advisory"
	ConfidenceEnforcementBlocking = "blocking"
)

// ObserveReview is the recorded human review of an observe-mode run: the
// compatibility evidence that must exist before enforcement can block.
type ObserveReview struct {
	SchemaVersion               string         `yaml:"schema_version"`
	ReviewedBy                  string         `yaml:"reviewed_by"`
	ReviewedAt                  string         `yaml:"reviewed_at"`
	CompatibilityEvidenceRef    string         `yaml:"compatibility_evidence_ref"`
	AgentDenominatorsReconciled bool           `yaml:"agent_denominators_reconciled"`
	SamplesPerAgent             map[string]int `yaml:"samples_per_agent"`
}

// ValidateObserveReview checks the review is complete and internally sound.
func ValidateObserveReview(r ObserveReview) error {
	var errs []error
	for name, value := range map[string]string{
		"reviewed_by": r.ReviewedBy, "compatibility_evidence_ref": r.CompatibilityEvidenceRef,
	} {
		if value == "" {
			errs = append(errs, fmt.Errorf("observe review: %s is required", name))
		}
	}
	if _, err := time.Parse(time.RFC3339, r.ReviewedAt); err != nil {
		errs = append(errs, fmt.Errorf("observe review: reviewed_at %q is not RFC3339", r.ReviewedAt))
	}
	if !r.AgentDenominatorsReconciled {
		errs = append(errs, errors.New("observe review: per-agent denominators are not reconciled"))
	}
	if len(r.SamplesPerAgent) == 0 {
		errs = append(errs, errors.New("observe review: samples_per_agent is required"))
	}
	return errors.Join(errs...)
}

// ReadObserveReview loads a review file.
func ReadObserveReview(path string) (ObserveReview, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // operator-supplied evidence file
	if err != nil {
		return ObserveReview{}, fmt.Errorf("observe review: read: %w", err)
	}
	var review ObserveReview
	if err := yaml.Unmarshal(raw, &review); err != nil {
		return ObserveReview{}, fmt.Errorf("observe review: parse: %w", err)
	}
	return review, nil
}

// ValidateEnforcementChange decides whether a requested enforcement mode is
// admissible. Advisory always is (rollback never needs evidence and never
// touches records or the human gate); blocking needs a valid review.
func ValidateEnforcementChange(requested string, review *ObserveReview) error {
	switch requested {
	case ConfidenceEnforcementAdvisory:
		return nil
	case ConfidenceEnforcementBlocking:
		if review == nil {
			return errors.New("enforcement stays advisory: blocking requires a recorded observe-mode review")
		}
		if err := ValidateObserveReview(*review); err != nil {
			return fmt.Errorf("enforcement stays advisory: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("enforcement %q is not allowed (advisory | blocking)", requested)
	}
}
