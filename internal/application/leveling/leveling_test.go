package levelingapp_test

import (
	"path/filepath"
	"testing"

	levelingapp "github.com/SergioLacerda/strategist-skill/internal/application/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	leveling "github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/stretchr/testify/require"
)

func TestResolveLevelKeepsLedgerBehaviorBehindApplicationBoundary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	result, err := levelingapp.ResolveLevel(domain.DefaultRoleRegistry(), func() (leveling.Policy, error) {
		return leveling.Policy{}, nil
	}, domain.LevelingConfig{}, filepath.Join(root, "role-levels.jsonl"), levelingapp.LevelingInput{Role: "ranger", Mission: "m-1", HostModel: "opus", HostEffort: "high"}, 20)
	require.NoError(t, err)
	require.Equal(t, "ranger", result.Level.Role)
}
