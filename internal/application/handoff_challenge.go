package application

import (
	"errors"
	"time"
)

// HandoffChallengeInput contains the application-level facts needed to
// record one Handoff Challenge verification attempt. It deliberately mirrors
// the persistence shape without depending on telemetry or internal/handoff.
type HandoffChallengeInput struct {
	MissionID         string
	Transition        string
	PolicyTransition  string
	Attempt           int
	Status            string
	Passed            bool
	MissingRefs       []string
	MissingChallenges []string
	MisclassifiedRefs []string
	GateMismatch      bool

	CounterfactualMismatches []string
	ForbiddenClaimViolations []string
	CriticalFailures         int
}

// HandoffChallengeRecord is the adapter-neutral record passed to the
// persistence port after application-level transition and timestamp rules are
// resolved.
type HandoffChallengeRecord struct {
	MissionID                string
	Transition               string
	Attempt                  int
	Timestamp                string
	Status                   string
	Passed                   bool
	MissingRefs              []string
	MissingChallenges        []string
	MisclassifiedRefs        []string
	GateMismatch             bool
	CounterfactualMismatches []string
	ForbiddenClaimViolations []string
	CriticalFailures         int
}

// HandoffChallengeRecorder is the persistence port for one challenge record.
type HandoffChallengeRecorder func(HandoffChallengeRecord) error

// RecordHandoffChallenge resolves application rules and delegates persistence
// to the caller-owned adapter.
func RecordHandoffChallenge(input HandoffChallengeInput, now time.Time, record HandoffChallengeRecorder) error {
	if record == nil {
		return errors.New("handoff challenge recorder is unavailable")
	}
	return record(BuildHandoffChallengeRecord(input, now))
}

// BuildHandoffChallengeRecord applies the policy-transition fallback and
// canonical UTC timestamp used by the handoff history contract.
func BuildHandoffChallengeRecord(input HandoffChallengeInput, now time.Time) HandoffChallengeRecord {
	transition := input.Transition
	if transition == "" {
		transition = input.PolicyTransition
	}
	return HandoffChallengeRecord{
		MissionID:                input.MissionID,
		Transition:               transition,
		Attempt:                  input.Attempt,
		Timestamp:                now.UTC().Format(time.RFC3339),
		Status:                   input.Status,
		Passed:                   input.Passed,
		MissingRefs:              input.MissingRefs,
		MissingChallenges:        input.MissingChallenges,
		MisclassifiedRefs:        input.MisclassifiedRefs,
		GateMismatch:             input.GateMismatch,
		CounterfactualMismatches: input.CounterfactualMismatches,
		ForbiddenClaimViolations: input.ForbiddenClaimViolations,
		CriticalFailures:         input.CriticalFailures,
	}
}
