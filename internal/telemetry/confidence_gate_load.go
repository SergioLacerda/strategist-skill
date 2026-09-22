package telemetry

import "fmt"

// LoadConfidenceGateReview is the single materializer of the advisory gate
// review: the gate step and `strategist metrics confidence` both call it, so
// they cannot diverge. A non-empty missionID scopes records to that mission;
// read diagnostics stay global because malformed lines cannot be attributed.
func LoadConfidenceGateReview(strategistRoot, missionID string) (ConfidenceGateReview, error) {
	return LoadConfidenceGateReviewForRun(strategistRoot, missionID, "")
}

// LoadConfidenceGateReviewForRun materializes the advisory review for one
// mission and, when supplied, one explicit run. Empty run preserves the legacy
// mission-wide projection and never assigns legacy records to a run.
func LoadConfidenceGateReviewForRun(strategistRoot, missionID, run string) (ConfidenceGateReview, error) {
	records, diagnostics, err := ReadConfidenceRecordsWithDiagnostics(ConfidenceHistoryPath(strategistRoot))
	if err != nil {
		return ConfidenceGateReview{}, fmt.Errorf("load confidence gate review: %w", err)
	}
	return BuildConfidenceGateReview(filterConfidenceRecords(records, missionID, run), diagnostics), nil
}

func filterConfidenceRecords(records []ConfidenceRecord, missionID, run string) []ConfidenceRecord {
	if missionID == "" {
		return records
	}
	scoped := make([]ConfidenceRecord, 0, len(records))
	for _, record := range records {
		if record.MissionID == missionID && (run == "" || record.Run == run) {
			scoped = append(scoped, record)
		}
	}
	return scoped
}
