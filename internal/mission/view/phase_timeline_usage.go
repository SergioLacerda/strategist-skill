package view

import "github.com/SergioLacerda/strategist-skill/internal/domain"

func phaseRunKey(phase, runID string) string {
	if phase == string(domain.PhaseBootstrap) {
		phase = string(domain.PhaseIntake)
	}
	return phase + "|" + runID
}

// WithUsage returns the timeline with token usage applied to its rows, the
// unattributed bucket, the mission total and the coverage counters. The legacy
// provider-reported ledger is never an input: it has its own row. An unreadable
// usage ledger leaves every token member null.
func (tl PhaseTimeline) WithUsage(summary UsageSummary, usageErr error) PhaseTimeline {
	empty := UsageBucket{Quality: QualityUnavailable}
	tl.Unattributed, tl.Total = empty, empty
	switch {
	case usageErr != nil:
		tl.UsageAvailability = Unavailable
		return tl
	case summary.Total.Observations == 0 && summary.Coverage == (UsageCoverage{}):
		tl.UsageAvailability = NotApplicable
		return tl
	}
	tl.UsageAvailability = Available
	tl.Unattributed, tl.Total, tl.Coverage = summary.Unattributed, summary.Total, summary.Coverage
	tl.Phases = projectUsageRows(tl.Phases, summary)
	return tl
}

func projectUsageRows(source []PhaseRow, summary UsageSummary) []PhaseRow {
	rows := append([]PhaseRow(nil), source...)
	for i := range rows {
		rows[i].RunDetails = projectRunUsage(rows[i].RunDetails, summary.ByRun)
		if rows[i].Phase == string(domain.PhaseApprovalGate) {
			continue
		}
		if b, ok := summary.ByPhase[rows[i].Phase]; ok {
			rows[i].Tokens, rows[i].Quality = b.Tokens, b.Quality
			rows[i].Observations, rows[i].QualityBreakdown = b.Observations, b.QualityBreakdown
		}
	}
	return rows
}

func projectRunUsage(source []PhaseRun, byRun map[string]UsageBucket) []PhaseRun {
	runs := append([]PhaseRun(nil), source...)
	for i := range runs {
		runs[i].Quality = QualityUnavailable
		if b, ok := byRun[phaseRunKey(runs[i].Phase, runs[i].RunID)]; ok {
			runs[i].Tokens, runs[i].Quality = b.Tokens, b.Quality
			runs[i].Observations, runs[i].QualityBreakdown = b.Observations, b.QualityBreakdown
		}
	}
	return runs
}
