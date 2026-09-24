package leveling

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/initiative"
	"github.com/stretchr/testify/require"
)

func TestSignalsFromPreciseShotMapsClosedSignals(t *testing.T) {
	request := initiative.EscalationRequest{
		Mechanism: initiative.PreciseShotMechanism, AssessmentID: "assessment",
		IdempotencyKey: "idempotency", Effective: initiative.ConfidenceLow,
		Signals: initiative.EscalationSignals{SecuritySensitive: true, Risk: "high"},
	}
	signals, err := SignalsFromEscalation(request)
	require.NoError(t, err)
	require.True(t, signals.SecuritySensitive)
	require.Equal(t, "high", signals.Risk)
	require.Equal(t, "insufficient", signals.Evidence)
}

func TestSignalsFromPreciseShotRejectsUnknownMechanism(t *testing.T) {
	_, err := SignalsFromEscalation(initiative.EscalationRequest{Mechanism: "other"})
	require.ErrorContains(t, err, "unsupported mechanism")
}

func TestSignalsFromPreciseShotPreservesTypedMappings(t *testing.T) {
	cases := []struct {
		name  string
		in    initiative.EscalationSignals
		check func(*testing.T, Signals)
	}{
		{"conflict", initiative.EscalationSignals{Evidence: "conflicting", ConflictingEvidence: true}, func(t *testing.T, got Signals) {
			require.True(t, got.ConflictingEvidence)
			require.Equal(t, "conflicting", got.Evidence)
		}},
		{"scope", initiative.EscalationSignals{Scope: "cross_module", ArchitecturalChange: true}, func(t *testing.T, got Signals) {
			require.True(t, got.ArchitecturalChange)
			require.Equal(t, "cross_module", got.Scope)
		}},
		{"repeated failure", initiative.EscalationSignals{RepeatedFailures: 2}, func(t *testing.T, got Signals) { require.Equal(t, 2, got.RepeatedFailures) }},
		{"low diligence", initiative.EscalationSignals{Evidence: "insufficient"}, func(t *testing.T, got Signals) { require.Equal(t, "insufficient", got.Evidence) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SignalsFromEscalation(initiative.EscalationRequest{
				Mechanism: initiative.PreciseShotMechanism, AssessmentID: "assessment", IdempotencyKey: tc.name,
				Effective: initiative.ConfidenceLow, Signals: tc.in,
			})
			require.NoError(t, err)
			tc.check(t, got)
		})
	}
}
