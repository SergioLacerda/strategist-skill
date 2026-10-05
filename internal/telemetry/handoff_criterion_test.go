package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTheArchivistToSniperEventCarriesBoundedFactsOnly(t *testing.T) {
	event := NewArchivistToSniperEvent("run-1", "passed", true, 2, "passed", 0, "")

	require.Equal(t, ArchivistToSniperEventName, event.Name)
	require.Equal(t, ArchivistToSniperContractID, event.Attributes[AttrEventContractID])
	require.Equal(t, "archivist_to_sniper", event.Attributes[AttrTransitionGroup])
	require.Equal(t, "passed", event.Attributes[AttrHandoffOutcome])
	require.Equal(t, 2, event.Attributes[AttrHandoffAttempt])
	for key := range event.Attributes {
		require.NotContains(t, key, "artifact", "no artifact content or digest is ever an attribute")
		require.NotContains(t, key, "answer")
		require.NotContains(t, key, "digest")
	}
}

func TestAnEventWithoutAProviderNamesTheMainPathAsItsCriterion(t *testing.T) {
	event := WithHandoffCriterion(NewArchivistToSniperEvent("r", "passed", true, 1, "passed", 0, ""), HandoffCriterion{})
	require.Equal(t, CriterionSourceMain, event.Attributes[AttrHandoffCriterionSource])
	require.NotContains(t, event.Attributes, AttrHandoffProviderConfidencePercent)
}

func TestAProviderCriterionCarriesConfidenceThresholdProviderAndModel(t *testing.T) {
	event := WithHandoffCriterion(NewRangerToArchivistEvent("r", "passed", true, 1, "passed", 0, ""), HandoffCriterion{
		Provider: "jev", Model: "jev-1.13.0", ConfidencePercent: 94, ThresholdPercent: 90, Approved: true,
	})
	require.Equal(t, CriterionSourceProvider, event.Attributes[AttrHandoffCriterionSource])
	require.Equal(t, "jev", event.Attributes[AttrHandoffProvider])
	require.Equal(t, "jev-1.13.0", event.Attributes[AttrHandoffProviderModel])
	require.Equal(t, 94, event.Attributes[AttrHandoffProviderConfidencePercent])
	require.Equal(t, 90, event.Attributes[AttrHandoffProviderThresholdPercent])
	require.Equal(t, true, event.Attributes[AttrHandoffProviderApproved])
}
