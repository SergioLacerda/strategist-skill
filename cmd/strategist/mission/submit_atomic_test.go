package mission

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistSubmitStateRestoresAnalysisWhenMissionSaveFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	original := []byte("---\nmission_status: archivist_done\n---\n")
	require.NoError(t, os.WriteFile(path, []byte("---\nmission_status: gate_analysis_accepted\n---\n"), 0o600))
	deps := LifecycleDependencies{Save: func(string, domain.MissionEngineStatus) error { return errors.New("save failed") }}

	err := persistSubmitState(deps, "root", domain.MissionEngineStatus{}, path, original, true)

	require.ErrorContains(t, err, "save failed")
	restored, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, original, restored)
}
