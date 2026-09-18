package conformance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validMatrix() Matrix {
	return Matrix{
		SchemaVersion: "matrix/v1",
		Clients: []Client{{
			ID: "codex", Owner: "codex", Surface: ".codex",
			Roles: []string{"ranger"}, ProviderModes: []string{"custom"},
			UnsupportedPolicy: "blocked",
		}},
		Rows: []Row{{
			ID: "codex-ranger", Client: "codex", Role: "ranger", Slot: "discovery",
			ProviderMode: "custom", EnvelopeVersion: "envelope/v1",
			EvidenceTier: EvidenceStructural, ExpectedState: StateCertified,
			ReasonCode: "verified", AuthorityOwner: "domain",
		}},
	}
}

func TestMatrixValidateRejectsIncompleteRowsAndUnknownClients(t *testing.T) {
	matrix := validMatrix()
	matrix.Rows[0].ReasonCode = ""
	require.ErrorContains(t, matrix.Validate(), "row")

	matrix = validMatrix()
	matrix.Rows[0].Client = "unknown"
	require.ErrorContains(t, matrix.Validate(), "unclassified client")
}

func TestMatrixEvaluateAndLiveProbeClassifyEvidence(t *testing.T) {
	matrix := validMatrix()
	report, err := matrix.Evaluate(map[string]bool{"codex": true})
	require.NoError(t, err)
	require.Len(t, report.Results, 1)
	require.True(t, report.Results[0].Passed)

	live := matrix.Rows[0]
	live.EvidenceTier = EvidenceLive
	result, err := EvaluateLiveProbe(live, StateUnknown)
	require.NoError(t, err)
	require.False(t, result.Passed)
	result, err = EvaluateLiveProbe(live, StateCertified)
	require.NoError(t, err)
	require.True(t, result.Passed)
}
