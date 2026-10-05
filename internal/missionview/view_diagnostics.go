package missionview

// timelineDiagnostics reports the unreadable lifecycle and usage ledgers.
func timelineDiagnostics(lifecycleErr, usageErr error) []Diagnostic {
	var out []Diagnostic
	if lifecycleErr != nil {
		out = append(out, Diagnostic{Reason: ReasonMissionViewLifecycleUnavailable, Action: "inspect mission-lifecycle ledger"})
	}
	if usageErr != nil {
		out = append(out, Diagnostic{Reason: ReasonMissionViewUsageUnavailable, Action: "inspect mission-usage ledger"})
	}
	return out
}
