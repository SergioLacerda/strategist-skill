package missionview

import "github.com/SergioLacerda/strategist-skill/internal/telemetry"

// TokenUsageSection reports the mission's explicitly agent-reported token
// usage (`strategist mission report-usage`), giving
// memory/mission-token-usage.jsonl its first reader (F-T2, ADR-0057 §
// design.md task 3.3 — the code change
// .analysis/pending/melhorias/confidence-and-metrics-improvements/20260925-token-economy-baseline-protocol.md's own § 1
// and Open Items defer). DeclaredTokenBudget is skill.yaml's
// budget_policy.token_budget verbatim (a qualitative tier — "high", not a
// token count): it is surfaced next to the reported totals for the reader to
// judge, not compared numerically, because no config in this workspace maps
// a tier to a token ceiling. NotApplicable means no usage has ever been
// reported for this mission, which is the common case, not an error.
//
// Split out of view.go to keep that file under the repo's file-size budget.
type TokenUsageSection struct {
	Availability        string                              `json:"availability"`
	DeclaredTokenBudget string                              `json:"declared_token_budget,omitempty"`
	TotalTokensIn       int64                               `json:"total_tokens_in"`
	TotalTokensOut      int64                               `json:"total_tokens_out"`
	Records             []telemetry.MissionTokenUsageRecord `json:"records,omitempty"`
	LedgerComparison    telemetry.TokenLedgerComparison     `json:"ledger_comparison"`
}

// buildTokenUsage sums the mission's reported token-usage records.
// Availability is NotApplicable when none were ever reported for this
// mission — the common case (most missions run through a host agent that
// never calls `mission report-usage`), not an error.
func buildTokenUsage(records []telemetry.MissionTokenUsageRecord, handoff []telemetry.RefinementHandoffLine, declaredBudget string) TokenUsageSection {
	section := TokenUsageSection{Availability: NotApplicable, DeclaredTokenBudget: declaredBudget, LedgerComparison: telemetry.CompareTokenLedgers(records, handoff)}
	if len(records) == 0 {
		return section
	}
	section.Availability = Available
	section.Records = records
	for _, rec := range records {
		section.TotalTokensIn += rec.TokensIn
		section.TotalTokensOut += rec.TokensOut
	}
	return section
}
