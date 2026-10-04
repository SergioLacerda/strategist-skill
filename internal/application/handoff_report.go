package application

import "errors"

// HandoffChallengeMetric is the adapter-neutral projection required by the
// handoff metrics aggregator.
type HandoffChallengeMetric struct {
	MissionID         string
	Attempt           int
	Passed            bool
	MissingRefs       []string
	MissingChallenges []string
	MisclassifiedRefs []string
}

// HandoffApplicationGroundTruth is the reviewed downstream label used to
// measure whether acknowledged constraints were applied.
type HandoffApplicationGroundTruth struct {
	Label string
}

// HandoffMetricsReport is the application DTO returned by the reporting
// service. Persistence and telemetry-specific record types stay at the edge.
type HandoffMetricsReport struct {
	HandoffPassRate                float64
	FirstAttemptPassRate           float64
	CriticalConstraintRecall       float64
	DecisionClassificationAccuracy float64
	ScopeViolationRate             float64
	HandoffRepairRate              float64
	SemanticLoss                   HandoffSemanticLoss
	SampleSize                     int
	ApplicationSampleSize          int
}

// HandoffSemanticLoss contains the semantic-loss components of the report.
type HandoffSemanticLoss struct {
	Recall         float64
	Classification float64
	Application    float64
}

// HandoffMetricsReportPorts keep file and telemetry adapters outside the
// application service while allowing the existing aggregator to remain the
// source of metric definitions during migration.
type HandoffMetricsReportPorts struct {
	ReadChallenges func(root string) ([]HandoffChallengeMetric, error)
	ReadLabels     func(root string) ([]HandoffApplicationGroundTruth, error)
	Compute        func([]HandoffChallengeMetric, []HandoffApplicationGroundTruth) HandoffMetricsReport
}

// ReportHandoffMetrics reads the required histories in application order and
// delegates persistence/aggregation details through explicit ports.
func ReportHandoffMetrics(root string, ports HandoffMetricsReportPorts) (HandoffMetricsReport, error) {
	if ports.ReadChallenges == nil || ports.ReadLabels == nil || ports.Compute == nil {
		return HandoffMetricsReport{}, errors.New("handoff metrics report ports are required")
	}
	records, err := ports.ReadChallenges(root)
	if err != nil {
		return HandoffMetricsReport{}, err
	}
	labels, err := ports.ReadLabels(root)
	if err != nil {
		return HandoffMetricsReport{}, err
	}
	return ports.Compute(records, labels), nil
}
