package domain

import (
	"fmt"
)

// MissionQualityCheck names one predicate in the mission_quality result
// contract proposed by critique_skill.txt item 1. These checks evaluate the
// Decision/Evidence model instead of free-form prose.
type MissionQualityCheck string

// Mission quality checks, matching critique_skill.txt's mission_quality keys.
const (
	CheckUnsupportedClaims            MissionQualityCheck = "unsupported_claims"
	CheckFactInferenceSeparation      MissionQualityCheck = "fact_inference_separation"
	CheckTraceableFindings            MissionQualityCheck = "traceable_findings"
	CheckAcceptanceCriteria           MissionQualityCheck = "acceptance_criteria"
	CheckUnresolvedQuestionsPreserved MissionQualityCheck = "unresolved_questions_preserved"
	CheckSourceScopeRespected         MissionQualityCheck = "source_scope_respected"
)

// MissionQualityCheckResult is one check's outcome.
type MissionQualityCheckResult struct {
	Check      MissionQualityCheck
	Applicable bool
	Passed     bool
	Violations []string
}

// MissionQualityInput is everything EvaluateMissionQuality needs.
type MissionQualityInput struct {
	Decisions []Decision
	Evidence  []Evidence

	// AcceptanceCriteria, when non-empty, satisfies CheckAcceptanceCriteria.
	AcceptanceCriteria []string

	// PreviouslyOpenDecisionIDs, when non-nil, enables
	// CheckUnresolvedQuestionsPreserved.
	PreviouslyOpenDecisionIDs []string

	// ApprovedScopePrefixes, when non-nil, enables CheckSourceScopeRespected.
	ApprovedScopePrefixes []string
}

// MissionQualityResult is the full evaluation outcome.
type MissionQualityResult struct {
	Checks []MissionQualityCheckResult
}

// Passed reports whether every applicable check in the result passed.
func (r MissionQualityResult) Passed() bool {
	for _, c := range r.Checks {
		if c.Applicable && !c.Passed {
			return false
		}
	}
	return true
}

// EvaluateMissionQuality runs every mission_quality predicate against in.
func EvaluateMissionQuality(in MissionQualityInput) MissionQualityResult {
	evidenceByID := indexEvidenceByID(in.Evidence)
	return MissionQualityResult{
		Checks: []MissionQualityCheckResult{
			checkUnsupportedClaims(in.Decisions, in.Evidence),
			checkFactInferenceSeparation(in.Evidence),
			checkTraceableFindings(in.Decisions, evidenceByID),
			checkAcceptanceCriteria(in.AcceptanceCriteria),
			checkUnresolvedQuestionsPreserved(in.Decisions, in.PreviouslyOpenDecisionIDs),
			checkSourceScopeRespected(in.Evidence, in.ApprovedScopePrefixes),
		},
	}
}

func indexEvidenceByID(evidence []Evidence) map[string]Evidence {
	byID := make(map[string]Evidence, len(evidence))
	for _, e := range evidence {
		byID[e.ID] = e
	}
	return byID
}

// checkUnsupportedClaims fails for every Decision with no cited evidence.
func checkUnsupportedClaims(decisions []Decision, evidence []Evidence) MissionQualityCheckResult {
	var violations []string
	for _, d := range decisions {
		violations = append(violations, unsupportedClaimViolations(d, evidence)...)
	}
	return MissionQualityCheckResult{
		Check:      CheckUnsupportedClaims,
		Applicable: true,
		Passed:     len(violations) == 0,
		Violations: violations,
	}
}

func unsupportedClaimViolations(decision Decision, evidence []Evidence) []string {
	var violations []string
	if len(decision.EvidenceIDs) == 0 {
		violations = append(violations, fmt.Sprintf("decision %s cites no evidence", decision.ID))
	}
	if decision.ClaimKind != "" || decision.ConfidencePercent != nil {
		if err := ValidateDecisionWithEvidence(decision, evidence); err != nil {
			violations = append(violations, fmt.Sprintf("decision %s confidence validation: %v", decision.ID, err))
		}
	}
	return violations
}

// checkFactInferenceSeparation fails for invalid evidence classes.
func checkFactInferenceSeparation(evidence []Evidence) MissionQualityCheckResult {
	var violations []string
	for _, e := range evidence {
		if e.Class == "" || !hasString(allowedEvidenceClasses, e.Class) {
			violations = append(violations, fmt.Sprintf("evidence %s has no valid class (got %q)", e.ID, e.Class))
		}
		if err := ValidateEvidence(e); err != nil {
			violations = append(violations, fmt.Sprintf("evidence %s validation: %v", e.ID, err))
		}
	}
	return MissionQualityCheckResult{
		Check:      CheckFactInferenceSeparation,
		Applicable: true,
		Passed:     len(violations) == 0,
		Violations: violations,
	}
}

// checkTraceableFindings fails for unresolved evidence references.
func checkTraceableFindings(decisions []Decision, evidenceByID map[string]Evidence) MissionQualityCheckResult {
	var violations []string
	for _, d := range decisions {
		for _, id := range d.EvidenceIDs {
			if _, ok := evidenceByID[id]; !ok {
				violations = append(violations, fmt.Sprintf("decision %s cites unresolved evidence id %s", d.ID, id))
			}
		}
	}
	return MissionQualityCheckResult{
		Check:      CheckTraceableFindings,
		Applicable: true,
		Passed:     len(violations) == 0,
		Violations: violations,
	}
}

func checkAcceptanceCriteria(criteria []string) MissionQualityCheckResult {
	passed := len(criteria) > 0
	var violations []string
	if !passed {
		violations = append(violations, "no acceptance criteria supplied")
	}
	return MissionQualityCheckResult{
		Check:      CheckAcceptanceCriteria,
		Applicable: true,
		Passed:     passed,
		Violations: violations,
	}
}

// checkUnresolvedQuestionsPreserved requires all prior open IDs to remain.
func checkUnresolvedQuestionsPreserved(decisions []Decision, previouslyOpenIDs []string) MissionQualityCheckResult {
	if previouslyOpenIDs == nil {
		return MissionQualityCheckResult{Check: CheckUnresolvedQuestionsPreserved, Applicable: false}
	}
	present := make(map[string]struct{}, len(decisions))
	for _, d := range decisions {
		present[d.ID] = struct{}{}
	}
	var violations []string
	for _, id := range previouslyOpenIDs {
		if _, ok := present[id]; !ok {
			violations = append(violations, fmt.Sprintf("previously open decision %s is missing from the current set", id))
		}
	}
	return MissionQualityCheckResult{
		Check:      CheckUnresolvedQuestionsPreserved,
		Applicable: true,
		Passed:     len(violations) == 0,
		Violations: violations,
	}
}
