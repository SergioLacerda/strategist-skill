package mission

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/initiative"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func (r InitiativeRuntime) emitAdvice(advice initiative.Advice, reused bool) error {
	event := telemetry.NewEvent("strategist.initiative.advice", telemetry.SeverityDebug, advice.RunID, true)
	event.Attributes = map[string]any{
		telemetry.AttrComponent:                       "initiative",
		telemetry.AttrFeat:                            initiative.FeatName,
		telemetry.AttrInitiativeFeat:                  initiative.FeatName,
		telemetry.AttrInitiativeFeatLabel:             initiative.FeatName,
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
