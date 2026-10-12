package view

import (
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Timeline row and mission statuses. `interrupted` is deliberately absent:
// Wave 1 never infers it (a mission without a terminal marker is `running`).
const (
	StatusRunning        = "running"
	StatusSkipped        = "skipped"
	StatusNotStarted     = "not_started"
	StatusNotRecorded    = "not_recorded"
	StatusGap            = "gap"
	StatusBlocked        = "blocked"
	StatusCompleted      = "completed"
	QualityUnavailable   = "unavailable"
	QualityNotApplicable = "not_applicable"
	noLifecycleNote      = "no lifecycle recorded"
)

// PhaseRun is one run of a canonical phase.
type PhaseRun struct {
	Phase     string  `json:"phase"`
	RunID     string  `json:"run_id"`
	StartedAt string  `json:"started_at"`
	EndedAt   *string `json:"ended_at"`
	EndReason string  `json:"end_reason,omitempty"`
}

// PhaseRow is one display row of the timeline.
type PhaseRow struct {
	Phase           string      `json:"phase"`
	Label           string      `json:"label"`
	StartedAt       *string     `json:"started_at"`
	EndedAt         *string     `json:"ended_at"`
	DurationSeconds *int64      `json:"duration_seconds"`
	Runs            int         `json:"runs"`
	Status          string      `json:"status"`
	Tokens          PhaseTokens `json:"tokens"`
	Quality         string      `json:"quality"`
	Observations    int         `json:"observations"`
	// QualityBreakdown lists the qualities behind a `mixed` row.
	QualityBreakdown map[string]int `json:"quality_breakdown,omitempty"`
	RunDetails       []PhaseRun     `json:"run_details,omitempty"`
}

// PhaseTimeline is the per-phase lifecycle projection shared by `mission view`
// and `mission usage`.
type PhaseTimeline struct {
	Availability    string     `json:"availability"`
	Note            string     `json:"note,omitempty"`
	Route           string     `json:"route,omitempty"`
	Status          string     `json:"status"`
	StartedAt       *string    `json:"started_at"`
	EndedAt         *string    `json:"ended_at"`
	DurationSeconds *int64     `json:"duration_seconds"`
	LastMarkerAt    *string    `json:"last_marker_at"`
	Phases          []PhaseRow `json:"phases"`
	// UsageAvailability reports the usage ledger, independently of the
	// lifecycle ledger: available, not_applicable (no observation) or unavailable.
	UsageAvailability string        `json:"usage_availability"`
	Unattributed      UsageBucket   `json:"unattributed"`
	Total             UsageBucket   `json:"total"`
	Coverage          UsageCoverage `json:"coverage"`
}

type timelineGroup struct {
	phase, label, quality string
	members               []string
}

var timelineGroups = []timelineGroup{
	{"INTAKE", "Intake", QualityUnavailable, []string{string(domain.PhaseBootstrap), string(domain.PhaseIntake)}},
	{string(domain.PhaseDiscovery), "Discovery", QualityUnavailable, []string{string(domain.PhaseDiscovery)}},
	{string(domain.PhaseRefinement), "Refining", QualityUnavailable, []string{string(domain.PhaseRefinement)}},
	{string(domain.PhaseApprovalGate), "Gate", QualityNotApplicable, []string{string(domain.PhaseApprovalGate)}},
	{string(domain.PhaseExecution), "Delivery", QualityUnavailable, []string{string(domain.PhaseExecution)}},
}

// BuildPhaseTimeline folds the mission's lifecycle markers and the engine
// status into per-phase rows. A read error is `unavailable`; an empty ledger is
// `not_applicable` with the note "no lifecycle recorded". It never invents a
// time: an open run has no end and contributes no duration.
func BuildPhaseTimeline(status domain.MissionEngineStatus, recs []telemetry.MissionLifecycleRecord, readErr error) PhaseTimeline {
	switch {
	case readErr != nil:
		return PhaseTimeline{Availability: Unavailable, Status: Unavailable, Phases: skeletonRows(StatusNotRecorded)}
	case len(recs) == 0:
		return PhaseTimeline{Availability: NotApplicable, Note: noLifecycleNote, Status: StatusNotRecorded, Phases: skeletonRows(StatusNotRecorded)}
	}
	runs := collectRuns(recs)
	mission := missionSummary(status, recs)
	tl := PhaseTimeline{
		Availability: Available, Route: stageOf(recs), Status: mission.status,
		StartedAt: mission.startedAt, EndedAt: mission.endedAt, LastMarkerAt: ptr(recs[len(recs)-1].ObservedAt),
	}
	tl.DurationSeconds = secondsBetween(tl.StartedAt, tl.EndedAt)
	ctx := rowContext{missionEnded: mission.outcome != "", engineRow: engineRowFor(status.Phase), enginePhase: string(status.Phase)}
	for _, g := range timelineGroups {
		tl.Phases = append(tl.Phases, buildRow(g, runs, ctx))
	}
	return tl
}

func skeletonRows(status string) []PhaseRow {
	rows := make([]PhaseRow, 0, len(timelineGroups))
	for _, g := range timelineGroups {
		rows = append(rows, PhaseRow{Phase: g.phase, Label: g.label, Status: status, Quality: g.quality})
	}
	return rows
}

func stageOf(recs []telemetry.MissionLifecycleRecord) string {
	for _, r := range recs {
		if r.Stage != "" {
			return r.Stage
		}
	}
	return ""
}

func ptr(s string) *string { return &s }
