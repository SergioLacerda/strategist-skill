package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsageQualityRankOrdersFromStrongestToWeakest(t *testing.T) {
	order := []UsageQuality{
		UsageProviderMeasured, UsageHostMeasured, UsageReportedUnverified,
		UsageTokenizerEstimated, UsageHeuristicEstimated, UsageUnavailable,
	}
	for i := 0; i < len(order)-1; i++ {
		assert.Greater(t, UsageQualityRank(order[i]), UsageQualityRank(order[i+1]), "%s must outrank %s", order[i], order[i+1])
	}
	assert.Equal(t, -1, UsageQualityRank("made_up"))
}

func TestUsageQualityAllowedForSourceNeverExceedsTheSource(t *testing.T) {
	cases := []struct {
		source  UsageSource
		quality UsageQuality
		allowed bool
	}{
		{UsageSourceAgentReport, UsageReportedUnverified, true},
		{UsageSourceAgentReport, UsageHeuristicEstimated, true},
		{UsageSourceAgentReport, UsageProviderMeasured, false},
		{UsageSourceAgentReport, UsageHostMeasured, false},
		{UsageSourceEstimator, UsageTokenizerEstimated, true},
		{UsageSourceEstimator, UsageHeuristicEstimated, true},
		{UsageSourceEstimator, UsageReportedUnverified, false},
		{UsageSourceHostCounter, UsageHostMeasured, true},
		{UsageSourceHostCounter, UsageProviderMeasured, false},
		{UsageSourceProviderResponse, UsageProviderMeasured, true},
		{"unknown_source", UsageUnavailable, false},
	}
	for _, c := range cases {
		assert.Equal(t, c.allowed, UsageQualityAllowedForSource(c.source, c.quality), "%s/%s", c.source, c.quality)
	}
}

func TestDefaultUsageQuality(t *testing.T) {
	assert.Equal(t, UsageReportedUnverified, DefaultUsageQuality(UsageSourceAgentReport))
	assert.Equal(t, UsageHeuristicEstimated, DefaultUsageQuality(UsageSourceEstimator))
	assert.Equal(t, UsageHostMeasured, DefaultUsageQuality(UsageSourceHostCounter))
	assert.Equal(t, UsageProviderMeasured, DefaultUsageQuality(UsageSourceProviderResponse))
	assert.Equal(t, UsageUnavailable, DefaultUsageQuality("nope"))
}

func TestUsagePhaseVocabulary(t *testing.T) {
	for _, p := range []string{"BOOTSTRAP", "INTAKE", "DISCOVERY", "REFINEMENT", "EXECUTION", "UNATTRIBUTED"} {
		assert.True(t, ValidUsagePhase(p), p)
	}
	for _, p := range []string{"APPROVAL_GATE", "DONE", "BLOCKED", "delivery", ""} {
		assert.False(t, ValidUsagePhase(p), p)
	}
}

func TestParseUsagePhaseAcceptsLowercaseAliases(t *testing.T) {
	cases := map[string]string{
		"discovery": "DISCOVERY", "Refinement": "REFINEMENT", "refining": "REFINEMENT",
		"execution": "EXECUTION", "delivery": "EXECUTION", "intake": "INTAKE",
		"bootstrap": "BOOTSTRAP", "unattributed": "UNATTRIBUTED", "DISCOVERY": "DISCOVERY",
	}
	for in, want := range cases {
		got, ok := ParseUsagePhase(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, bad := range []string{"gate", "approval_gate", "done", ""} {
		_, ok := ParseUsagePhase(bad)
		assert.False(t, ok, bad)
	}
}

func TestUsageRecordKinds(t *testing.T) {
	assert.True(t, ValidUsageRecordKind(string(UsageKindIncremental)))
	assert.True(t, ValidUsageRecordKind(string(UsageKindCumulativeSnapshot)))
	assert.False(t, ValidUsageRecordKind("running_total"))
}
