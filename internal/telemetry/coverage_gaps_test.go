package telemetry

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfidenceProducerAdapter_Gaps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	historyPath := filepath.Join(dir, ConfidenceHistoryRelPath)

	// Unsupported agent
	_, err := NewConfidenceProducerAdapter(historyPath, "unsupported_agent", "m-1")
	require.ErrorContains(t, err, "unsupported agent")

	// Empty mission_id
	_, err = NewConfidenceProducerAdapter(historyPath, "ranger", "")
	require.ErrorContains(t, err, "mission_id is required")

	// Valid adapter + WithRun
	adapter, err := NewConfidenceProducerAdapter(historyPath, "ranger", "m-1")
	require.NoError(t, err)

	scoped := adapter.WithRun("run-42")
	assert.Equal(t, "run-42", scoped.Run)

	// Record missing
	err = scoped.RecordMissing("key-1", "no_claim")
	require.NoError(t, err)

	// Record claim
	claim := domain.ConfidenceClaim{
		ID:                "C-1",
		Statement:         "test claim",
		ClaimKind:         domain.ClaimKindAssertion,
		ConfidencePercent: 95,
		CorrelationKey:    "key-1",
		EvidenceIDs:       []string{"E-1"},
	}
	evidence := []domain.Evidence{
		{
			ID:         "E-1",
			SourceRef:  "docs/test.md",
			Class:      domain.EvidenceClassExplicit,
			Confidence: domain.ConfidenceHigh,
		},
	}
	rec, err := scoped.RecordClaim(claim, evidence)
	require.NoError(t, err)
	assert.Equal(t, "ranger", rec.Agent)
	assert.Equal(t, "run-42", rec.Run)
}

func TestDiscovery_Gaps(t *testing.T) {
	t.Parallel()

	event := NewDiscoveryWeaponEvent("run-1", "prov-1", "/path/to/art", DiscoveryInvocationFailed, DiscoveryNormalizationRejected, "ev", "reas", "auth", "pin", "iso")
	assert.Equal(t, SeverityError, event.SeverityNumber)

	// WithDiscoveryAdapterProvenance empty vs populated
	unchanged := WithDiscoveryAdapterProvenance(event, "", "")
	assert.Equal(t, event, unchanged)

	withAdapter := WithDiscoveryAdapterProvenance(event, "adapter-1", "child-policy-1")
	assert.Equal(t, "adapter-1", withAdapter.Attributes[AttrExecutionAdapter])
	assert.Equal(t, "child-policy-1", withAdapter.Attributes[AttrChildPolicyID])

	// WithDiscoveryRequestCorrelation
	withReq := WithDiscoveryRequestCorrelation(event, "req-123")
	assert.Equal(t, "req-123", withReq.Attributes[AttrInvocationRequestID])
}

func TestInitiativeEventHistoryPath(t *testing.T) {
	t.Parallel()

	path := InitiativeEventHistoryPath("/tmp/strategist")
	assert.Equal(t, filepath.Join("/tmp/strategist", "memory", "initiative-events.jsonl"), path)
}

func TestRefinementHandoffLine_ValidationAndRead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := HandoffMetricsPath(dir)
	assert.Equal(t, filepath.Join(dir, "memory", "handoff-metrics.jsonl"), path)

	// Validation errors
	line := RefinementHandoffLine{}
	require.ErrorContains(t, line.Validate(), "mission_id is required")

	neg := int64(-1)
	line.MissionID = "m-1"
	line.DiscoveryTokens = &neg
	require.ErrorContains(t, line.Validate(), "discovery_tokens must be >= 0")

	line.DiscoveryTokens = nil
	line.BriefTokens = &neg
	require.ErrorContains(t, line.Validate(), "brief_tokens must be >= 0")

	negInt := -1
	line.BriefTokens = nil
	line.RefinementReopens = &negInt
	require.ErrorContains(t, line.Validate(), "refinement_reopens must be >= 0")

	line.RefinementReopens = nil
	line.Revision = &negInt
	require.ErrorContains(t, line.Validate(), "revision must be >= 1")

	negFloat := -0.5
	line.Revision = nil
	line.BriefCompressionRatio = &negFloat
	require.ErrorContains(t, line.Validate(), "brief_compression_ratio must be >= 0")

	line.BriefCompressionRatio = nil
	line.EvidenceCoverageRatio = &negFloat
	require.ErrorContains(t, line.Validate(), "evidence_coverage_ratio must be >= 0")

	// Append & Read lines
	lineValid := RefinementHandoffLine{
		MissionID: "m-1",
	}
	appended, err := AppendRefinementHandoffLine(path, lineValid)
	require.NoError(t, err)
	assert.True(t, appended)

	// Append duplicate base line
	appended, err = AppendRefinementHandoffLine(path, lineValid)
	require.NoError(t, err)
	assert.False(t, appended)

	// Read lines
	lines, err := ReadRefinementHandoffLines(path)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	assert.Equal(t, "m-1", lines[0].MissionID)

	// Read non-existent file
	none, err := ReadRefinementHandoffLines(filepath.Join(dir, "nonexistent.jsonl"))
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestTokenLedgerComparison_Contradictory(t *testing.T) {
	t.Parallel()

	// Contradictory reported mission usage
	rec1 := MissionTokenUsageRecord{MissionID: "m-1", TokensIn: 100, TokensOut: 50}
	rec2 := MissionTokenUsageRecord{MissionID: "m-1", TokensIn: 200, TokensOut: 50}
	comp := CompareTokenLedgers([]MissionTokenUsageRecord{rec1, rec2}, nil)
	assert.Equal(t, TokenLedgerInconsistent, comp.Status)
	require.Len(t, comp.Inconsistencies, 1)

	// Contradictory handoff ledger
	tok1 := int64(100)
	tok2 := int64(200)
	line1 := RefinementHandoffLine{MissionID: "m-1", DiscoveryTokens: &tok1}
	line2 := RefinementHandoffLine{MissionID: "m-1", DiscoveryTokens: &tok2}
	comp = CompareTokenLedgers(nil, []RefinementHandoffLine{line1, line2})
	assert.Equal(t, TokenLedgerInconsistent, comp.Status)
	require.Len(t, comp.Inconsistencies, 1)
}

func TestGroundTruthMetrics_BoolToInt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, boolToInt(true))
	assert.Equal(t, 0, boolToInt(false))
}
