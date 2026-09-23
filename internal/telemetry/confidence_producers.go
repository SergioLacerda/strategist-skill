package telemetry

import (
	"fmt"
	"strings"
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
	Run       string
}

// WithRun scopes subsequently written observations to an explicit role run.
// Empty preserves the legacy mission-wide record behavior.
func (p ConfidenceProducerAdapter) WithRun(run string) ConfidenceProducerAdapter {
	p.Run = run
	return p
}

// NewConfidenceProducerAdapter creates the shared persistence boundary for a
// supported confidence-producing agent.
func NewConfidenceProducerAdapter(path, agent, missionID string) (ConfidenceProducerAdapter, error) {
	if !isConfidenceAgent(agent) {
		return ConfidenceProducerAdapter{}, fmt.Errorf("confidence producer: unsupported agent %q (want one of %s)", agent, strings.Join(ConfidenceAgents(), ", "))
	}
	if missionID == "" {
		return ConfidenceProducerAdapter{}, fmt.Errorf("confidence producer: mission_id is required")
	}
	return ConfidenceProducerAdapter{Path: path, Agent: agent, MissionID: missionID}, nil
}

// RecordClaim validates and persists one claim observation.
func (p ConfidenceProducerAdapter) RecordClaim(claim domain.ConfidenceClaim, evidence []domain.Evidence) (ConfidenceRecord, error) {
	return AppendConfidenceObservationForRun(p.Path, p.Agent, p.MissionID, p.Run, time.Now().UTC().Format(time.RFC3339Nano), claim, evidence)
}

// RecordMissing persists an explicit missing-producer observation.
func (p ConfidenceProducerAdapter) RecordMissing(correlationKey, reason string) error {
	return AppendMissingConfidenceRecordForRun(p.Path, p.Agent, p.MissionID, p.Run, correlationKey, reason, time.Now().UTC().Format(time.RFC3339Nano))
}

// ConfidenceAgents lists every accepted producing agent: the registered roles
// (exact ids, phase order) followed by the producers that are not roles — the
// critic, mission quality and handoff challenge. There are no aliases.
func ConfidenceAgents() []string {
	agents := domain.DefaultRoleRegistry().IDs()
	return append(agents, ConfidenceAgentCritic, ConfidenceAgentMissionQuality, ConfidenceAgentHandoffChallenge)
}

func isConfidenceAgent(agent string) bool {
	for _, id := range ConfidenceAgents() {
		if id == agent {
			return true
		}
	}
	return false
}
