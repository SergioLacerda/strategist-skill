package domain

import "testing"

func wellFormedInput() MissionQualityInput {
	return MissionQualityInput{
		Decisions: []Decision{
			{ID: "DEC-001", Statement: "s", Status: DecisionStatusApproved, EvidenceIDs: []string{"EVD-001"}, Confidence: ConfidenceHigh},
		},
		Evidence: []Evidence{
			{ID: "EVD-001", SourceRef: "docs/adr/0005-slot-write-contracts.md", Class: EvidenceClassExplicit, Confidence: ConfidenceHigh},
		},
		AcceptanceCriteria:        []string{"builds and tests pass"},
		PreviouslyOpenDecisionIDs: []string{},
		ApprovedScopePrefixes:     []string{"docs/"},
	}
}

func findCheck(t *testing.T, result MissionQualityResult, check MissionQualityCheck) MissionQualityCheckResult {
	t.Helper()
	for _, c := range result.Checks {
		if c.Check == check {
			return c
		}
	}
	t.Fatalf("check %q not found in result", check)
	return MissionQualityCheckResult{}
}

func TestEvaluateMissionQuality_AllPassOnWellFormedInput(t *testing.T) {
	t.Parallel()
	result := EvaluateMissionQuality(wellFormedInput())
	if !result.Passed() {
		t.Fatalf("expected Passed()=true, got result: %+v", result)
	}
	for _, c := range result.Checks {
		if !c.Applicable {
			t.Fatalf("expected all checks applicable when full input supplied, got %+v", c)
		}
		if !c.Passed {
			t.Fatalf("check %q unexpectedly failed: %v", c.Check, c.Violations)
		}
	}
}

func TestCheckUnsupportedClaims_FailsOnEmptyEvidence(t *testing.T) {
	t.Parallel()
	in := wellFormedInput()
	in.Decisions[0].EvidenceIDs = nil
	result := EvaluateMissionQuality(in)
	c := findCheck(t, result, CheckUnsupportedClaims)
	if c.Passed {
		t.Fatal("expected CheckUnsupportedClaims to fail for a decision with no evidence")
	}
	if result.Passed() {
		t.Fatal("expected overall Passed()=false")
	}
}

func TestCheckFactInferenceSeparation_FailsOnMissingClass(t *testing.T) {
	t.Parallel()
	in := wellFormedInput()
	in.Evidence[0].Class = ""
	result := EvaluateMissionQuality(in)
	c := findCheck(t, result, CheckFactInferenceSeparation)
	if c.Passed {
		t.Fatal("expected CheckFactInferenceSeparation to fail for evidence with empty class")
	}
}

func TestCheckTraceableFindings_FailsOnUnresolvedEvidenceID(t *testing.T) {
	t.Parallel()
	in := wellFormedInput()
	in.Decisions[0].EvidenceIDs = []string{"EVD-999"}
	result := EvaluateMissionQuality(in)
	c := findCheck(t, result, CheckTraceableFindings)
	if c.Passed {
		t.Fatal("expected CheckTraceableFindings to fail for an unresolved evidence id")
	}
	if len(c.Violations) != 1 {
		t.Fatalf("expected exactly one violation, got %v", c.Violations)
	}
}

func TestCheckAcceptanceCriteria_FailsWhenEmpty(t *testing.T) {
	t.Parallel()
	in := wellFormedInput()
	in.AcceptanceCriteria = nil
	result := EvaluateMissionQuality(in)
	c := findCheck(t, result, CheckAcceptanceCriteria)
	if c.Passed {
		t.Fatal("expected CheckAcceptanceCriteria to fail when empty")
	}
}

func TestCheckUnresolvedQuestionsPreserved_NotApplicableWhenNil(t *testing.T) {
	t.Parallel()
	in := wellFormedInput()
	in.PreviouslyOpenDecisionIDs = nil
	result := EvaluateMissionQuality(in)
	c := findCheck(t, result, CheckUnresolvedQuestionsPreserved)
	if c.Applicable {
		t.Fatal("expected CheckUnresolvedQuestionsPreserved to be not applicable when input is nil")
	}
	// A not-applicable check must not affect the overall verdict.
	if !result.Passed() {
		t.Fatalf("expected Passed()=true despite one not-applicable check, got %+v", result)
	}
}

func TestCheckUnresolvedQuestionsPreserved_FailsWhenDecisionDropped(t *testing.T) {
	t.Parallel()
	in := wellFormedInput()
	in.PreviouslyOpenDecisionIDs = []string{"DEC-999"}
	result := EvaluateMissionQuality(in)
	c := findCheck(t, result, CheckUnresolvedQuestionsPreserved)
	if c.Passed {
		t.Fatal("expected CheckUnresolvedQuestionsPreserved to fail when a previously open decision vanished")
	}
}

func TestCheckUnresolvedQuestionsPreserved_PassesWhenStatusChangedButPresent(t *testing.T) {
	t.Parallel()
	in := wellFormedInput()
	// DEC-001 was open before, is now approved — present, not dropped.
	in.PreviouslyOpenDecisionIDs = []string{"DEC-001"}
	result := EvaluateMissionQuality(in)
	c := findCheck(t, result, CheckUnresolvedQuestionsPreserved)
	if !c.Passed {
		t.Fatalf("expected pass — decision progressed, not dropped: %v", c.Violations)
	}
}

func TestCheckSourceScopeRespected_NotApplicableWhenNil(t *testing.T) {
	t.Parallel()
	in := wellFormedInput()
	in.ApprovedScopePrefixes = nil
	result := EvaluateMissionQuality(in)
	c := findCheck(t, result, CheckSourceScopeRespected)
	if c.Applicable {
		t.Fatal("expected CheckSourceScopeRespected to be not applicable when input is nil")
	}
}

func TestCheckSourceScopeRespected_FailsOutsideScope(t *testing.T) {
	t.Parallel()
	in := wellFormedInput()
	in.ApprovedScopePrefixes = []string{"internal/telemetry/"}
	result := EvaluateMissionQuality(in)
	c := findCheck(t, result, CheckSourceScopeRespected)
	if c.Passed {
		t.Fatal("expected CheckSourceScopeRespected to fail — evidence source_ref is under docs/, not internal/telemetry/")
	}
}

func TestMissionQualityResult_PassedIgnoresNotApplicableChecks(t *testing.T) {
	t.Parallel()
	in := wellFormedInput()
	in.PreviouslyOpenDecisionIDs = nil
	in.ApprovedScopePrefixes = nil
	result := EvaluateMissionQuality(in)
	if !result.Passed() {
		t.Fatalf("expected Passed()=true with two not-applicable checks and all applicable checks passing, got %+v", result)
	}
}

func TestEvaluateMissionQuality_EmptyInput(t *testing.T) {
	t.Parallel()
	result := EvaluateMissionQuality(MissionQualityInput{})
	// No decisions/evidence means no violations for the per-item checks,
	// but acceptance_criteria has nothing supplied, so overall must fail.
	if result.Passed() {
		t.Fatal("expected Passed()=false — acceptance_criteria has nothing supplied")
	}
	if !findCheck(t, result, CheckUnsupportedClaims).Passed {
		t.Fatal("expected CheckUnsupportedClaims to vacuously pass with no decisions")
	}
}
