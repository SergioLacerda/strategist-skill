package application

import (
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestPersistMissionSubmitStateRollsBackChangedAnalysisOnSaveFailure(t *testing.T) {
	saveErr := errors.New("save failed")
	restored := false

	err := PersistMissionSubmitState("root", domain.MissionEngineStatus{MissionID: "m-1"}, "analysis.md", []byte("before"), true, MissionSubmitPersistencePorts{
		Save: func(string, domain.MissionEngineStatus) error { return saveErr },
		RestoreAnalysis: func(path string, original []byte) error {
			restored = path == "analysis.md" && string(original) == "before"
			return nil
		},
	})

	require.ErrorIs(t, err, saveErr)
	require.True(t, restored)
}

func TestPersistMissionSubmitStateDoesNotRollbackWhenSaveSucceeds(t *testing.T) {
	restored := false
	err := PersistMissionSubmitState("root", domain.MissionEngineStatus{MissionID: "m-1"}, "analysis.md", []byte("before"), true, MissionSubmitPersistencePorts{
		Save:            func(string, domain.MissionEngineStatus) error { return nil },
		RestoreAnalysis: func(string, []byte) error { restored = true; return nil },
	})

	require.NoError(t, err)
	require.False(t, restored)
}
