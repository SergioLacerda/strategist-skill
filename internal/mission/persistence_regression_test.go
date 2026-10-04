package mission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMissionIDValidationBranches(t *testing.T) {
	for _, id := range []string{"", "Upper", "has_underscore", "../escape", "-starts"} {
		require.Error(t, ValidateMissionID(id))
	}
	require.NoError(t, ValidateMissionID("20261003-valid-mission"))
	require.NoError(t, ValidateMissionID("valid"))
}

func TestMissionPersistenceRoundTrip(t *testing.T) {
	root := t.TempDir()
	engine, status, err := domain.StartMission(domain.MissionStartRequest{MissionID: "coverage-mission"})
	require.NoError(t, err)
	require.NotNil(t, engine)

	require.NoError(t, Save(root, status))
	loaded, loadedStatus, err := Load(root, status.MissionID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, status, loadedStatus)
}

func TestMissionPersistenceRejectsMissingAndInvalidState(t *testing.T) {
	root := t.TempDir()
	_, _, err := Load(root, "missing-mission")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	path := Path(root, "invalid-mission")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("{"), 0o644))
	_, _, err = Load(root, "invalid-mission")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid persisted state")

	require.NoError(t, os.WriteFile(path, []byte(`{"mission_id":"bad-mission"}`), 0o644))
	_, _, err = Load(root, "invalid-mission")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restore mission state")
}
