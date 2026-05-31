package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestGuardedTransitionRequiresGate(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.MissionPolicy{DoneScope: domain.DoneScopeEntrega, ApplyChanges: true},
		domain.TransitionGroupExecution,
		false,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "approval_required", decision.Reason)
}

func TestDefaultMissionModePreservesLegacyExecution(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.MissionPolicy{},
		domain.TransitionGroupExecution,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
	assert.Equal(t, domain.MissionModeEntregaExecutada, decision.Policy.Mode)
}

func TestDoneAnaliseSkipsExecution(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.MissionPolicy{DoneScope: domain.DoneScopeAnalise, ApplyChanges: true},
		domain.TransitionGroupExecution,
		true,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "policy_blocked", decision.Reason)
}

func TestExecutionAllowedWhenEntregaAndApplyChanges(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.MissionPolicy{DoneScope: domain.DoneScopeEntrega, ApplyChanges: true},
		domain.TransitionGroupExecution,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
}

func TestFinalizeAnalysisAllowedWithGate(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.MissionPolicy{DoneScope: domain.DoneScopeAnalise, ApplyChanges: false},
		domain.TransitionGroupFinalizeAnalysis,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
}

func TestIncidentUXStratetist(t *testing.T) {
	t.Parallel()

	// Regression: no explicit approval must never allow execution-like transition.
	decision := domain.EvaluateGuardedTransition(
		domain.MissionPolicy{DoneScope: domain.DoneScopeEntrega, ApplyChanges: true},
		domain.TransitionGroupExecution,
		false,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "approval_required", decision.Reason)
}

func TestQuickDrawPolicyLockedForAnalysisMode(t *testing.T) {
	t.Parallel()

	// Quick draw execution is an execution-like guarded transition.
	decision := domain.EvaluateGuardedTransition(
		domain.NewMissionPolicy(domain.MissionModeAnalise),
		domain.TransitionGroupExecution,
		true,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "policy_blocked", decision.Reason)
}

func TestQuickDrawAllowedForEntregaExecutadaWithGate(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.NewMissionPolicy(domain.MissionModeEntregaExecutada),
		domain.TransitionGroupExecution,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
}
