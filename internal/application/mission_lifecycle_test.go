package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestStartMissionKeepsPolicyBeforePersistence(t *testing.T) {
	t.Parallel()

	var order []string
	status, err := application.StartMission("root", "mission", application.MissionStartPorts{
		RequireNoExisting: func(string, string) error { order = append(order, "precondition"); return nil },
		InitiativeStart:   func(string, string) error { order = append(order, "initiative"); return nil },
		Save:              func(string, domain.MissionEngineStatus) error { order = append(order, "save"); return nil },
	})
	require.NoError(t, err)
	require.Equal(t, "mission", status.MissionID)
	require.Equal(t, []string{"precondition", "initiative", "save"}, order)
}

func TestStartMissionDoesNotPersistWhenInitiativeFails(t *testing.T) {
	t.Parallel()

	saved := false
	_, err := application.StartMission("root", "mission", application.MissionStartPorts{
		RequireNoExisting: func(string, string) error { return nil },
		InitiativeStart:   func(string, string) error { return errors.New("unavailable") },
		Save:              func(string, domain.MissionEngineStatus) error { saved = true; return nil },
	})
	require.ErrorContains(t, err, "initiative consultation")
	require.False(t, saved)
}

func TestRecordRouteCopiesInputAndUsesConnector(t *testing.T) {
	t.Parallel()

	raw := []byte("route")
	var got []byte
	called := false
	appended, err := application.RecordRoute(context.Background(), application.RecordRouteRequest{Root: "root", MissionID: "mission", Raw: raw}, func(_ context.Context, _, _ string, input []byte) (bool, error) {
		called = true
		got = input
		return true, nil
	})
	require.NoError(t, err)
	require.True(t, appended)
	require.True(t, called)
	raw[0] = 'X'
	require.Equal(t, []byte("route"), got)
}
