package telemetry

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ConfidenceProducerAdapter is the single write boundary shared by all
// confidence-producing roles. Agent-specific wrappers keep provenance
// explicit while the persistence and validation rules remain centralized.
type ConfidenceProducerAdapter struct {
	Path      string
	Agent     string
	MissionID string
}

// NewConfidenceProducerAdapter creates the shared persistence boundary for a
// supported confidence-producing agent.
func NewConfidenceProducerAdapter(path, agent, missionID string) (ConfidenceProducerAdapter, error) {
	if !isConfidenceAgent(agent) {
		return ConfidenceProducerAdapter{}, fmt.Errorf("confidence producer: unsupported agent %q", agent)
	}
	if missionID == "" {
		return ConfidenceProducerAdapter{}, fmt.Errorf("confidence producer: mission_id is required")
	}
	return ConfidenceProducerAdapter{Path: path, Agent: agent, MissionID: missionID}, nil
}

// RecordClaim validates and persists one claim observation.
func (p ConfidenceProducerAdapter) RecordClaim(claim domain.ConfidenceClaim, evidence []domain.Evidence) (ConfidenceRecord, error) {
	return AppendConfidenceObservation(p.Path, p.Agent, p.MissionID, time.Now().UTC().Format(time.RFC3339Nano), claim, evidence)
}

// RecordMissing persists an explicit missing-producer observation.
func (p ConfidenceProducerAdapter) RecordMissing(correlationKey, reason string) error {
	return AppendMissingConfidenceRecord(p.Path, p.Agent, p.MissionID, correlationKey, reason, time.Now().UTC().Format(time.RFC3339Nano))
}

func isConfidenceAgent(agent string) bool {
	switch agent {
	case ConfidenceAgentScout, ConfidenceAgentRanger, ConfidenceAgentArchivist,
		ConfidenceAgentCritic, ConfidenceAgentMissionQuality, ConfidenceAgentHandoffChallenge,
		ConfidenceAgentSniper:
		return true
	default:
		return false
	}
}
