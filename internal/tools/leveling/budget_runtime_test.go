package leveling_test

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/stretchr/testify/require"
)

func TestSuggestIssuesBudgetOnlyAfterAuthoritativeProviderMapping(t *testing.T) {
	policy := defaultPolicy(t)

	suggestion, err := leveling.Suggest(policy, "CODEX", "ranger", leveling.Signals{})
	require.NoError(t, err)
	require.NotNil(t, suggestion.Budget)
	require.NoError(t, suggestion.Budget.Validate())
	require.Equal(t, "ranger", suggestion.Budget.Role)
	require.Equal(t, "CODEX", suggestion.Budget.Provider)
	require.Equal(t, suggestion.Model, suggestion.Budget.Model)
	require.Equal(t, policy.Digest(), suggestion.Budget.PolicyDigest)

	fallback, err := leveling.Suggest(policy, "new-provider", "ranger", leveling.Signals{})
	require.NoError(t, err)
	require.Nil(t, fallback.Budget, "unknown provider has no authoritative model mapping")
}

func TestBudgetBoundaryIsSynchronousAndFailClosed(t *testing.T) {
	budget := domain.EffortBudget{
		SchemaVersion:     domain.EffortBudgetSchemaVersion,
		MissionID:         "mission-1",
		RunID:             "run-1",
		Role:              "ranger",
		MaxDelegations:    1,
		MaxDepth:          1,
		Parallelism:       1,
		ContextBudget:     "medium",
		EvidenceDepth:     "standard",
		Permissions:       domain.BudgetPermissions{Specialists: true},
		EscalationAllowed: true,
		MaxEscalations:    1,
	}
	state, err := leveling.NewBudgetState(budget)
	require.NoError(t, err)

	denied := state.Begin(leveling.DelegationRequest{MissionID: "other", RunID: "run-1", Role: "ranger", Kind: leveling.DelegationSpecialist})
	require.False(t, denied.Granted)
	require.Equal(t, leveling.ReasonLevelingBudgetIdentityMismatch, denied.Reason)

	granted := state.Begin(leveling.DelegationRequest{MissionID: "mission-1", RunID: "run-1", Role: "ranger", Kind: leveling.DelegationSpecialist, Depth: 1, ContextBudget: "medium", EvidenceDepth: "standard"})
	require.True(t, granted.Granted)
	require.Equal(t, 0, granted.RemainingDelegations)

	parallel := state.Begin(leveling.DelegationRequest{MissionID: "mission-1", RunID: "run-1", Role: "ranger", Kind: leveling.DelegationSpecialist})
	require.False(t, parallel.Granted)
	require.Equal(t, leveling.ReasonLevelingBudgetExhausted, parallel.Reason)
	require.NoError(t, state.End())

	depth := state.Begin(leveling.DelegationRequest{MissionID: "mission-1", RunID: "run-1", Role: "ranger", Kind: leveling.DelegationSpecialist})
	require.False(t, depth.Granted)
	require.Equal(t, leveling.ReasonLevelingBudgetExhausted, depth.Reason, "consumption remains monotonic after End")
}

func TestEscalateBudgetCreatesImmutableLinkedChild(t *testing.T) {
	parent := domain.EffortBudget{
		SchemaVersion:     domain.EffortBudgetSchemaVersion,
		Role:              "ranger",
		MaxDelegations:    1,
		MaxDepth:          1,
		Parallelism:       1,
		ContextBudget:     "medium",
		EvidenceDepth:     "standard",
		EscalationAllowed: true,
		MaxEscalations:    1,
	}
	child, err := leveling.EscalateBudget(parent, "resolution-1", "repeated_failure")
	require.NoError(t, err)
	require.NoError(t, child.Validate())
	require.Equal(t, 0, parent.EscalationLevel)
	require.Equal(t, 1, child.EscalationLevel)
	require.Equal(t, "resolution-1", child.ParentResolutionID)
	require.Equal(t, "repeated_failure", child.EscalationReason)
	require.NotEmpty(t, child.ResolutionID)
	require.Equal(t, parent.MaxDelegations+1, child.MaxDelegations)

	_, err = leveling.EscalateBudget(child, child.ResolutionID, "again")
	require.ErrorContains(t, err, leveling.ReasonLevelingBudgetEscalationExhausted)
}

func TestEscalateBudgetReportsDisabledPolicySeparately(t *testing.T) {
	parent := domain.EffortBudget{
		SchemaVersion: domain.EffortBudgetSchemaVersion, Role: "ranger",
		MaxDelegations: 1, MaxDepth: 1, Parallelism: 1,
		ContextBudget: "medium", EvidenceDepth: "standard",
		EscalationAllowed: false, MaxEscalations: 0,
	}

	_, err := leveling.EscalateBudget(parent, "resolution-1", "repeated_failure")
	require.ErrorContains(t, err, leveling.ReasonLevelingBudgetEscalationDenied)
}

type recordingBudgetSink struct{ events []telemetry.Event }

func (s *recordingBudgetSink) Emit(_ context.Context, event telemetry.Event) error {
	s.events = append(s.events, event)
	return nil
}

func TestBudgetBoundarySeparatesLocalEnforcementTelemetry(t *testing.T) {
	budget := domain.EffortBudget{
		SchemaVersion: domain.EffortBudgetSchemaVersion, Role: "ranger",
		RunID: "run-telemetry", MaxDelegations: 1, MaxDepth: 1, Parallelism: 1,
		ContextBudget: "medium", EvidenceDepth: "standard",
		Permissions:       domain.BudgetPermissions{Specialists: true},
		EscalationAllowed: true, MaxEscalations: 1,
	}
	sink := &recordingBudgetSink{}
	state, err := leveling.NewBudgetStateWithTelemetry(budget, sink)
	require.NoError(t, err)
	require.True(t, state.Begin(leveling.DelegationRequest{Kind: leveling.DelegationSpecialist}).Granted)
	require.NoError(t, state.End())
	require.False(t, state.Begin(leveling.DelegationRequest{Kind: leveling.DelegationSpecialist}).Granted)

	require.Len(t, sink.events, 3)
	require.Equal(t, telemetry.LevelingBudgetGrantedEvent, sink.events[0].Name)
	require.Equal(t, telemetry.LevelingBudgetConsumedEvent, sink.events[1].Name)
	require.Equal(t, telemetry.LevelingBudgetExhaustedEvent, sink.events[2].Name)
	require.Equal(t, "unverified", sink.events[0].Attributes[telemetry.AttrLevelingProviderEvidence])
}
