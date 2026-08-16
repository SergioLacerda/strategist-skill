package eval

import (
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// The scenario runners in this file share a common shape: call a
// domain.Validate*/Evaluate* function, reduce its result to an
// allowed/blocked status plus reason, and hand both to
// checkStatusAndReason. runStateMachineScenario (harness.go) doesn't fit
// this shape — it compares a final state, not a status/reason pair — so it
// stays in harness.go alongside the RunScenario dispatcher.

func runRouteDecisionScenario(s Scenario, res *ScenarioResult) {
	p := s.Input.Params
	route := paramString(p, "route")
	metadata := domain.RouteRequestMetadata{
		HasContext:           paramBool(p, "has_context"),
		TouchesSourceCode:    paramBool(p, "touches_source_code"),
		AnalysisArtifactOnly: paramBool(p, "analysis_artifact_only"),
		RequiresDiscovery:    paramBool(p, "requires_discovery"),
	}

	decision := domain.ValidateRouteDecision(route, metadata)
	checkStatusAndReason(res, string(decision.Status), decision.Reason, s.Expected)
}

func runSlotWriteScopeScenario(s Scenario, res *ScenarioResult) {
	p := s.Input.Params
	scope := domain.SlotWriteScope{
		SlotName:      paramString(p, "slot_name"),
		AllowedPrefix: paramString(p, "allowed_prefix"),
		AllowedExt:    paramString(p, "allowed_ext"),
	}
	attemptedPath := paramString(p, "attempted_path")

	err := domain.ValidateSlotWrite(scope, attemptedPath)
	actualStatus, actualReason := "allowed", ""
	if err != nil {
		actualStatus, actualReason = "blocked", err.Error()
	}
	checkStatusAndReason(res, actualStatus, actualReason, s.Expected)
}

func runChestGradeScenario(s Scenario, res *ScenarioResult) {
	p := s.Input.Params
	chestID := paramString(p, "chest_id")
	grade := domain.ChestGrade{
		SourceGrade:          paramString(p, "source_grade"),
		ReuseValue:           paramString(p, "reuse_value"),
		ImplementationStatus: paramString(p, "implementation_status"),
	}

	err := domain.ValidateChestGrade(chestID, grade)
	actualStatus, actualReason := "allowed", ""
	if err != nil {
		actualStatus, actualReason = "blocked", err.Error()
	}
	checkStatusAndReason(res, actualStatus, actualReason, s.Expected)
}

func runJewelTrustScenario(s Scenario, res *ScenarioResult) {
	p := s.Input.Params
	jewelID := paramString(p, "jewel_id")
	jewelTrust := paramString(p, "jewel_trust")
	chestTrustTier := paramString(p, "chest_trust_tier")

	err := domain.ValidateJewelTrust(jewelID, jewelTrust, chestTrustTier)
	actualStatus, actualReason := "allowed", ""
	if err != nil {
		actualStatus, actualReason = "blocked", err.Error()
	}
	checkStatusAndReason(res, actualStatus, actualReason, s.Expected)
}

func runCriticalHitTriggerScenario(s Scenario, res *ScenarioResult) {
	p := s.Input.Params
	evidence := domain.CriticalHitEvidence{
		Mode:                           domain.CriticalHitMode(paramString(p, "mode")),
		TaskType:                       paramString(p, "task_type"),
		SourcePath:                     paramString(p, "source_path"),
		TargetPath:                     paramString(p, "target_path"),
		BasePath:                       paramString(p, "base_path"),
		FileTypes:                      paramStringSlice(p, "file_types"),
		RiskLevel:                      paramString(p, "risk_level"),
		FileCount:                      paramInt(p, "file_count"),
		ExplicitCompletionClaim:        paramBool(p, "explicit_completion_claim"),
		EvidenceSummaryPresent:         paramBool(p, "evidence_summary_present"),
		CompletionInferredFromCodeOnly: paramBool(p, "completion_inferred_from_code_only"),
		PartialImplementationWithDeclaredResiduals: paramBool(p, "partial_implementation_with_declared_residuals"),
	}

	decision := domain.EvaluateCriticalHit(evidence)
	actualStatus := "blocked"
	if decision.Allowed {
		actualStatus = "allowed"
	}
	checkStatusAndReason(res, actualStatus, decision.Reason, s.Expected)
}

func checkStatusAndReason(res *ScenarioResult, actualStatus, actualReason string, expected Expected) {
	if expected.Status != "" && actualStatus != expected.Status {
		res.Violations = append(res.Violations, Violation{
			AssertionType: AssertEqualState,
			Message:       "status mismatch",
			Expected:      expected.Status,
			Actual:        actualStatus,
		})
	}
	if expected.Reason != "" && !strings.Contains(actualReason, expected.Reason) {
		res.Violations = append(res.Violations, Violation{
			AssertionType: AssertEqualState,
			Message:       "reason does not contain expected substring",
			Expected:      expected.Reason,
			Actual:        actualReason,
		})
	}
}
