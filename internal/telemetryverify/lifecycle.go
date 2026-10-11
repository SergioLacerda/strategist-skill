package telemetryverify

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

type runState struct {
	open    bool
	startAt time.Time
	count   int
}

func lifecycleFindings(in Input) []Finding {
	var out []Finding
	seen := map[string]bool{}
	runs := map[string]*runState{}
	var prev time.Time
	for _, r := range in.Lifecycle {
		found, at, ok := analyseMarker(r, seen, runs, prev)
		out = append(out, found...)
		if ok {
			prev = at
		}
	}
	return append(out, engineFindings(in, runs)...)
}

// analyseMarker validates one marker and folds it into the run state. ok is
// false when the marker could not be analysed (invalid or duplicated).
func analyseMarker(r telemetry.MissionLifecycleRecord, seen map[string]bool, runs map[string]*runState, prev time.Time) ([]Finding, time.Time, bool) {
	if bad := recordProblems(r, seen); bad != nil {
		return []Finding{*bad}, time.Time{}, false
	}
	seen[r.MarkerID] = true
	at, err := time.Parse(time.RFC3339, r.ObservedAt)
	if err != nil {
		return nil, time.Time{}, false // already reported as invalid by recordProblems
	}
	var out []Finding
	if !prev.IsZero() && at.Before(prev) {
		out = append(out, Finding{Code: "lifecycle_time_regression", Severity: SeverityWarning, Ref: r.MarkerID, Message: "marker is older than the one before it in file order"})
	}
	return append(out, runFindings(r, at, runs)...), at, true
}

// recordProblems returns a finding when the record is invalid or repeats a
// marker id already seen; nil means the record may be analysed further.
func recordProblems(r telemetry.MissionLifecycleRecord, seen map[string]bool) *Finding {
	if err := telemetry.ValidateMissionLifecycleRecord(r); err != nil {
		return &Finding{Code: "lifecycle_record_invalid", Severity: SeverityError, Ref: r.MarkerID, Message: err.Error()}
	}
	if seen[r.MarkerID] {
		return &Finding{Code: "lifecycle_duplicate_marker", Severity: SeverityError, Ref: r.MarkerID, Message: "the same marker id appears more than once"}
	}
	return nil
}

func runFindings(r telemetry.MissionLifecycleRecord, at time.Time, runs map[string]*runState) []Finding {
	switch r.MarkerKind {
	case string(domain.MarkerPhaseStarted):
		return startFindings(r, at, runs)
	case string(domain.MarkerPhaseEnded):
		return endFindings(r, at, runs)
	case string(domain.MarkerMissionEnded):
		return missionEndFindings(r, runs)
	}
	return nil
}

func startFindings(r telemetry.MissionLifecycleRecord, at time.Time, runs map[string]*runState) []Finding {
	st := runs[r.Phase]
	if st == nil {
		st = &runState{}
		runs[r.Phase] = st
	}
	var out []Finding
	if st.open {
		out = append(out, Finding{Code: "lifecycle_overlapping_run", Severity: SeverityError, Ref: r.MarkerID, Message: "a run of " + r.Phase + " started while another was still open"})
	}
	st.count++
	if want := fmt.Sprintf("r%d", st.count); r.PhaseRunID != want {
		out = append(out, Finding{Code: "lifecycle_run_id_not_contiguous", Severity: SeverityWarning, Ref: r.MarkerID, Message: "expected " + want + ", found " + r.PhaseRunID})
	}
	st.open, st.startAt = true, at
	return out
}

func endFindings(r telemetry.MissionLifecycleRecord, at time.Time, runs map[string]*runState) []Finding {
	st := runs[r.Phase]
	switch {
	case st == nil || !st.open:
		return []Finding{{Code: "lifecycle_end_without_start", Severity: SeverityError, Ref: r.MarkerID, Message: "a phase ended with no open run"}}
	case at.Before(st.startAt):
		st.open = false
		return []Finding{{Code: "lifecycle_end_before_start", Severity: SeverityError, Ref: r.MarkerID, Message: "the end is earlier than its start"}}
	}
	st.open = false
	return nil
}

func missionEndFindings(r telemetry.MissionLifecycleRecord, runs map[string]*runState) []Finding {
	var out []Finding
	for phase, st := range runs {
		if st.open {
			out = append(out, Finding{Code: "lifecycle_open_run_at_mission_end", Severity: SeverityError, Ref: r.MarkerID, Message: phase + " is still open when the mission ended"})
		}
	}
	return out
}

// engineFindings compares the ledger with the engine status: a gap is the
// difference between "the mission did this" and "the ledger recorded it".
func engineFindings(in Input, runs map[string]*runState) []Finding {
	if len(in.Lifecycle) == 0 {
		return nil
	}
	var out []Finding
	if in.Status.Phase == domain.PhaseDone && !hasKind(in.Lifecycle, domain.MarkerMissionEnded) {
		out = append(out, Finding{Code: "lifecycle_engine_mismatch", Severity: SeverityWarning, Message: "the mission is done but no mission_ended marker was recorded"})
	}
	for phase, st := range runs {
		if st.open && string(in.Status.Phase) != phase {
			out = append(out, Finding{Code: "lifecycle_engine_mismatch", Severity: SeverityWarning, Ref: phase, Message: phase + " has an open run but the engine is in " + string(in.Status.Phase)})
		}
	}
	return out
}

func hasKind(recs []telemetry.MissionLifecycleRecord, kind domain.LifecycleMarkerKind) bool {
	for _, r := range recs {
		if r.MarkerKind == string(kind) {
			return true
		}
	}
	return false
}
