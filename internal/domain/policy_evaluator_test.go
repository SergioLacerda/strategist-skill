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
		domain.TransitionGroupDocumentation,
		false, // gate not approved
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "approval_required", decision.Reason)
}

func TestDocumentationAllowedAfterReviewAcceptance(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.DefaultMissionPolicy(),
		domain.TransitionGroupDocumentation,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed", decision.Reason)
}

func TestReviewGateAllowedAfterAcceptance(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.DefaultMissionPolicy(),
		domain.TransitionGroupReviewGate,
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

func TestNormalizePolicyIsNoOp(t *testing.T) {
	t.Parallel()

	p := domain.NormalizePolicy(domain.MissionPolicy{})
	assert.Equal(t, domain.MissionPolicy{}, p)
}

func TestDefaultMissionPolicyIsEmpty(t *testing.T) {
	t.Parallel()

	p := domain.DefaultMissionPolicy()
	assert.Equal(t, domain.MissionPolicy{}, p)
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

func TestDocumentationRequiresReviewGateAcceptance(t *testing.T) {
	t.Parallel()

	// Regression: no explicit review acceptance must never allow documentation materialization.
	decision := domain.EvaluateGuardedTransition(
		domain.DefaultMissionPolicy(),
		domain.TransitionGroupDocumentation,
		false,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, "approval_required", decision.Reason)
}
