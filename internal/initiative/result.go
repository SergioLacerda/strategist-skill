package initiative

import (
	"fmt"
	"strings"
)

// CheckStatus is the outcome of a single obligation check reported in a
// Result.
type CheckStatus string

const (
	// CheckSatisfied indicates the obligation was fully met.
	CheckSatisfied CheckStatus = "satisfied"
	// CheckPartial indicates the obligation was only partly met.
	CheckPartial CheckStatus = "partial"
	// CheckBlocked indicates the obligation could not be met.
	CheckBlocked CheckStatus = "blocked"
	// CheckNotApplicable indicates the obligation did not apply.
	CheckNotApplicable CheckStatus = "not_applicable"
)

// EvidenceRef points to evidence supporting a check or outcome.
type EvidenceRef struct {
	ID          string `json:"id" yaml:"id"`
	Fingerprint string `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`
	Class       string `json:"class" yaml:"class"`
}

var validEvidenceClasses = map[string]struct{}{
	"explicit": {}, "corroborated_inference": {}, "weak_inference": {}, "unknown": {},
}

const maxInitiativeFieldBytes = 4096

// ObligationCheck records the status of a single obligation.
type ObligationCheck struct {
	ID           string        `json:"id" yaml:"id"`
	Status       CheckStatus   `json:"status" yaml:"status"`
	EvidenceRefs []EvidenceRef `json:"evidence_refs,omitempty" yaml:"evidence_refs,omitempty"`
	Reason       string        `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// Deviation records a departure from an obligation and its impact.
type Deviation struct {
	ObligationID string `json:"obligation_id" yaml:"obligation_id"`
	Impact       string `json:"impact" yaml:"impact"`
	Reason       string `json:"reason" yaml:"reason"`
}

// OutcomeCorrelation links a reported outcome back to its evidence.
type OutcomeCorrelation struct {
	ID           string        `json:"id" yaml:"id"`
	Status       string        `json:"status" yaml:"status"`
	EvidenceRefs []EvidenceRef `json:"evidence_refs,omitempty" yaml:"evidence_refs,omitempty"`
}

// Result is the role-reported outcome of acting on an Advice.
type Result struct {
	ResultID        string               `json:"result_id,omitempty" yaml:"result_id,omitempty"`
	Sequence        int                  `json:"sequence,omitempty" yaml:"sequence,omitempty"`
	Supersedes      string               `json:"supersedes,omitempty" yaml:"supersedes,omitempty"`
	AdviceID        string               `json:"advice_id" yaml:"advice_id"`
	MissionID       string               `json:"mission_id" yaml:"mission_id"`
	Role            string               `json:"role" yaml:"role"`
	RunID           string               `json:"run_id" yaml:"run_id"`
	Checks          []ObligationCheck    `json:"checks" yaml:"checks"`
	Deviations      []Deviation          `json:"deviations,omitempty" yaml:"deviations,omitempty"`
	EvidenceRefs    []EvidenceRef        `json:"evidence_refs,omitempty" yaml:"evidence_refs,omitempty"`
	Outcomes        []OutcomeCorrelation `json:"outcomes,omitempty" yaml:"outcomes,omitempty"`
	GateIndependent bool                 `json:"gate_independent" yaml:"gate_independent"`
}

// ResultAssessment is the outcome of evaluating a Result against its
// Advice.
type ResultAssessment struct {
	SchemaVersion     string             `json:"schema_version,omitempty" yaml:"schema_version,omitempty"`
	Mechanism         string             `json:"mechanism,omitempty" yaml:"mechanism,omitempty"`
	AlgorithmVersion  string             `json:"algorithm_version,omitempty" yaml:"algorithm_version,omitempty"`
	AssessmentID      string             `json:"assessment_id,omitempty" yaml:"assessment_id,omitempty"`
	InputDigest       string             `json:"input_digest,omitempty" yaml:"input_digest,omitempty"`
	AdviceID          string             `json:"advice_id,omitempty" yaml:"advice_id,omitempty"`
	PolicyVersion     string             `json:"policy_version,omitempty" yaml:"policy_version,omitempty"`
	PolicyDigest      string             `json:"policy_digest,omitempty" yaml:"policy_digest,omitempty"`
	ResultID          string             `json:"result_id,omitempty" yaml:"result_id,omitempty"`
	ResultSequence    int                `json:"result_sequence,omitempty" yaml:"result_sequence,omitempty"`
	CalibrationStatus string             `json:"calibration_status,omitempty" yaml:"calibration_status,omitempty"`
	Challenge         bool               `json:"challenge" yaml:"challenge"`
	Assessed          ConfidenceTier     `json:"assessed,omitempty" yaml:"assessed,omitempty"`
	Ceiling           ConfidenceTier     `json:"ceiling,omitempty" yaml:"ceiling,omitempty"`
	ConfidenceCeiling ConfidenceTier     `json:"confidence_ceiling" yaml:"confidence_ceiling"`
	Effective         ConfidenceTier     `json:"effective,omitempty" yaml:"effective,omitempty"`
	Reasons           []string           `json:"reasons,omitempty" yaml:"reasons,omitempty"`
	EvidenceSummary   EvidenceSummary    `json:"evidence_summary,omitempty" yaml:"evidence_summary,omitempty"`
	Escalation        *EscalationRequest `json:"escalation,omitempty" yaml:"escalation,omitempty"`
}

// ValidateAgainst checks that the result correlates with and satisfies the
// structural requirements of the given advice.
func (r Result) ValidateAgainst(advice Advice) error {
	if err := r.validateShape(advice); err != nil {
		return err
	}
	if err := validateChecks(r.Checks, advice.Diligence.Checks); err != nil {
		return err
	}
	if err := validateEvidence(r); err != nil {
		return err
	}
	if err := validateDeviations(r.Deviations); err != nil {
		return err
	}
	if err := r.validateRevision(); err != nil {
		return err
	}
	return validateOutcomes(r.Outcomes)
}

func validateDeviations(deviations []Deviation) error {
	seen := make(map[string]struct{}, len(deviations))
	for _, deviation := range deviations {
		if err := validateDeviation(deviation); err != nil {
			return err
		}
		if _, exists := seen[deviation.ObligationID]; exists {
			return fmt.Errorf("initiative_result_invalid: duplicate deviation %q", deviation.ObligationID)
		}
		seen[deviation.ObligationID] = struct{}{}
	}
	return nil
}

func validateDeviation(deviation Deviation) error {
	fields := []string{deviation.ObligationID, deviation.Impact, deviation.Reason}
	for _, field := range fields {
		if strings.TrimSpace(field) == "" {
			return fmt.Errorf("initiative_result_invalid: deviation obligation_id, impact, and reason are required")
		}
	}
	for _, field := range fields {
		if len(field) > maxInitiativeFieldBytes {
			return fmt.Errorf("initiative_result_invalid: deviation %q exceeds the field size limit", deviation.ObligationID)
		}
	}
	return nil
}

func (r Result) validateShape(advice Advice) error {
	if !r.correlatesWith(advice) {
		return fmt.Errorf("initiative_result_invalid: result does not correlate with advice")
	}
	if !r.GateIndependent {
		return fmt.Errorf("initiative_result_invalid: gate_independent must be true")
	}
	if len(r.Checks) == 0 {
		return fmt.Errorf("initiative_result_invalid: checks are required")
	}
	return nil
}

func (r Result) validateRevision() error {
	if r.ResultID != "" && r.Sequence < 1 {
		return fmt.Errorf("initiative_result_invalid: result sequence must be positive")
	}
	if r.Supersedes != "" && r.ResultID == "" {
		return fmt.Errorf("initiative_result_invalid: superseding result requires result_id")
	}
	return nil
}

func (r Result) correlatesWith(advice Advice) bool {
	return r.AdviceID != "" && r.AdviceID == advice.AdviceID &&
		r.MissionID == advice.MissionID &&
		normalizeRole(r.Role) == advice.Role &&
		r.RunID == advice.RunID
}

func validateOutcomes(outcomes []OutcomeCorrelation) error {
	for _, outcome := range outcomes {
		if strings.TrimSpace(outcome.ID) == "" || strings.TrimSpace(outcome.Status) == "" {
			return fmt.Errorf("initiative_result_invalid: outcome id and status are required")
		}
		if len(outcome.ID) > maxInitiativeFieldBytes || len(outcome.Status) > maxInitiativeFieldBytes {
			return fmt.Errorf("initiative_result_invalid: outcome %q exceeds the field size limit", outcome.ID)
		}
	}
	return nil
}

// AssessResult validates result against advice and derives a
// ResultAssessment describing whether the result should be challenged.
func AssessResult(advice Advice, result Result) (ResultAssessment, error) {
	return AssessConfidence(advice, result)
}
