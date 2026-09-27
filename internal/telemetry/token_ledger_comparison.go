package telemetry

import "strconv"

// TokenLedgerComparison is a read-only reconciliation of the two token
// authorities. MissionTokenUsageRecord is provider-reported mission usage;
// RefinementHandoffLine is handoff-local phase evidence. Their quantities are
// intentionally not added or converted into one another.
type TokenLedgerComparison struct {
	Status              string   `json:"status"`
	ReportedRecordCount int      `json:"reported_record_count"`
	HandoffRecordCount  int      `json:"handoff_record_count"`
	Inconsistencies     []string `json:"inconsistencies,omitempty"`
}

const (
	// TokenLedgerNoData indicates that neither ledger supplied records.
	TokenLedgerNoData = "no_data"
	// TokenLedgerReportedOnly indicates that only provider-reported usage exists.
	TokenLedgerReportedOnly = "reported_only"
	// TokenLedgerHandoffOnly indicates that only handoff-local evidence exists.
	TokenLedgerHandoffOnly = "handoff_only"
	// TokenLedgerNotComparable indicates that both ledgers exist but differ in authority.
	TokenLedgerNotComparable = "not_comparable"
	// TokenLedgerInconsistent indicates contradictory records within a ledger.
	TokenLedgerInconsistent = "inconsistent"
)

// CompareTokenLedgers exposes provenance and contradictions without deriving
// totals, budget ceilings, or phase values. A reported mission total and a
// handoff-local estimate are not the same quantity, so a healthy pair is
// explicitly not_comparable rather than falsely marked equal.
func CompareTokenLedgers(reported []MissionTokenUsageRecord, handoff []RefinementHandoffLine) TokenLedgerComparison {
	result := TokenLedgerComparison{ReportedRecordCount: len(reported), HandoffRecordCount: len(handoff)}
	result.Inconsistencies = append(result.Inconsistencies, reportedInconsistencies(reported)...)
	result.Inconsistencies = append(result.Inconsistencies, handoffInconsistencies(handoff)...)
	if len(result.Inconsistencies) > 0 {
		result.Status = TokenLedgerInconsistent
		return result
	}
	switch {
	case len(reported) == 0 && len(handoff) == 0:
		result.Status = TokenLedgerNoData
	case len(reported) == 0:
		result.Status = TokenLedgerHandoffOnly
	case len(handoff) == 0:
		result.Status = TokenLedgerReportedOnly
	default:
		result.Status = TokenLedgerNotComparable
	}
	return result
}

func reportedInconsistencies(records []MissionTokenUsageRecord) []string {
	if len(records) < 2 {
		return nil
	}
	first := records[0]
	for _, record := range records[1:] {
		if record.TokensIn != first.TokensIn || record.TokensOut != first.TokensOut {
			return []string{"reported mission usage contains contradictory totals"}
		}
	}
	return nil
}

func handoffInconsistencies(lines []RefinementHandoffLine) []string {
	seen := map[string]RefinementHandoffLine{}
	var out []string
	for _, line := range lines {
		key := revisionKey(line.Revision)
		previous, ok := seen[key]
		if ok && !sameHandoffTokens(previous, line) {
			out = append(out, "handoff ledger contains contradictory estimates for "+key)
			continue
		}
		seen[key] = line
	}
	return out
}

func revisionKey(revision *int) string {
	if revision == nil {
		return "base"
	}
	return "revision:" + strconv.Itoa(*revision)
}

func sameHandoffTokens(a, b RefinementHandoffLine) bool {
	return equalInt64Pointer(a.DiscoveryTokens, b.DiscoveryTokens) && equalInt64Pointer(a.BriefTokens, b.BriefTokens)
}

func equalInt64Pointer(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
