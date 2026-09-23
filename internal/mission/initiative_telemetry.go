package mission

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/initiative"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// ConsumeHandoff validates advisory metadata at the next role boundary and
// emits an observation for that consumer. It never recalculates the source
// recommendation and never changes Gate or LEVELING state.
func (r InitiativeRuntime) ConsumeHandoff(handoff InitiativeHandoff) error {
	if err := handoff.Validate(); err != nil {
		return err
	}
	event := telemetry.NewEvent("strategist.initiative.handoff_consumed", telemetry.SeverityDebug, handoff.Result.RunID, true)
	event.Attributes = map[string]any{
		telemetry.AttrComponent:                   "initiative",
		telemetry.AttrAbility:                     initiative.AbilityName,
		telemetry.AttrMissionID:                   handoff.Result.MissionID,
		telemetry.AttrRole:                        handoff.ToRole,
		telemetry.AttrRoleRun:                     handoff.Result.RunID,
		telemetry.AttrInitiativeAdviceID:          handoff.Result.AdviceID,
		telemetry.AttrInitiativePolicyVersion:     handoff.Advice.PolicyVersion,
		telemetry.AttrInitiativePolicyDigest:      handoff.Advice.PolicyDigest,
		telemetry.AttrInitiativeResultStatus:      resultStatus(handoff.Assessment),
		telemetry.AttrInitiativeConfidenceCeiling: handoff.Assessment.ConfidenceCeiling,
		telemetry.AttrInitiativeSourceRole:        handoff.FromRole,
	}
	if err := r.eventSink.Emit(context.Background(), event); err != nil {
		return fmt.Errorf("initiative telemetry: emit handoff consumed: %w", err)
	}
	return nil
}

func (r InitiativeRuntime) emitAdvice(advice initiative.Advice, reused bool) error {
	event := telemetry.NewEvent("strategist.initiative.advice", telemetry.SeverityDebug, advice.RunID, true)
	event.Attributes = map[string]any{
		telemetry.AttrComponent:                       "initiative",
		telemetry.AttrAbility:                         initiative.AbilityName,
		telemetry.AttrMissionID:                       advice.MissionID,
		telemetry.AttrRole:                            advice.Role,
		telemetry.AttrRoleRun:                         advice.RunID,
		telemetry.AttrInitiativeAdviceID:              advice.AdviceID,
		telemetry.AttrInitiativePolicyVersion:         advice.PolicyVersion,
		telemetry.AttrInitiativePolicyDigest:          advice.PolicyDigest,
		telemetry.AttrInitiativeTrigger:               string(advice.Trigger),
		telemetry.AttrInitiativeAlignment:             string(advice.Alignment),
		telemetry.AttrInitiativeConfidenceCeiling:     advice.Diligence.ConfidenceCeiling,
		telemetry.AttrInitiativeObservedModel:         advice.Observed.Model,
		telemetry.AttrInitiativeObservedProvider:      advice.Observed.Provider,
		telemetry.AttrInitiativeObservedEffort:        string(advice.Observed.Effort),
		telemetry.AttrInitiativeObservedLevelSource:   advice.Observed.LevelSource,
		telemetry.AttrInitiativeRecommendedCapability: advice.Recommendation.RecommendedCapability,
		telemetry.AttrInitiativeRecommendedEffort:     string(advice.Recommendation.RecommendedEffort),
		telemetry.AttrInitiativeAdviceReused:          reused,
	}
	if advice.Supersedes != "" {
		event.Attributes[telemetry.AttrInitiativeSupersedes] = advice.Supersedes
	}
	if err := r.eventSink.Emit(context.Background(), event); err != nil {
		return fmt.Errorf("initiative telemetry: emit advice: %w", err)
	}
	return nil
}

func (r InitiativeRuntime) emitResult(advice initiative.Advice, result initiative.Result, assessment initiative.ResultAssessment) error {
	event := telemetry.NewEvent("strategist.initiative.result", telemetry.SeverityDebug, result.RunID, true)
	event.Attributes = map[string]any{
		telemetry.AttrComponent:                   "initiative",
		telemetry.AttrAbility:                     initiative.AbilityName,
		telemetry.AttrMissionID:                   result.MissionID,
		telemetry.AttrRole:                        result.Role,
		telemetry.AttrRoleRun:                     result.RunID,
		telemetry.AttrInitiativeAdviceID:          result.AdviceID,
		telemetry.AttrInitiativePolicyVersion:     advice.PolicyVersion,
		telemetry.AttrInitiativePolicyDigest:      advice.PolicyDigest,
		telemetry.AttrInitiativeResultStatus:      resultStatus(assessment),
		telemetry.AttrInitiativeConfidenceCeiling: assessment.ConfidenceCeiling,
		telemetry.AttrInitiativeEvidenceRefs:      evidenceIDs(result.EvidenceRefs),
		telemetry.AttrInitiativeOutcomeIDs:        outcomeIDs(result.Outcomes),
		telemetry.AttrInitiativeDeviationIDs:      deviationIDs(result.Deviations),
		telemetry.AttrInitiativeChallengeReasons:  append([]string(nil), assessment.Reasons...),
	}
	if err := r.eventSink.Emit(context.Background(), event); err != nil {
		return fmt.Errorf("initiative telemetry: emit result: %w", err)
	}
	return nil
}

func resultStatus(assessment initiative.ResultAssessment) string {
	if assessment.Challenge {
		return "challenged"
	}
	return "accepted"
}

func evidenceIDs(refs []initiative.EvidenceRef) []string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.ID)
	}
	return ids
}

func outcomeIDs(outcomes []initiative.OutcomeCorrelation) []string {
	ids := make([]string, 0, len(outcomes))
	for _, outcome := range outcomes {
		ids = append(ids, outcome.ID)
	}
	return ids
}

func deviationIDs(deviations []initiative.Deviation) []string {
	ids := make([]string, 0, len(deviations))
	for _, deviation := range deviations {
		ids = append(ids, deviation.ObligationID)
	}
	return ids
}
