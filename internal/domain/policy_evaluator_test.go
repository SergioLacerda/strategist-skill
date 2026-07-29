package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestGuardedTransitionRequiresGate(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.TransitionGroupDocumentation,
		false, // gate not approved
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, domain.PolicyReasonApprovalRequired, decision.Reason)
	assert.Equal(t, domain.PolicyStatusApprovalRequired, decision.Status)
}

func TestDocumentationAllowedAfterReviewAcceptance(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.TransitionGroupDocumentation,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, domain.PolicyReasonAllowed, decision.Reason)
	assert.Equal(t, domain.PolicyStatusAllowed, decision.Status)
}

func TestReviewGateAllowedAfterAcceptance(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.TransitionGroupReviewGate,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, domain.PolicyReasonAllowed, decision.Reason)
	assert.Equal(t, domain.PolicyStatusAllowed, decision.Status)
}

func TestFinalizeAnalysisAllowedWithGate(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.TransitionGroupFinalizeAnalysis,
		true,
	)

	assert.True(t, decision.Allowed)
	assert.Equal(t, domain.PolicyReasonAllowed, decision.Reason)
	assert.Equal(t, domain.PolicyStatusAllowed, decision.Status)
}

func TestFinalizeAnalysisRequiresGate(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		domain.TransitionGroupFinalizeAnalysis,
		false,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, domain.PolicyReasonApprovalRequired, decision.Reason)
	assert.Equal(t, domain.PolicyStatusApprovalRequired, decision.Status)
}

func TestUnknownTransitionGroupBlocked(t *testing.T) {
	t.Parallel()

	decision := domain.EvaluateGuardedTransition(
		"unknown_group",
		true,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, domain.PolicyReasonUnknownTransitionGroup, decision.Reason)
	assert.Equal(t, domain.PolicyStatusBlocked, decision.Status)
}

func TestDocumentationRequiresReviewGateAcceptance(t *testing.T) {
	t.Parallel()

	// Regression: no explicit review acceptance must never allow documentation materialization.
	decision := domain.EvaluateGuardedTransition(
		domain.TransitionGroupDocumentation,
		false,
	)

	assert.False(t, decision.Allowed)
	assert.Equal(t, domain.PolicyReasonApprovalRequired, decision.Reason)
	assert.Equal(t, domain.PolicyStatusApprovalRequired, decision.Status)
}
