package mission

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// RecordRouteDecision persists Scout's route_decision for a mission so the
// execution boundary can read it back. The decision must belong to missionID;
// an absent timestamp is filled in. It reports false when a decision for the
// mission was already recorded (the history is idempotent by mission_id).
func RecordRouteDecision(strategistRoot, missionID string, raw []byte) (bool, error) {
	return recordRouteDecision(context.Background(), strategistRoot, missionID, raw, nil)
}

// RecordRouteDecisionWithTelemetry persists a route decision and emits the
// canonical Stage event when a new decision is appended. Replays remain
// idempotent for both persistence and telemetry.
func RecordRouteDecisionWithTelemetry(ctx context.Context, strategistRoot, missionID string, raw []byte, sink telemetry.EventSink) (bool, error) {
	return recordRouteDecision(ctx, strategistRoot, missionID, raw, sink)
}

func recordRouteDecision(ctx context.Context, strategistRoot, missionID string, raw []byte, sink telemetry.EventSink) (bool, error) {
	decision, err := parseRouteDecision(missionID, raw)
	if err != nil {
		return false, err
	}
	appended, decision, err := appendRouteDecision(strategistRoot, decision)
	if err != nil {
		return false, err
	}
	if err := recordScoutRouteConfidence(strategistRoot, decision); err != nil {
		return appended, fmt.Errorf("record Scout route confidence: %w", err)
	}
	if err := emitStageResolutionTelemetry(ctx, sink, appended, decision); err != nil {
		return appended, err
	}
	return appended, nil
}

func emitStageResolutionTelemetry(ctx context.Context, sink telemetry.EventSink, appended bool, decision telemetry.RouteDecision) error {
	if !appended || sink == nil {
		return nil
	}
	artifact, err := stageResolutionArtifact(decision)
	if err != nil {
		return fmt.Errorf("build Stage telemetry artifact: %w", err)
	}
	if err := sink.Emit(ctx, telemetry.NewStageResolutionEvent(decision.MissionID, artifact)); err != nil {
		return fmt.Errorf("emit Stage resolution telemetry: %w", err)
	}
	return nil
}

func stageResolutionArtifact(decision telemetry.RouteDecision) (domain.StageResolutionArtifact, error) {
	artifact, err := domain.NewStageResolutionArtifact(domain.StageResolution{
		SchemaVersion:  domain.StageResolutionArtifactSchemaVersion,
		Stage:          domain.Stage(decision.Stage),
		LegacyRoute:    decision.SelectedRoute,
		Role:           decision.StageRole,
		Feat:           decision.StageFeat,
		MissionID:      decision.MissionID,
		CorrelationKey: decision.StageCorrelationID,
		GateRequired:   decision.StageGateRequired,
		PolicyVersion:  decision.StagePolicyVersion,
		Reason:         decision.StageReason,
	}, decision.StageTrigger)
	if err != nil {
		return domain.StageResolutionArtifact{}, fmt.Errorf("build Stage resolution artifact: %w", err)
	}
	return artifact, nil
}
