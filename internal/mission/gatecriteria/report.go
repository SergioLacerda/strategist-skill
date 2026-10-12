// Package gatecriteria builds the read-only quality-criteria report shown at the
// Approval Gate. It only reads records that earlier stages already wrote and never
// recomputes a criterion. Every row is always present and always filled: a value,
// or the default state "não avaliado" (the stage did not run or was not enabled),
// or "não sei" (data exists but cannot support a statement). Nothing is guessed
// and nothing defaults to a pass.
package gatecriteria

import (
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Row states.
const (
	StateEvaluated    = "avaliado"
	StateNotEvaluated = "não avaliado"
	StateUnknown      = "não sei"
)

// Row is one criterion. Every field is non-empty.
type Row struct {
	Criterion string `json:"criterion"`
	State     string `json:"state"`
	Value     string `json:"value"`
	Source    string `json:"source"`
	Detail    string `json:"detail"`
}

// Report is the full criteria set for a mission.
type Report struct {
	MissionID   string      `json:"mission_id"`
	Rows        []Row       `json:"rows"`
	TokenStages StageTokens `json:"token_stages"`
}

// Input locates the records of one mission.
type Input struct {
	Root      string
	BasePath  string
	MissionID string
}

func (in Input) memory(name string) string { return filepath.Join(in.Root, "memory", name) }

// Criteria lists every criterion in display order.
func Criteria() []string {
	return []string{
		"route_confidence", "claim_confidence.scout", "claim_confidence.ranger", "claim_confidence.archivist",
		"confidence_review", "handoff.ranger_to_archivist", "handoff.archivist_to_sniper",
		"critic", "mission_quality", "refinement_handoff_metrics",
		"provider.ranger_to_archivist", "provider.archivist_to_gate", "provider.archivist_to_sniper",
		stageVolumeCriterion,
	}
}

// Build gathers the rows from existing records.
func Build(in Input) Report {
	records := loadClaims(in)
	rows := []Row{routeRow(in)}
	for _, agent := range claimAgents() {
		rows = append(rows, claimRow(records, "claim_confidence."+agent, agent))
	}
	stages := TokenStages(in)
	rows = append(rows,
		reviewRow(in),
		handoffRow(in, "handoff.ranger_to_archivist", "ranger_to_archivist", "nenhum outcome registrado"),
		handoffRow(in, "handoff.archivist_to_sniper", "archivist_to_sniper", "só existe depois do gate"),
		claimRow(records, "critic", "response_critic"),
		claimRow(records, "mission_quality", "mission_quality"),
		refinementMetricsRow(in),
		providerRow(in, "provider.ranger_to_archivist", "ranger_to_archivist", "nenhuma leitura registrada"),
		providerRow(in, "provider.archivist_to_gate", "archivist_to_gate", "nenhuma leitura registrada"),
		providerRow(in, "provider.archivist_to_sniper", "archivist_to_sniper", "só existe depois do gate"),
		stageTokensRow(stages),
	)
	return Report{MissionID: in.MissionID, Rows: rows, TokenStages: stages}
}

// claimAgents are the boundaries whose claims exist before the gate.
func claimAgents() []string {
	return []string{telemetry.ConfidenceAgentScout, telemetry.ConfidenceAgentRanger, telemetry.ConfidenceAgentArchivist}
}

func evaluated(criterion, value, source, detail string) Row {
	return Row{Criterion: criterion, State: StateEvaluated, Value: value, Source: source, Detail: detail}
}

func notEvaluated(criterion, source, detail string) Row {
	return Row{Criterion: criterion, State: StateNotEvaluated, Value: StateNotEvaluated, Source: source, Detail: detail}
}

func unknown(criterion, source, detail string) Row {
	return Row{Criterion: criterion, State: StateUnknown, Value: StateUnknown, Source: source, Detail: detail}
}
