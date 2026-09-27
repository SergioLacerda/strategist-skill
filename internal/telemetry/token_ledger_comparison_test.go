package telemetry

import "testing"

func TestCompareTokenLedgersPreservesSeparateAuthorities(t *testing.T) {
	reported := []MissionTokenUsageRecord{{MissionID: "m", TokensIn: 100, TokensOut: 20}}
	in := int64(40)
	brief := int64(10)
	got := CompareTokenLedgers(reported, []RefinementHandoffLine{{MissionID: "m", DiscoveryTokens: &in, BriefTokens: &brief}})
	if got.Status != TokenLedgerNotComparable {
		t.Fatalf("status = %q, want %q", got.Status, TokenLedgerNotComparable)
	}
	if got.ReportedRecordCount != 1 || got.HandoffRecordCount != 1 {
		t.Fatalf("unexpected provenance counts: %+v", got)
	}
}

func TestCompareTokenLedgersSurfacesContradictoryReports(t *testing.T) {
	got := CompareTokenLedgers([]MissionTokenUsageRecord{
		{MissionID: "m", TokensIn: 100, TokensOut: 20},
		{MissionID: "m", TokensIn: 101, TokensOut: 20},
	}, nil)
	if got.Status != TokenLedgerInconsistent || len(got.Inconsistencies) != 1 {
		t.Fatalf("unexpected comparison: %+v", got)
	}
}
