package application_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/stretchr/testify/require"
)

func TestFilterMissionTokenUsageKeepsMissionOrder(t *testing.T) {
	t.Parallel()

	got := application.FilterMissionTokenUsage([]application.MissionTokenUsageRecord{
		{MissionID: "other", TokensIn: 1}, {MissionID: "m-1", TokensIn: 2}, {MissionID: "m-1", TokensIn: 3},
	}, "m-1")

	require.Equal(t, []application.MissionTokenUsageRecord{{MissionID: "m-1", TokensIn: 2}, {MissionID: "m-1", TokensIn: 3}}, got)
}

func TestFilterHandoffMetricsPreservesUnmeasuredFields(t *testing.T) {
	t.Parallel()

	zero := int64(0)
	got := application.FilterHandoffMetrics([]application.HandoffMetricsLine{
		{MissionID: "other"}, {MissionID: "m-1", DiscoveryTokens: &zero},
	}, "m-1")

	require.Len(t, got, 1)
	require.NotNil(t, got[0].DiscoveryTokens)
	require.Equal(t, int64(0), *got[0].DiscoveryTokens)
}
