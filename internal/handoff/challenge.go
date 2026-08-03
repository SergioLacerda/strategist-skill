// Package handoff validates Strategist handoff challenge contracts.
package handoff

import (
	"errors"
	"fmt"
)

const (
	// TransitionArchivistToSniper identifies the MVP handoff transition.
	TransitionArchivistToSniper = "archivist_to_sniper"
	// TransitionRangerToArchivist identifies the Ranger->Archivist handoff
	// transition — a structurally different handoff (known_facts/
	// uncertainties/evaluation_verdict, no execution.approval_required
	// concept), so it gets its own challenge-type vocabulary below rather
	// than reusing the MVP's objective/gate types. See
	// .analysis/refined/20260803-handoff-challenge-extensions/design.md §
	// Item 1.
	TransitionRangerToArchivist = "ranger_to_archivist"

	// ChallengeObjective checks whether Sniper preserved the handoff objective.
	// Valid only for TransitionArchivistToSniper.
	ChallengeObjective = "objective"
	// ChallengeBoundary checks scope-boundary preservation. Valid for both
	// transitions, with a different referent each time: for
	// TransitionArchivistToSniper it checks exclusions; for
	// TransitionRangerToArchivist it checks affected_scope vs. side_quests.
	ChallengeBoundary = "boundary"
	// ChallengeClassification checks classification preservation. Valid for
	// both transitions, with a different referent each time: for
	// TransitionArchivistToSniper it checks Evidence.Class; for
	// TransitionRangerToArchivist it checks known_facts vs. uncertainties.
	ChallengeClassification = "classification"
	// ChallengeGate checks whether Sniper preserved Approval Gate state.
	// Valid only for TransitionArchivistToSniper.
	ChallengeGate = "gate"
	// ChallengeRecall checks whether Archivist can restate the critical
	// known_facts entries of a Ranger handoff, by id. Valid only for
	// TransitionRangerToArchivist.
	ChallengeRecall = "recall"
	// ChallengeVerdict checks whether Archivist correctly restates a Ranger
	// handoff's evaluation_verdict — only applicable when
	// discovery_subtype: evaluation. Valid only for
	// TransitionRangerToArchivist.
	ChallengeVerdict = "verdict"

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

// allowedChallengeTypesByTransition scopes valid challenge types per
// transition — the MVP's four types stay valid only for
// TransitionArchivistToSniper, and TransitionRangerToArchivist has its own
// four, so a challenge type with no referent on a given transition (e.g.
// "gate" on a Ranger handoff, which never has an approval gate) is rejected
// by ValidatePolicy instead of silently accepted.
var allowedChallengeTypesByTransition = map[string]map[string]bool{
	TransitionArchivistToSniper: {
		ChallengeObjective:      true,
		ChallengeBoundary:       true,
		ChallengeClassification: true,
		ChallengeGate:           true,
	},
	TransitionRangerToArchivist: {
		ChallengeRecall:         true,
		ChallengeBoundary:       true,
		ChallengeClassification: true,
		ChallengeVerdict:        true,
	},
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
	errs = append(errs, validateChallengeTypes(p.Transition, p.RequiredTypes)...)
	errs = append(errs, validateMaxAttempts(p.MaxAttempts)...)
	errs = append(errs, validateOnFailure(p.OnFailure)...)
	return errors.Join(errs...)
}

func validateTransition(transition string) []error {
	if transition == "" {
		return []error{errors.New("handoff_policy_invalid: transition is required")}
	}
	if _, ok := allowedChallengeTypesByTransition[transition]; !ok {
		return []error{fmt.Errorf("handoff_policy_invalid: transition %q is not supported", transition)}
	}
	return nil
}

func validateRequiredTypesPresence(p Policy) []error {
	if p.Enabled && len(p.RequiredTypes) == 0 {
		return []error{errors.New("handoff_policy_invalid: required_types is required when enabled")}
	}
	return nil
}

func validateChallengeTypes(transition string, types []string) []error {
	allowed := allowedChallengeTypesByTransition[transition]
	var errs []error
	for _, typ := range types {
		if !allowed[typ] {
			errs = append(errs, fmt.Errorf("handoff_policy_invalid: challenge type %q is not allowed for transition %q", typ, transition))
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
