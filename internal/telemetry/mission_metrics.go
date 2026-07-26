package telemetry

import (
	"fmt"
	"log/slog"
)

// MissionMetrics captures the canonical end-to-end timing and volume fields
// used to instrument Strategist missions.
type MissionMetrics struct {
	MissionID            string
	TStartToIntakeMS     int64
	TIntakeToScoutMS     int64 // decomposition of TIntakeToRangerMS: intake -> Scout route decision
	TScoutToRangerMS     int64 // decomposition of TIntakeToRangerMS: Scout route decision -> Ranger start
	TIntakeToRangerMS    int64
	TRangerToArchivistMS int64
	TArchivistToGateMS   int64
	TGateWaitMS          int64 // human latency between gate presented and response
	TGateToSniperMS      int64
	TSniperToDoneMS      int64
	TotalWallTimeMS      int64
	TokensIn             int64
	TokensOut            int64
	LinesEmitted         int64
}

// FormatMissionMetrics returns a canonical mission metrics line.
func FormatMissionMetrics(m MissionMetrics) string {
	return fmt.Sprintf(
		"[Strategist] metrics mission=%s t_start_to_intake_ms=%d t_intake_to_scout_ms=%d t_scout_to_ranger_ms=%d t_intake_to_ranger_ms=%d t_ranger_to_archivist_ms=%d t_archivist_to_gate_ms=%d t_gate_wait_ms=%d t_gate_to_sniper_ms=%d t_sniper_to_done_ms=%d total_wall_time_ms=%d tokens_in=%d tokens_out=%d lines_emitted=%d",
		m.MissionID,
		m.TStartToIntakeMS,
		m.TIntakeToScoutMS,
		m.TScoutToRangerMS,
		m.TIntakeToRangerMS,
		m.TRangerToArchivistMS,
		m.TArchivistToGateMS,
		m.TGateWaitMS,
		m.TGateToSniperMS,
		m.TSniperToDoneMS,
		m.TotalWallTimeMS,
		m.TokensIn,
		m.TokensOut,
		m.LinesEmitted,
	)
}

// EmitMissionMetrics logs a canonical mission metrics line through slog.
func EmitMissionMetrics(m MissionMetrics) {
	slog.Info(
		FormatMissionMetrics(m),
		AttrMissionID, m.MissionID,
		AttrStartToIntakeMS, m.TStartToIntakeMS,
		AttrIntakeToScoutMS, m.TIntakeToScoutMS,
		AttrScoutToRangerMS, m.TScoutToRangerMS,
		AttrIntakeToRangerMS, m.TIntakeToRangerMS,
		AttrRangerToArchivistMS, m.TRangerToArchivistMS,
		AttrArchivistToGateMS, m.TArchivistToGateMS,
		AttrGateWaitMS, m.TGateWaitMS,
		AttrGateToSniperMS, m.TGateToSniperMS,
		AttrSniperToDoneMS, m.TSniperToDoneMS,
		AttrTotalWallTimeMS, m.TotalWallTimeMS,
		AttrTokensIn, m.TokensIn,
		AttrTokensOut, m.TokensOut,
		AttrLinesEmitted, m.LinesEmitted,
	)
}
