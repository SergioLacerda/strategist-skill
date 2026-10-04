package mission

import (
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/stretchr/testify/require"
)

func TestNextHandoffAttemptKeepsGlobalAuditButResetsSemanticBudgetForRepairedRevision(t *testing.T) {
	store := handoff.OutcomeStore{Root: t.TempDir(), Clock: func() time.Time { return time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC) }}
	policy := handoff.DefaultPolicy()
	policy.MaxAttempts = 2

	first := attemptOutcome(1, "sha256:old", handoff.OutcomePassed)
	_, err := store.Append(first)
	require.NoError(t, err)
	latest, err := store.Latest("m1")
	require.NoError(t, err)
	require.NoError(t, store.Invalidate(latest, "refinement_artifact_invalid"))

	attempt, err := nextHandoffAttempt(store, "m1", "sha256:repaired", policy)
	require.NoError(t, err)
	require.Equal(t, 2, attempt, "audit order is global while the new package has no failed semantic attempt")
}

func TestNextHandoffAttemptExhaustsOnlyFailuresForCurrentRevision(t *testing.T) {
	store := handoff.OutcomeStore{Root: t.TempDir(), Clock: func() time.Time { return time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC) }}
	policy := handoff.DefaultPolicy()
	policy.MaxAttempts = 2
	for attempt := 1; attempt <= 2; attempt++ {
		_, err := store.Append(attemptOutcome(attempt, "sha256:current", handoff.OutcomeFailed))
		require.NoError(t, err)
	}
	_, err := nextHandoffAttempt(store, "m1", "sha256:current", policy)
	require.ErrorContains(t, err, "semantic attempts")
}

func attemptOutcome(attempt int, digest, result string) handoff.Outcome {
	return handoff.Outcome{
		MissionID: "m1", Transition: handoff.TransitionArchivistToSniper, PackageDigest: digest,
		Attempt: attempt, Result: result, PolicyID: handoff.PolicyIdentity(handoff.DefaultPolicy()),
		Required: result != handoff.OutcomeSkipped, GateObserved: "handoff_challenge",
	}
}
