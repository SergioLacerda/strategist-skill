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
	Challenge         bool
	ConfidenceCeiling string
	Reasons           []string
}

// ValidateAgainst checks that the result correlates with and satisfies the
// structural requirements of the given advice.
func (r Result) ValidateAgainst(advice Advice) error {
	if !r.correlatesWith(advice) {
		return fmt.Errorf("initiative_result_invalid: result does not correlate with advice")
	}
	if !r.GateIndependent {
		return fmt.Errorf("initiative_result_invalid: gate_independent must be true")
	}
	if len(r.Checks) == 0 {
		return fmt.Errorf("initiative_result_invalid: checks are required")
	}
	if err := validateChecks(r.Checks); err != nil {
		return err
	}
	return validateOutcomes(r.Outcomes)
}

func (r Result) correlatesWith(advice Advice) bool {
	return r.AdviceID != "" && r.AdviceID == advice.AdviceID &&
		r.MissionID == advice.MissionID &&
		normalizeRole(r.Role) == advice.Role &&
		r.RunID == advice.RunID
}

func validateChecks(checks []ObligationCheck) error {
	for _, check := range checks {
		if err := validateCheck(check); err != nil {
			return err
		}
	}
	return nil
}

func validateCheck(check ObligationCheck) error {
	if strings.TrimSpace(check.ID) == "" || !validCheckStatus(check.Status) {
		return fmt.Errorf("initiative_result_invalid: check id and valid status are required")
	}
	if check.Status == CheckSatisfied && len(check.EvidenceRefs) == 0 {
		return fmt.Errorf("initiative_result_invalid: satisfied check %q requires evidence", check.ID)
	}
	return nil
}

func validateOutcomes(outcomes []OutcomeCorrelation) error {
	for _, outcome := range outcomes {
		if strings.TrimSpace(outcome.ID) == "" || strings.TrimSpace(outcome.Status) == "" {
			return fmt.Errorf("initiative_result_invalid: outcome id and status are required")
		}
	}
	return nil
}

// AssessResult validates result against advice and derives a
// ResultAssessment describing whether the result should be challenged.
func AssessResult(advice Advice, result Result) (ResultAssessment, error) {
	if err := result.ValidateAgainst(advice); err != nil {
		return ResultAssessment{}, err
	}
	assessment := ResultAssessment{ConfidenceCeiling: advice.Diligence.ConfidenceCeiling}
	for _, check := range result.Checks {
		switch check.Status {
		case CheckBlocked:
			assessment.Challenge = true
			assessment.Reasons = append(assessment.Reasons, "blocked_obligation:"+check.ID)
		case CheckPartial:
			assessment.Challenge = true
			assessment.Reasons = append(assessment.Reasons, "partial_obligation:"+check.ID)
		case CheckSatisfied, CheckNotApplicable:
			// no challenge needed
		}
	}
	if len(result.EvidenceRefs) == 0 {
		assessment.Challenge = true
		assessment.ConfidenceCeiling = "low"
		assessment.Reasons = append(assessment.Reasons, "missing_result_evidence")
	}
	return assessment, nil
}

func validCheckStatus(status CheckStatus) bool {
	switch status {
	case CheckSatisfied, CheckPartial, CheckBlocked, CheckNotApplicable:
		return true
	default:
		return false
	}
}
