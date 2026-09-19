package mission

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvaluateBudget(t *testing.T) {
	t.Parallel()
	observed := int64(9)

	tests := []struct {
		name  string
		input BudgetInput
		want  BudgetOutcome
	}{
		{
			name:  "fits with an unavailable provider observation",
			input: BudgetInput{Phase: "discovery", Role: "ranger", Content: "three estimated tokens", Limit: 3},
			want:  BudgetOutcome{Admitted: true, Reason: BudgetReasonAdmitted, EstimatedTokens: 3, RemainingTokens: 0},
		},
		{
			name:  "records provider observation separately",
			input: BudgetInput{Phase: "discovery", Role: "ranger", Content: "one", Limit: 2, ObservedProvider: &observed},
			want:  BudgetOutcome{Admitted: true, Reason: BudgetReasonAdmitted, EstimatedTokens: 1, RemainingTokens: 1, ObservedProvider: &observed, ObservationAvailable: true},
		},
		{
			name:  "rejects overflow without fallback",
			input: BudgetInput{Phase: "discovery", Role: "ranger", Content: "one two", Limit: 1},
			want:  BudgetOutcome{Reason: BudgetReasonOverflow, EstimatedTokens: 2, RemainingTokens: -1},
		},
		{
			name:  "rejects malformed limit",
			input: BudgetInput{Phase: "discovery", Role: "ranger", Limit: -1},
			want:  BudgetOutcome{Reason: BudgetReasonMalformed},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EvaluateBudget(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestBudgetOutcomeTelemetryAttributesAreRedacted(t *testing.T) {
	t.Parallel()
	outcome := EvaluateBudget(BudgetInput{Phase: "discovery", Role: "ranger", Content: "secret payload", Limit: 3})
	attributes := outcome.TelemetryAttributes()

	require.Equal(t, true, attributes["strategist.budget.admitted"])
	require.Equal(t, int64(2), attributes["strategist.budget.estimated_tokens"])
	require.NotContains(t, attributes, "content")
	require.NotContains(t, attributes, "provider_payload")
}
