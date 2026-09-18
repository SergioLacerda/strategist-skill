package conformance_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/plugins/conformance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvocationProvenanceValidatesRuntimeEvidenceAndFingerprintsDeterministically(t *testing.T) {
	input := validInvocationProvenance()

	first, err := conformance.NewInvocationProvenance(input)
	require.NoError(t, err)
	second, err := conformance.NewInvocationProvenance(input)
	require.NoError(t, err)

	assert.Equal(t, first.Fingerprint, second.Fingerprint)
	assert.NotEmpty(t, first.Fingerprint)
	assert.NoError(t, first.Validate())
}

func TestInvocationProvenanceRejectsSuccessfulStaticEvidence(t *testing.T) {
	input := validInvocationProvenance()
	input.Status = conformance.InvocationStatusInvoked
	input.RuntimeEvidence = false

	_, err := conformance.NewInvocationProvenance(input)
	require.Error(t, err)
	assert.ErrorContains(t, err, "runtime evidence")
}

func TestInvocationProvenanceRejectsSilentFallback(t *testing.T) {
	input := validInvocationProvenance()
	input.FallbackDecision = "native"

	_, err := conformance.NewInvocationProvenance(input)
	require.Error(t, err)
	assert.ErrorContains(t, err, "fallback")
}

func TestInvocationProvenanceRejectsInvalidInvokedEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*conformance.InvocationProvenanceInput)
		want   string
	}{
		{
			name: "schema result",
			mutate: func(input *conformance.InvocationProvenanceInput) {
				input.SchemaResult = "rejected"
			},
			want: "accepted schema result",
		},
		{
			name: "failure reason",
			mutate: func(input *conformance.InvocationProvenanceInput) {
				input.FailureReasonCode = "provider_error"
			},
			want: "failure reason",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validInvocationProvenance()
			test.mutate(&input)
			_, err := conformance.NewInvocationProvenance(input)
			require.Error(t, err)
			assert.ErrorContains(t, err, test.want)
		})
	}
}

func TestInvocationProvenanceAcceptsNonInvokedStatusesWithFailureEvidence(t *testing.T) {
	for _, status := range []conformance.InvocationStatus{
		conformance.InvocationStatusFailed,
		conformance.InvocationStatusBlocked,
		conformance.InvocationStatusUnsupported,
		conformance.InvocationStatusUnknown,
	} {
		t.Run(string(status), func(t *testing.T) {
			input := validInvocationProvenance()
			input.Status = status
			input.RuntimeEvidence = false
			input.FailureReasonCode = "provider_unavailable"

			provenance, err := conformance.NewInvocationProvenance(input)
			require.NoError(t, err)
			assert.NoError(t, provenance.Validate())
		})
	}
}

func TestInvocationProvenanceRejectsInvalidNonInvokedEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*conformance.InvocationProvenanceInput)
		want   string
	}{
		{
			name: "runtime evidence",
			mutate: func(input *conformance.InvocationProvenanceInput) {
				input.Status = conformance.InvocationStatusFailed
				input.RuntimeEvidence = true
				input.FailureReasonCode = "provider_error"
			},
			want: "cannot claim runtime evidence",
		},
		{
			name: "missing failure reason",
			mutate: func(input *conformance.InvocationProvenanceInput) {
				input.Status = conformance.InvocationStatusBlocked
				input.RuntimeEvidence = false
				input.FailureReasonCode = ""
			},
			want: "requires a failure reason",
		},
		{
			name: "unknown status",
			mutate: func(input *conformance.InvocationProvenanceInput) {
				input.Status = "not-a-status"
			},
			want: "unknown status",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validInvocationProvenance()
			test.mutate(&input)
			_, err := conformance.NewInvocationProvenance(input)
			require.Error(t, err)
			assert.ErrorContains(t, err, test.want)
		})
	}
}

func TestInvocationProvenanceRequiresAllIdentityFields(t *testing.T) {
	fields := []struct {
		name   string
		mutate func(*conformance.InvocationProvenanceInput)
	}{
		{"mission_id", func(input *conformance.InvocationProvenanceInput) { input.MissionID = "" }},
		{"role", func(input *conformance.InvocationProvenanceInput) { input.Role = "" }},
		{"slot", func(input *conformance.InvocationProvenanceInput) { input.Slot = "" }},
		{"provider", func(input *conformance.InvocationProvenanceInput) { input.Provider = "" }},
		{"mode", func(input *conformance.InvocationProvenanceInput) { input.Mode = "" }},
		{"binding_digest", func(input *conformance.InvocationProvenanceInput) { input.BindingDigest = "" }},
		{"envelope_digest", func(input *conformance.InvocationProvenanceInput) { input.EnvelopeDigest = "" }},
		{"schema_result", func(input *conformance.InvocationProvenanceInput) { input.SchemaResult = "" }},
		{"fallback_decision", func(input *conformance.InvocationProvenanceInput) { input.FallbackDecision = "" }},
	}
	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			input := validInvocationProvenance()
			field.mutate(&input)
			_, err := conformance.NewInvocationProvenance(input)
			require.Error(t, err)
			assert.ErrorContains(t, err, field.name+" is required")
		})
	}
}

func TestInvocationProvenanceEventRejectsInvalidEvidence(t *testing.T) {
	provenance := conformance.InvocationProvenance{Status: conformance.InvocationStatusInvoked}
	_, err := conformance.InvocationProvenanceEvent(provenance)
	require.Error(t, err)
	assert.ErrorContains(t, err, "is required")
}

func TestInvocationProvenanceEventCarriesEvidenceWithoutChangingAuthority(t *testing.T) {
	provenance, err := conformance.NewInvocationProvenance(validInvocationProvenance())
	require.NoError(t, err)

	event, err := conformance.InvocationProvenanceEvent(provenance)
	require.NoError(t, err)
	require.NoError(t, event.Validate())
	assert.Equal(t, "strategist.invocation.invoked", event.Name)
	assert.Equal(t, provenance.Fingerprint, event.Attributes["strategist.invocation.fingerprint"])
	assert.Equal(t, true, event.Attributes["strategist.invocation.runtime_evidence"])
}

func validInvocationProvenance() conformance.InvocationProvenanceInput {
	return conformance.InvocationProvenanceInput{
		MissionID:         "mission-1",
		Role:              "ranger",
		Slot:              "discovery",
		Provider:          "brainstorming",
		Mode:              "ranked",
		BindingDigest:     "sha256:binding",
		EnvelopeDigest:    "sha256:envelope",
		SchemaResult:      "accepted",
		Status:            conformance.InvocationStatusInvoked,
		RuntimeEvidence:   true,
		FallbackDecision:  "none",
		FailureReasonCode: "",
	}
}
