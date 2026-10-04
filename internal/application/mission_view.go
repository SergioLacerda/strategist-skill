package application

// FilterMissionTokenUsage returns only usage reports for missionID, keeping
// the source order and preserving the original records unchanged.
func FilterMissionTokenUsage(records []MissionTokenUsageRecord, missionID string) []MissionTokenUsageRecord {
	filtered := make([]MissionTokenUsageRecord, 0, len(records))
	for _, record := range records {
		if record.MissionID == missionID {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

// FilterHandoffMetrics returns only handoff metric lines for missionID, in
// ledger order. Nil pointer measurements remain nil and are not derived.
func FilterHandoffMetrics(lines []HandoffMetricsLine, missionID string) []HandoffMetricsLine {
	filtered := make([]HandoffMetricsLine, 0, len(lines))
	for _, line := range lines {
		if line.MissionID == missionID {
			filtered = append(filtered, line)
		}
	}
	return filtered
}
