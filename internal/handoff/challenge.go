// Package handoff validates Strategist handoff challenge contracts.
package handoff

import (
	"errors"
	"fmt"
)

const (
	// TransitionArchivistToSniper identifies the MVP handoff transition.
	TransitionArchivistToSniper = "archivist_to_sniper"

	// ChallengeObjective checks whether Sniper preserved the handoff objective.
	ChallengeObjective = "objective"
	// ChallengeBoundary checks whether Sniper preserved scope boundaries.
	ChallengeBoundary = "boundary"
	// ChallengeClassification checks whether Sniper preserved evidence classes.
	ChallengeClassification = "classification"
	// ChallengeGate checks whether Sniper preserved Approval Gate state.
	ChallengeGate = "gate"

	// DecisionApproved is the expected classification for approved decisions.
	DecisionApproved = "approved_decision"
	// QuestionUnresolved is the expected classification for open questions.
	QuestionUnresolved = "unresolved_question"

	// FailureActionReturnToArchivist asks Sniper to return failed handoffs.
	FailureActionReturnToArchivist = "return_to_archivist"

	// StatusRequired means risk policy requires a challenge.
	StatusRequired = "required"
	// StatusSkipped means risk policy does not require a challenge.
	StatusSkipped = "skipped"
	// StatusPassed means verification completed without failures.
	StatusPassed = "passed"
	// StatusFailed means verification found blocking failures.
	StatusFailed = "failed"
)

var allowedChallengeTypes = map[string]bool{
	ChallengeObjective:      true,
	ChallengeBoundary:       true,
	ChallengeClassification: true,
	ChallengeGate:           true,
}

// Policy decides whether a semantic handoff challenge is required for a transition.
// It is intentionally data-only so contracts, fixtures, and runtime code can share
// the same vocabulary without depending on provider invocation.
type Policy struct {
	Enabled            bool
	Transition         string
	RequiredTypes      []string
	RequireAllCritical bool
	MaxAttempts        int
	OnFailure          string
}

// RiskSignals describe handoff traits that make a challenge mandatory.
type RiskSignals struct {
	ApprovalGatePresent          bool
	MandatoryConstraintsPresent  bool
	UnresolvedQuestionsPresent   bool
	ForbiddenScopePresent        bool
	ImplementationHandoffPresent bool
	DestructiveOperationPossible bool
	SecuritySensitiveTask        bool
	InformationalOnly            bool
}

// Challenge is one Archivist-authored verification question for Sniper.
type Challenge struct {
	ID                     string
	Type                   string
	SourceRefs             []string
	Critical               bool
	ExpectedClassification map[string]string
	ExpectedGateAllowed    *bool
}

// Acknowledgment is Sniper's machine-checkable response to challenges.
type Acknowledgment struct {
	ChallengeRefs   []string
	UnderstoodRefs  []string
	Classifications map[string]string
	GateAllowed     *bool
}

// Result summarizes deterministic handoff challenge verification.
type Result struct {
	Status            string
	Passed            bool
	MissingRefs       []string
	MissingChallenges []string
	MisclassifiedRefs []string
	GateMismatch      bool
	CriticalFailures  int
	NextAction        string
}

// DefaultPolicy returns the Handoff Challenge MVP policy.
func DefaultPolicy() Policy {
	return Policy{
		Enabled:            true,
		Transition:         TransitionArchivistToSniper,
		RequiredTypes:      []string{ChallengeObjective, ChallengeBoundary, ChallengeClassification, ChallengeGate},
		RequireAllCritical: true,
		MaxAttempts:        2,
		OnFailure:          FailureActionReturnToArchivist,
	}
}

// ValidatePolicy verifies that a policy is usable by the MVP verifier.
func ValidatePolicy(p Policy) error {
	var errs []error
	errs = append(errs, validateTransition(p.Transition)...)
	errs = append(errs, validateRequiredTypesPresence(p)...)
	errs = append(errs, validateChallengeTypes(p.RequiredTypes)...)
	errs = append(errs, validateMaxAttempts(p.MaxAttempts)...)
	errs = append(errs, validateOnFailure(p.OnFailure)...)
	return errors.Join(errs...)
}

func validateTransition(transition string) []error {
	if transition == "" {
		return []error{errors.New("handoff_policy_invalid: transition is required")}
	}
	if transition != TransitionArchivistToSniper {
		return []error{fmt.Errorf("handoff_policy_invalid: transition %q is not supported by the MVP", transition)}
	}
	return nil
}

func validateRequiredTypesPresence(p Policy) []error {
	if p.Enabled && len(p.RequiredTypes) == 0 {
		return []error{errors.New("handoff_policy_invalid: required_types is required when enabled")}
	}
	return nil
}

func validateChallengeTypes(types []string) []error {
	var errs []error
	for _, typ := range types {
		if !allowedChallengeTypes[typ] {
			errs = append(errs, fmt.Errorf("handoff_policy_invalid: challenge type %q is not allowed", typ))
		}
	}
	return errs
}

func validateMaxAttempts(maxAttempts int) []error {
	if maxAttempts < 0 {
		return []error{errors.New("handoff_policy_invalid: max_attempts must be >= 0")}
	}
	return nil
}

func validateOnFailure(onFailure string) []error {
	if onFailure != "" && onFailure != FailureActionReturnToArchivist {
		return []error{fmt.Errorf("handoff_policy_invalid: on_failure %q is not supported by the MVP", onFailure)}
	}
	return nil
}

// RequiredByRisk reports whether risk signals require a challenge.
func RequiredByRisk(s RiskSignals) bool {
	if s.InformationalOnly && !s.ApprovalGatePresent && !s.MandatoryConstraintsPresent &&
		!s.UnresolvedQuestionsPresent && !s.ForbiddenScopePresent &&
		!s.ImplementationHandoffPresent && !s.DestructiveOperationPossible &&
		!s.SecuritySensitiveTask {
		return false
	}
	return s.ApprovalGatePresent ||
		s.MandatoryConstraintsPresent ||
		s.UnresolvedQuestionsPresent ||
		s.ForbiddenScopePresent ||
		s.ImplementationHandoffPresent ||
		s.DestructiveOperationPossible ||
		s.SecuritySensitiveTask
}

// StatusForRisk returns the policy status implied by risk signals.
func StatusForRisk(s RiskSignals) string {
	if RequiredByRisk(s) {
		return StatusRequired
	}
	return StatusSkipped
}
