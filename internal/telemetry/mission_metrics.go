package telemetry

import (
	"fmt"
	"log/slog"
)

// MissionMetrics captures the canonical end-to-end timing and volume fields
// used to instrument Strategist missions.
type MissionMetrics struct {
	MissionID         string
	TStartToIntakeMS  int64
	TIntakeToRangerMS int64
	TotalWallTimeMS   int64
	TokensIn          int64
	TokensOut         int64
	LinesEmitted      int64
}

// FormatMissionMetrics returns a canonical mission metrics line.
func FormatMissionMetrics(m MissionMetrics) string {
	return fmt.Sprintf(
		"[Strategist] metrics mission=%s t_start_to_intake_ms=%d t_intake_to_ranger_ms=%d total_wall_time_ms=%d tokens_in=%d tokens_out=%d lines_emitted=%d",
		m.MissionID,
		m.TStartToIntakeMS,
		m.TIntakeToRangerMS,
		m.TotalWallTimeMS,
		m.TokensIn,
		m.TokensOut,
		m.LinesEmitted,
	)
}

// EmitMissionMetrics logs a canonical mission metrics line through slog.
func EmitMissionMetrics(m MissionMetrics) {
	slog.Info(FormatMissionMetrics(m))
}
