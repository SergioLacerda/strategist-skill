// Package handoff validates Strategist handoff challenge contracts.
package handoff

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
	// TransitionSniperToValidation identifies the third handoff point
	// proposed by quiz.txt § "Executor -> validação/reconciliação": after
	// Sniper materializes documentation, a validation/reconciliation pass
	// confirms it understood which files were meant to change and which
	// deviations were authorized. Strategist's pipeline has no dedicated
	// "Validator" role today — this transition's challenges are consumed by
	// whichever process reviews a Sniper completion report (human,
	// follow-up mission, or a future dedicated role), the same way the
	// completion report itself already exists without a formal consumer
	// role. See .analysis/refined/20260805-quiz-handoff-challenge-evaluation/.
	TransitionSniperToValidation = "sniper_to_validation"

	// ChallengeObjective checks whether Sniper preserved the handoff objective.
	// Valid only for TransitionArchivistToSniper.
	ChallengeObjective = "objective"
	// ChallengeBoundary checks scope-boundary preservation. Valid for all
	// three transitions, with a different referent each time: for
	// TransitionArchivistToSniper it checks exclusions; for
	// TransitionRangerToArchivist it checks affected_scope vs. side_quests;
	// for TransitionSniperToValidation it checks which files were declared
	// in scope (materialized) vs. explicitly out of scope.
	ChallengeBoundary = "boundary"
	// ChallengeClassification checks classification preservation. Valid for
	// all three transitions, with a different referent each time: for
	// TransitionArchivistToSniper it checks Evidence.Class; for
	// TransitionRangerToArchivist it checks known_facts vs. uncertainties;
	// for TransitionSniperToValidation it checks authorized deviations vs.
	// unauthorized ones.
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
	// ChallengeCounterfactual tests rule *application* rather than recall:
	// the respondent is given a short scenario and must say whether a named
	// constraint permits it, per quiz.txt's own worked example (a test
	// failing because the production API is hard to simulate — does the
	// constraint allow changing the API just to ease testing?). quiz.txt
	// calls this its most valuable type, since parroting a constraint's
	// text back does not prove the respondent can apply it. Valid for
	// TransitionArchivistToSniper and TransitionSniperToValidation, where a
	// concrete constraint/invariant exists to apply; not valid for
	// TransitionRangerToArchivist, which has no constraints to apply yet.
	ChallengeCounterfactual = "counterfactual"

	// DecisionApproved is the expected classification for approved decisions.
	DecisionApproved = "approved_decision"
	// QuestionUnresolved is the expected classification for open questions.
	QuestionUnresolved = "unresolved_question"

	// ForbiddenClaimExecutionAuthorized is the reserved ForbiddenClaims value
	// meaning the acknowledgment must not assert execution is authorized
	// (Acknowledgment.GateAllowed must not be true).
	ForbiddenClaimExecutionAuthorized = "execution_authorized"
	// forbiddenClaimAsApprovedSuffix marks a ForbiddenClaims entry of the
	// form "<ref>_as_approved" — the acknowledgment must not classify ref as
	// DecisionApproved. Matches quiz.txt's own literal example
	// ("Q-01_as_approved").
	forbiddenClaimAsApprovedSuffix = "_as_approved"

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

// Challenge is one Archivist-authored verification question for Sniper.
type Challenge struct {
	ID                     string
	Type                   string
	SourceRefs             []string
	Critical               bool
	ExpectedClassification map[string]string
	ExpectedGateAllowed    *bool
	// ExpectedCounterfactual is the correct yes/no answer for a
	// type=counterfactual challenge — e.g. "may C-01 be relaxed just to
	// make a test easier?" expects false. Only meaningful when
	// Type == ChallengeCounterfactual.
	ExpectedCounterfactual *bool
}

// Acknowledgment is Sniper's machine-checkable response to challenges.
type Acknowledgment struct {
	ChallengeRefs   []string
	UnderstoodRefs  []string
	Classifications map[string]string
	GateAllowed     *bool
	// CounterfactualAnswers maps a counterfactual Challenge's ID to the
	// answer given, for comparison against that Challenge's
	// ExpectedCounterfactual.
	CounterfactualAnswers map[string]bool
}

// Result summarizes deterministic handoff challenge verification.
type Result struct {
	Status                   string
	Passed                   bool
	MissingRefs              []string
	MissingChallenges        []string
	MisclassifiedRefs        []string
	GateMismatch             bool
	CounterfactualMismatches []string
	ForbiddenClaimViolations []string
	CriticalFailures         int
	NextAction               string
}
