package handoff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredByRisk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   RiskSignals
		want bool
	}{
		{name: "skip informational only", in: RiskSignals{InformationalOnly: true}, want: false},
		{name: "approval gate requires challenge", in: RiskSignals{ApprovalGatePresent: true}, want: true},
		{name: "mandatory constraints require challenge", in: RiskSignals{MandatoryConstraintsPresent: true}, want: true},
		{name: "unresolved questions require challenge", in: RiskSignals{UnresolvedQuestionsPresent: true}, want: true},
		{name: "forbidden scope requires challenge", in: RiskSignals{ForbiddenScopePresent: true}, want: true},
		{name: "implementation handoff requires challenge", in: RiskSignals{ImplementationHandoffPresent: true}, want: true},
		{name: "destructive operation requires challenge", in: RiskSignals{DestructiveOperationPossible: true}, want: true},
		{name: "security task requires challenge", in: RiskSignals{SecuritySensitiveTask: true}, want: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, RequiredByRisk(tt.in))
		})
	}
}

func TestValidatePolicy(t *testing.T) {
	t.Parallel()

	require.NoError(t, ValidatePolicy(DefaultPolicy()))
	require.ErrorContains(t, ValidatePolicy(Policy{Enabled: true, Transition: TransitionArchivistToSniper}), "required_types")
	require.ErrorContains(t, ValidatePolicy(Policy{Enabled: true, Transition: "ranger_to_archivist", RequiredTypes: []string{ChallengeObjective}}), "not supported")
	assert.ErrorContains(t, ValidatePolicy(Policy{Enabled: true, Transition: TransitionArchivistToSniper, RequiredTypes: []string{"essay"}}), "not allowed")
}
