package telemetry

import (
	"fmt"
	"log/slog"
)

// MetricsScopePerInvocation is the MissionMetrics.Scope value every CLI
// invocation emits (F-T1/F-T2, ADR-0057 § design.md task 3.2). Strategist is
// a CLI invoked as a fresh subprocess per command — internal/mission's own
// MissionRun starts at that process's construction and ends microseconds
// later at Finish(), so the phase-to-phase deltas below (t_intake_to_scout_ms
// through t_sniper_to_done_ms) measure elapsed time within that one process,
// not across the mission's whole lifetime. A phase this invocation's own
// role never marked collapses to 0 by construction (see mission_run.go's
// Snapshot, which cascades an unmarked phase's timestamp forward from the
// previous one) — that 0 means "not marked in this process," never "this
// phase took no time." Before this constant existed, a mission's own
// `mission start`/`submit` output showed the real mission_id right next to a
// metrics line whose deltas were zero for this same, unstated reason, with no
// signal distinguishing "genuinely instantaneous" from "not measured here."
// Persisting marks across invocations to get true mission-level phase timing
// is a distinct, larger change, deliberately deferred — see
// .analysis/refined/20260927-strategist-hardening-review/tasks.md task 7.4.
const MetricsScopePerInvocation = "per_invocation"

// MissionMetrics captures the canonical timing and volume fields used to
// instrument one Strategist CLI invocation. See MetricsScopePerInvocation's
// doc comment for what the phase-to-phase deltas do and do not measure.
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

// FormatMissionMetrics returns a canonical mission metrics line. The trailing
// scope=per_invocation token (see MetricsScopePerInvocation) marks every
// t_*_ms field as this one CLI process's own elapsed time, never a
// cross-invocation mission-level measurement.
func FormatMissionMetrics(m MissionMetrics) string {
	return fmt.Sprintf(
		"[Strategist] metrics mission=%s t_start_to_intake_ms=%d t_intake_to_scout_ms=%d t_scout_to_ranger_ms=%d t_intake_to_ranger_ms=%d t_ranger_to_archivist_ms=%d t_archivist_to_gate_ms=%d t_gate_wait_ms=%d t_gate_to_sniper_ms=%d t_sniper_to_done_ms=%d total_wall_time_ms=%d tokens_in=%d tokens_out=%d lines_emitted=%d scope=%s",
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
		MetricsScopePerInvocation,
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
		AttrMetricsScope, MetricsScopePerInvocation,
	)
}

// Explicit, agent-reported mission token-usage records
// (MissionTokenUsageRecord and its read/write/validate helpers) live in
// mission_token_usage.go, split out to keep this file under the repo's
// file-size budget.
