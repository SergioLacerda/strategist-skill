package application_test

import (
	"errors"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/stretchr/testify/require"
)

func TestRecordMissionTokenUsageBuildsAndAppendsRecord(t *testing.T) {
	t.Parallel()

	var got application.MissionTokenUsageRecord
	record, err := application.RecordMissionTokenUsage("root", "base", application.MissionTokenUsageInput{
		MissionID: "m-1", TokensIn: 12, TokensOut: 7,
	}, time.Date(2026, time.October, 3, 1, 2, 3, 0, time.FixedZone("BRT", -3*60*60)), application.MissionTokenUsagePorts{
		MissionKnown: func(string, string) bool { return true },
		Append: func(_ string, record application.MissionTokenUsageRecord) error {
			got = record
			return nil
		},
	})

	require.NoError(t, err)
	require.Equal(t, application.MissionUsageSourceAgentReport, record.Source)
	require.Equal(t, "2026-10-03T04:02:03Z", record.ReportedAt)
	require.Equal(t, record, got)
}

func TestRecordMissionTokenUsageRejectsUnknownMission(t *testing.T) {
	t.Parallel()

	_, err := application.RecordMissionTokenUsage("root", "base", application.MissionTokenUsageInput{MissionID: "missing"}, time.Time{}, application.MissionTokenUsagePorts{
		MissionKnown: func(string, string) bool { return false },
		Append:       func(string, application.MissionTokenUsageRecord) error { return nil },
	})
	require.ErrorContains(t, err, `unknown mission_id "missing"`)
}

func TestRecordMissionTokenUsagePropagatesAppendError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("append failed")
	_, err := application.RecordMissionTokenUsage("root", "base", application.MissionTokenUsageInput{MissionID: "m-1"}, time.Time{}, application.MissionTokenUsagePorts{
		MissionKnown: func(string, string) bool { return true },
		Append:       func(string, application.MissionTokenUsageRecord) error { return wantErr },
	})
	require.ErrorIs(t, err, wantErr)
}

func TestRecordMissionTokenUsageRejectsMissingPorts(t *testing.T) {
	t.Parallel()

	_, err := application.RecordMissionTokenUsage("root", "base", application.MissionTokenUsageInput{MissionID: "m-1"}, time.Time{}, application.MissionTokenUsagePorts{})
	require.EqualError(t, err, "mission report-usage: usage ports are required")
}
