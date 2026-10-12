package verify

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// correlationFindings checks the identity joins that make a usage observation
// attributable. Missing historical authorities remain readable, but are
// reported as unverifiable instead of being silently treated as valid.
func correlationFindings(r telemetry.MissionUsageRecord, missionID string, lifecycle []telemetry.MissionLifecycleRecord, knownInvocations []string) []Finding {
	var out []Finding
	if missionID != "" && r.MissionID != missionID {
		out = append(out, Finding{
			Code: "usage_mission_mismatch", Severity: SeverityError, Ref: r.RecordID,
			Message: fmt.Sprintf("usage belongs to mission %q, but the verified mission is %q", r.MissionID, missionID),
		})
	}
	// An empty known-invocation set is deliberately not a failure: callers may
	// be verifying a legacy export without the invocation ledger. When the set
	// is available, contextFindings below checks the id against it.
	_ = knownInvocations
	if r.Phase == domain.UsagePhaseUnattributed {
		return out
	}
	if r.PhaseRunID == "" {
		// Legacy records predate the lifecycle join. They remain readable and
		// are not re-written; new CLI records use the resolver port when one is
		// available. There is no safe run identity to validate here.
		return out
	}
	if len(lifecycle) == 0 {
		out = append(out, Finding{
			Code: "usage_phase_run_unverifiable", Severity: SeverityInfo, Ref: r.RecordID,
			Message: "phase_run_id is present but the lifecycle ledger is unavailable",
		})
		return out
	}
	if !lifecycleHasRun(lifecycle, r.Phase, r.PhaseRunID) {
		out = append(out, Finding{
			Code: "usage_phase_run_unknown", Severity: SeverityError, Ref: r.RecordID,
			Message: fmt.Sprintf("phase_run_id %q was not started for phase %q", r.PhaseRunID, r.Phase),
		})
	}
	return out
}

func lifecycleHasRun(records []telemetry.MissionLifecycleRecord, phase, runID string) bool {
	for _, record := range records {
		if record.MarkerKind == string(domain.MarkerPhaseStarted) && record.Phase == phase && record.PhaseRunID == runID {
			return true
		}
	}
	return false
}
