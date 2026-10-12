package view

import (
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry/usagesummary"
)

// The aggregation lives in internal/telemetry/usagesummary so the Approval Gate criteria
// can use it without importing the read model. These aliases keep the read
// model's vocabulary and JSON shape unchanged.
type (
	// PhaseTokens holds per-phase token counts; a member is null when unobserved.
	PhaseTokens = usagesummary.Tokens
	// UsageBucket is the token usage of one row, the unattributed bucket or the mission.
	UsageBucket = usagesummary.Bucket
	// UsageCoverage states how much of the mission the numbers cover.
	UsageCoverage = usagesummary.Coverage
	// UsageSummary is the aggregation of a mission's usage ledger.
	UsageSummary = usagesummary.Summary
)

// QualityMixed labels a bucket whose observations carry different qualities.
const QualityMixed = usagesummary.QualityMixed

// SummarizeUsage aggregates declared observations (see usagesummary.Summarize).
func SummarizeUsage(recs []telemetry.MissionUsageRecord, knownInvocations []string) UsageSummary {
	return usagesummary.Summarize(recs, knownInvocations)
}
