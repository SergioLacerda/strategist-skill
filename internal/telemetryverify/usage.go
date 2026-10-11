package telemetryverify

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func usageFindings(in Input) []Finding {
	var out []Finding
	byID := map[string]telemetry.MissionUsageRecord{}
	known := toSet(in.KnownInvocations)
	started := startedPhases(in.Lifecycle)
	snapshots := map[string]telemetry.MissionUsageRecord{}
	unkeyed, unattributed := 0, 0
	for _, r := range in.Usage {
		if err := telemetry.ValidateMissionUsageRecord(r); err != nil {
			out = append(out, Finding{Code: "usage_record_invalid", Severity: SeverityError, Ref: r.RecordID, Message: err.Error()})
			continue
		}
		if prior, dup := byID[r.RecordID]; dup {
			out = append(out, duplicateUsage(prior, r))
			continue
		}
		byID[r.RecordID] = r
		out = append(out, contextFindings(r, known, started, len(in.Lifecycle) > 0)...)
		out = append(out, snapshotFindings(r, snapshots)...)
		unkeyed += boolInt(r.DedupUnavailable)
		unattributed += boolInt(r.Phase == domain.UsagePhaseUnattributed)
	}
	return append(out, usageInfo(unkeyed, unattributed)...)
}

func duplicateUsage(prior, r telemetry.MissionUsageRecord) Finding {
	if sameCounts(prior, r) {
		return Finding{Code: "usage_duplicate_record", Severity: SeverityWarning, Ref: r.RecordID, Message: "the same record appears more than once; aggregation counts it once"}
	}
	return Finding{Code: "usage_conflicting_counts", Severity: SeverityError, Ref: r.RecordID, Message: "the same key carries different counts"}
}

func contextFindings(r telemetry.MissionUsageRecord, known, started map[string]bool, haveLifecycle bool) []Finding {
	var out []Finding
	if r.InvocationID != "" && len(known) > 0 && !known[r.InvocationID] {
		out = append(out, Finding{Code: "usage_unknown_invocation", Severity: SeverityWarning, Ref: r.RecordID, Message: "the invocation id was not issued by this mission"})
	}
	if haveLifecycle && r.Phase != domain.UsagePhaseUnattributed && !started[usageRow(r.Phase)] {
		out = append(out, Finding{Code: "usage_phase_never_started", Severity: SeverityWarning, Ref: r.RecordID, Message: r.Phase + " has usage but no lifecycle run"})
	}
	return out
}

// snapshotFindings flags a cumulative counter that went backwards inside one
// session: that interval is unavailable in the aggregation.
func snapshotFindings(r telemetry.MissionUsageRecord, last map[string]telemetry.MissionUsageRecord) []Finding {
	if r.RecordKind != string(domain.UsageKindCumulativeSnapshot) {
		return nil
	}
	key := r.SessionID + "|" + r.UsageScope
	prev, had := last[key]
	last[key] = r
	if had && regressed(prev, r) {
		return []Finding{{Code: "usage_counter_reset", Severity: SeverityWarning, Ref: r.RecordID, Message: fmt.Sprintf("the cumulative counter of session %q went backwards: the interval is unavailable", r.SessionID)}}
	}
	return nil
}

func regressed(prev, cur telemetry.MissionUsageRecord) bool {
	lower := func(a, b *int64) bool { return a != nil && b != nil && *b < *a }
	return lower(prev.TokensInput, cur.TokensInput) || lower(prev.TokensOutput, cur.TokensOutput)
}

func usageInfo(unkeyed, unattributed int) []Finding {
	var out []Finding
	if unkeyed > 0 {
		out = append(out, Finding{Code: "usage_dedup_unavailable", Severity: SeverityInfo, Message: fmt.Sprintf("%d record(s) have no invocation or event id and cannot be deduplicated", unkeyed)})
	}
	if unattributed > 0 {
		out = append(out, Finding{Code: "usage_unattributed", Severity: SeverityInfo, Message: fmt.Sprintf("%d record(s) are not attributed to a phase", unattributed)})
	}
	return out
}

func sameCounts(a, b telemetry.MissionUsageRecord) bool {
	eq := func(x, y *int64) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	return eq(a.TokensInput, b.TokensInput) && eq(a.TokensOutput, b.TokensOutput) &&
		eq(a.TokensCache, b.TokensCache) && eq(a.TokensReasoning, b.TokensReasoning)
}

func startedPhases(recs []telemetry.MissionLifecycleRecord) map[string]bool {
	out := map[string]bool{}
	for _, r := range recs {
		if r.MarkerKind == string(domain.MarkerPhaseStarted) {
			out[usageRow(r.Phase)] = true
		}
	}
	return out
}

// usageRow groups bootstrap with intake, as the timeline does.
func usageRow(phase string) string {
	if phase == string(domain.PhaseBootstrap) {
		return string(domain.PhaseIntake)
	}
	return phase
}

func toSet(ids []string) map[string]bool {
	out := map[string]bool{}
	for _, id := range ids {
		if id != "" {
			out[id] = true
		}
	}
	return out
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
