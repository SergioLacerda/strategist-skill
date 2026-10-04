package metrics

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func handoffMetricsReportPorts() application.HandoffMetricsReportPorts {
	return application.HandoffMetricsReportPorts{
		ReadChallenges: readHandoffChallengeMetrics,
		ReadLabels:     readHandoffApplicationLabels,
		Compute:        computeHandoffMetricsReport,
	}
}

func readHandoffChallengeMetrics(root string) ([]application.HandoffChallengeMetric, error) {
	records, err := telemetry.ReadHandoffChallenges(telemetry.HandoffChallengeHistoryPath(root))
	if err != nil {
		return nil, fmt.Errorf("read handoff challenges: %w", err)
	}
	projected := make([]application.HandoffChallengeMetric, 0, len(records))
	for _, record := range records {
		projected = append(projected, application.HandoffChallengeMetric{
			MissionID:         record.MissionID,
			Attempt:           record.Attempt,
			Passed:            record.Passed,
			MissingRefs:       record.MissingRefs,
			MissingChallenges: record.MissingChallenges,
			MisclassifiedRefs: record.MisclassifiedRefs,
		})
	}
	return projected, nil
}

func readHandoffApplicationLabels(root string) ([]application.HandoffApplicationGroundTruth, error) {
	labels, err := telemetry.ReadGroundTruthLabels(telemetry.GroundTruthLabelHistoryPath(root), telemetry.GroundTruthSubjectHandoffApplication)
	if err != nil {
		return nil, fmt.Errorf("read handoff application labels: %w", err)
	}
	projected := make([]application.HandoffApplicationGroundTruth, 0, len(labels))
	for _, label := range labels {
		projected = append(projected, application.HandoffApplicationGroundTruth{Label: label.Label})
	}
	return projected, nil
}

func computeHandoffMetricsReport(records []application.HandoffChallengeMetric, labels []application.HandoffApplicationGroundTruth) application.HandoffMetricsReport {
	telemetryRecords := make([]telemetry.ChallengeRecord, 0, len(records))
	for _, record := range records {
		telemetryRecords = append(telemetryRecords, telemetry.ChallengeRecord{
			MissionID: record.MissionID, Attempt: record.Attempt, Passed: record.Passed,
			MissingRefs: record.MissingRefs, MissingChallenges: record.MissingChallenges,
			MisclassifiedRefs: record.MisclassifiedRefs,
		})
	}
	telemetryLabels := make([]telemetry.GroundTruthLabel, 0, len(labels))
	for _, label := range labels {
		telemetryLabels = append(telemetryLabels, telemetry.GroundTruthLabel{
			Subject: telemetry.GroundTruthSubjectHandoffApplication, Label: label.Label,
		})
	}
	metrics := telemetry.ApplyHandoffApplicationGroundTruth(telemetry.ComputeHandoffMetrics(telemetryRecords), telemetryLabels)
	return application.HandoffMetricsReport{
		HandoffPassRate: metrics.HandoffPassRate, FirstAttemptPassRate: metrics.FirstAttemptPassRate,
		CriticalConstraintRecall: metrics.CriticalConstraintRecall, DecisionClassificationAccuracy: metrics.DecisionClassificationAccuracy,
		ScopeViolationRate: metrics.ScopeViolationRate, HandoffRepairRate: metrics.HandoffRepairRate,
		SemanticLoss: application.HandoffSemanticLoss{
			Recall: metrics.SemanticLoss.Recall, Classification: metrics.SemanticLoss.Classification, Application: metrics.SemanticLoss.Application,
		},
		SampleSize: metrics.SampleSize, ApplicationSampleSize: metrics.ApplicationSampleSize,
	}
}
