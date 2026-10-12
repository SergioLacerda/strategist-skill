package view

import "github.com/SergioLacerda/strategist-skill/internal/domain"

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
	rows := make([]PhaseRow, len(tl.Phases))
	copy(rows, tl.Phases)
	for i, row := range rows {
		if row.Phase == string(domain.PhaseApprovalGate) {
			continue
		}
		if b, ok := summary.ByPhase[row.Phase]; ok {
			rows[i].Tokens, rows[i].Quality = b.Tokens, b.Quality
			rows[i].Observations, rows[i].QualityBreakdown = b.Observations, b.QualityBreakdown
		}
	}
	tl.Phases = rows
	return tl
}
