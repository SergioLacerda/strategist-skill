package application_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/stretchr/testify/require"
)

func TestHandoffMetricsPreservesExplicitZeroMeasurements(t *testing.T) {
	t.Parallel()

	zero := int64(0)
	line := application.HandoffMetricsLine{MissionID: "m-1", DiscoveryTokens: &zero}
	require.NotNil(t, line.DiscoveryTokens)
	require.Equal(t, int64(0), *line.DiscoveryTokens)
}
