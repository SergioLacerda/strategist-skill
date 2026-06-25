package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestGuardedTransitionRequiresGate(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.DefaultMissionPolicy(),
		domain.TransitionGroupExecution,
		false, // gate not approved
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "approval_required", decision.Reason)
}

func TestExecutionAllowedAfterGateApproval(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.DefaultMissionPolicy(),
		domain.TransitionGroupExecution,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
}

func TestFinalizeAnalysisAllowedWithGate(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.DefaultMissionPolicy(),
		domain.TransitionGroupFinalizeAnalysis,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
}

func TestFinalizeAnalysisRequiresGate(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.DefaultMissionPolicy(),
		domain.TransitionGroupFinalizeAnalysis,
		false,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "approval_required", decision.Reason)
}

func TestNormalizePolicySetsCanExecuteTrue(t *testing.T) {
	t.Parallel()

	// Even a zero-value MissionPolicy (CanExecute=false) is normalized to true.
	p := domain.NormalizePolicy(domain.MissionPolicy{})
	assert.True(t, p.CanExecute)
}

func TestDefaultMissionPolicyCanExecute(t *testing.T) {
	t.Parallel()

	p := domain.DefaultMissionPolicy()
	assert.True(t, p.CanExecute)
}

func TestUnknownTransitionGroupBlocked(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.DefaultMissionPolicy(),
		"unknown_group",
		true,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "unknown_transition_group", decision.Reason)
}

func TestIncidentUXStratetist(t *testing.T) {
	t.Parallel()

	// Regression: no explicit approval must never allow execution-like transition.
	decision := domain.EvaluateGuardedTransition(
		domain.DefaultMissionPolicy(),
		domain.TransitionGroupExecution,
		false,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "approval_required", decision.Reason)
}
