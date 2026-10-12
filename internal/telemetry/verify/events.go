package verify

import "github.com/SergioLacerda/strategist-skill/internal/telemetry"

// eventFindings cross-checks the exported stream against the ledgers: the same
// report must be reconstructable from valid events, so a record with no event,
// an event with no record, or a count that differs is reported.
func eventFindings(in Input) []Finding {
	var out []Finding
	lifecycle := map[string]telemetry.MissionLifecycleRecord{}
	usage := map[string]telemetry.MissionUsageRecord{}
	for _, e := range in.Events {
		out = append(out, noteEvent(e, in.Status.MissionID, lifecycle, usage)...)
	}
	for _, r := range in.Lifecycle {
		if _, ok := lifecycle[r.MarkerID]; !ok {
			out = append(out, Finding{Code: "event_missing_for_record", Severity: SeverityWarning, Ref: r.MarkerID, Message: "no exported event for this lifecycle marker"})
		}
	}
	for _, r := range in.Usage {
		if _, ok := usage[r.RecordID]; !ok {
			out = append(out, Finding{Code: "event_missing_for_record", Severity: SeverityWarning, Ref: r.RecordID, Message: "no exported event for this usage record"})
		}
	}
	return append(out, orphanEvents(in, lifecycle, usage)...)
}

func noteEvent(e telemetry.Event, missionID string, lifecycle map[string]telemetry.MissionLifecycleRecord, usage map[string]telemetry.MissionUsageRecord) []Finding {
	if rec, ok := telemetry.MissionLifecycleRecordFromEvent(e); ok && rec.MissionID == missionID {
		return noteLifecycleEvent(rec, lifecycle)
	}
	if rec, ok := telemetry.MissionUsageRecordFromEvent(e); ok && rec.MissionID == missionID {
		return noteUsageEvent(rec, usage)
	}
	return nil
}

func noteLifecycleEvent(rec telemetry.MissionLifecycleRecord, lifecycle map[string]telemetry.MissionLifecycleRecord) []Finding {
	if _, dup := lifecycle[rec.MarkerID]; dup {
		return []Finding{{Code: "event_duplicate", Severity: SeverityWarning, Ref: rec.MarkerID, Message: "the stream carries this lifecycle event more than once"}}
	}
	lifecycle[rec.MarkerID] = rec
	return nil
}

func noteUsageEvent(rec telemetry.MissionUsageRecord, usage map[string]telemetry.MissionUsageRecord) []Finding {
	if prior, dup := usage[rec.RecordID]; dup {
		if !sameCounts(prior, rec) {
			return []Finding{{Code: "event_conflicting_duplicate", Severity: SeverityError, Ref: rec.RecordID, Message: "the stream carries this usage event twice with different counts"}}
		}
		return []Finding{{Code: "event_duplicate", Severity: SeverityWarning, Ref: rec.RecordID, Message: "the stream carries this usage event more than once"}}
	}
	usage[rec.RecordID] = rec
	return nil
}

func orphanEvents(in Input, lifecycle map[string]telemetry.MissionLifecycleRecord, usage map[string]telemetry.MissionUsageRecord) []Finding {
	return append(orphanLifecycleEvents(in, lifecycle), orphanUsageEvents(in, usage)...)
}

func orphanLifecycleEvents(in Input, lifecycle map[string]telemetry.MissionLifecycleRecord) []Finding {
	var out []Finding
	ledger := map[string]bool{}
	for _, r := range in.Lifecycle {
		ledger[r.MarkerID] = true
	}
	for id := range lifecycle {
		if !ledger[id] {
			out = append(out, Finding{Code: "event_without_record", Severity: SeverityError, Ref: id, Message: "the stream has a lifecycle event the ledger does not"})
		}
	}
	return out
}

func orphanUsageEvents(in Input, usage map[string]telemetry.MissionUsageRecord) []Finding {
	var out []Finding
	byID := map[string]telemetry.MissionUsageRecord{}
	for _, r := range in.Usage {
		byID[r.RecordID] = r
	}
	for id, ev := range usage {
		rec, ok := byID[id]
		switch {
		case !ok:
			out = append(out, Finding{Code: "event_without_record", Severity: SeverityError, Ref: id, Message: "the stream has a usage event the ledger does not"})
		case !sameCounts(rec, ev):
			out = append(out, Finding{Code: "event_conflicts_with_record", Severity: SeverityError, Ref: id, Message: "the event and the ledger record carry different counts"})
		}
	}
	return out
}
