package application_test

import (
	"errors"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/stretchr/testify/require"
)

func TestBuildHandoffChallengeRecordUsesPolicyTransitionFallback(t *testing.T) {
	t.Parallel()

	record := application.BuildHandoffChallengeRecord(application.HandoffChallengeInput{
		MissionID:        "m-1",
		PolicyTransition: "archivist_to_sniper",
		Status:           "passed",
		Passed:           true,
	}, time.Date(2026, time.October, 3, 1, 2, 3, 0, time.FixedZone("BRT", -3*60*60)))

	require.Equal(t, "archivist_to_sniper", record.Transition)
	require.Equal(t, "2026-10-03T04:02:03Z", record.Timestamp)
}

func TestBuildHandoffChallengeRecordPrefersExplicitTransition(t *testing.T) {
	t.Parallel()

	record := application.BuildHandoffChallengeRecord(application.HandoffChallengeInput{
		Transition:       "sniper_to_validation",
		PolicyTransition: "archivist_to_sniper",
	}, time.Time{})

	require.Equal(t, "sniper_to_validation", record.Transition)
}

func TestRecordHandoffChallengeDelegatesRecord(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("append failed")
	var got application.HandoffChallengeRecord
	err := application.RecordHandoffChallenge(application.HandoffChallengeInput{MissionID: "m-1"}, time.Unix(0, 0), func(record application.HandoffChallengeRecord) error {
		got = record
		return wantErr
	})

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, "m-1", got.MissionID)
}

func TestRecordHandoffChallengeRejectsMissingRecorder(t *testing.T) {
	t.Parallel()

	err := application.RecordHandoffChallenge(application.HandoffChallengeInput{}, time.Time{}, nil)
	require.EqualError(t, err, "handoff challenge recorder is unavailable")
}
