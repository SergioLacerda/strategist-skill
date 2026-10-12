package view

import (
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

type rowContext struct {
	missionEnded bool
	engineRow    string
	enginePhase  string
}

type missionFacts struct {
	status, outcome    string
	startedAt, endedAt *string
}

// collectRuns pairs phase start and end markers into runs, in the order the
// runs started.
func collectRuns(recs []telemetry.MissionLifecycleRecord) []PhaseRun {
	var runs []PhaseRun
	for _, r := range recs {
		switch r.MarkerKind {
		case string(domain.MarkerPhaseStarted):
			runs = append(runs, PhaseRun{Phase: r.Phase, RunID: r.PhaseRunID, StartedAt: r.ObservedAt})
		case string(domain.MarkerPhaseEnded):
			closeRun(runs, r)
		}
	}
	return runs
}

func closeRun(runs []PhaseRun, end telemetry.MissionLifecycleRecord) {
	for i := len(runs) - 1; i >= 0; i-- {
		if runs[i].Phase == end.Phase && runs[i].RunID == end.PhaseRunID && runs[i].EndedAt == nil {
			runs[i].EndedAt = ptr(end.ObservedAt)
			runs[i].EndReason = end.EndReason
			return
		}
	}
}

// missionSummary derives the mission status. The first mission_ended marker is
// the outcome; the last one closes the timeline (an ADR flow may follow DONE).
func missionSummary(status domain.MissionEngineStatus, recs []telemetry.MissionLifecycleRecord) missionFacts {
	facts := missionFacts{startedAt: ptr(recs[0].ObservedAt)}
	for _, r := range recs {
		switch r.MarkerKind {
		case string(domain.MarkerMissionStarted):
			facts.startedAt = ptr(r.ObservedAt)
		case string(domain.MarkerMissionEnded):
			if facts.outcome == "" {
				facts.outcome = r.EndReason
			}
			facts.endedAt = ptr(r.ObservedAt)
		}
	}
	facts.status = facts.outcome
	switch {
	case facts.outcome != "":
	case status.Phase == domain.PhaseBlocked:
		facts.status = StatusBlocked
	case status.Phase == domain.PhaseDone:
		facts.status = StatusGap
	default:
		facts.status = StatusRunning
	}
	return facts
}

func engineRowFor(phase domain.PipelinePhase) string {
	switch phase {
	case domain.PhaseBootstrap, domain.PhaseIntake:
		return "INTAKE"
	case domain.PhaseDiscovery, domain.PhaseRefinement, domain.PhaseApprovalGate, domain.PhaseExecution:
		return string(phase)
	case domain.PhaseDone, domain.PhaseBlocked:
		return ""
	}
	return ""
}

func buildRow(g timelineGroup, all []PhaseRun, ctx rowContext) PhaseRow {
	row := PhaseRow{Phase: g.phase, Label: g.label, Quality: g.quality}
	for _, r := range all {
		if inGroup(g, r.Phase) {
			row.RunDetails = append(row.RunDetails, r)
		}
	}
	row.Runs = len(row.RunDetails)
	row.Status = rowStatus(row.RunDetails, ctx, g.phase)
	if row.Runs == 0 {
		return row
	}
	row.StartedAt = ptr(row.RunDetails[0].StartedAt)
	row.EndedAt, row.DurationSeconds = rowEnd(row.RunDetails)
	return row
}

func inGroup(g timelineGroup, phase string) bool {
	for _, m := range g.members {
		if m == phase {
			return true
		}
	}
	return false
}

func rowStatus(runs []PhaseRun, ctx rowContext, phase string) string {
	if len(runs) == 0 {
		if ctx.missionEnded {
			return StatusSkipped
		}
		return StatusNotStarted
	}
	last := runs[len(runs)-1]
	switch {
	case last.EndedAt == nil && last.Phase == ctx.enginePhase && !ctx.missionEnded:
		return StatusRunning
	case last.EndedAt == nil:
		return StatusGap // an open run the engine has already left: its end marker was lost
	case ctx.engineRow == phase && !ctx.missionEnded:
		return StatusGap
	case last.EndReason == string(domain.EndBlocked), last.EndReason == string(domain.EndRejected),
		last.EndReason == string(domain.EndTimedOut), last.EndReason == string(domain.EndAnalysisDelivered):
		return last.EndReason
	}
	return StatusCompleted
}

// rowEnd returns the latest end when every run is closed (nil while one is
// open) and the sum of the closed runs' durations (nil when none is closed).
func rowEnd(runs []PhaseRun) (*string, *int64) {
	var total int64
	closed, open := 0, false
	var latest *string
	for _, r := range runs {
		if r.EndedAt == nil {
			open = true
			continue
		}
		if secs := secondsBetween(ptr(r.StartedAt), r.EndedAt); secs != nil {
			total += *secs
			closed++
		}
		latest = r.EndedAt
	}
	var dur *int64
	if closed > 0 {
		dur = &total
	}
	if open {
		return nil, dur
	}
	return latest, dur
}

func secondsBetween(start, end *string) *int64 {
	if start == nil || end == nil {
		return nil
	}
	s, errS := time.Parse(time.RFC3339, *start)
	e, errE := time.Parse(time.RFC3339, *end)
	if errS != nil || errE != nil || e.Before(s) {
		return nil
	}
	secs := int64(e.Sub(s).Seconds())
	return &secs
}
