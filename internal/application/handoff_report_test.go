package application_test

import (
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/stretchr/testify/require"
)

func TestReportHandoffMetricsReadsBothSourcesBeforeComputing(t *testing.T) {
	t.Parallel()

	var order []string
	report, err := application.ReportHandoffMetrics("root", application.HandoffMetricsReportPorts{
		ReadChallenges: func(string) ([]application.HandoffChallengeMetric, error) {
			order = append(order, "challenges")
			return []application.HandoffChallengeMetric{{MissionID: "m-1"}}, nil
		},
		ReadLabels: func(string) ([]application.HandoffApplicationGroundTruth, error) {
			order = append(order, "labels")
			return []application.HandoffApplicationGroundTruth{{Label: "applied"}}, nil
		},
		Compute: func(records []application.HandoffChallengeMetric, labels []application.HandoffApplicationGroundTruth) application.HandoffMetricsReport {
			order = append(order, "compute")
			return application.HandoffMetricsReport{SampleSize: len(records), ApplicationSampleSize: len(labels)}
		},
	})

	require.NoError(t, err)
	require.Equal(t, []string{"challenges", "labels", "compute"}, order)
	require.Equal(t, 1, report.SampleSize)
	require.Equal(t, 1, report.ApplicationSampleSize)
}

func TestReportHandoffMetricsPropagatesReadErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("history unavailable")
	_, err := application.ReportHandoffMetrics("root", application.HandoffMetricsReportPorts{
		ReadChallenges: func(string) ([]application.HandoffChallengeMetric, error) { return nil, wantErr },
		ReadLabels:     func(string) ([]application.HandoffApplicationGroundTruth, error) { return nil, nil },
		Compute: func([]application.HandoffChallengeMetric, []application.HandoffApplicationGroundTruth) application.HandoffMetricsReport {
			return application.HandoffMetricsReport{}
		},
	})

	require.ErrorIs(t, err, wantErr)
}

func TestReportHandoffMetricsRejectsMissingPort(t *testing.T) {
	t.Parallel()

	_, err := application.ReportHandoffMetrics("root", application.HandoffMetricsReportPorts{})
	require.EqualError(t, err, "handoff metrics report ports are required")
}
