package levelingapp_test

import (
	"path/filepath"
	"testing"

	levelingapp "github.com/SergioLacerda/strategist-skill/internal/application/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
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

func TestResolveLevelCorrelatesBudgetWithMissionAndRun(t *testing.T) {
	raw, err := (embed.Extractor{}).ReadFile("leveling.yaml")
	require.NoError(t, err)
	policy, err := leveling.Parse(raw)
	require.NoError(t, err)

	result, err := levelingapp.ResolveLevel(domain.DefaultRoleRegistry(), func() (leveling.Policy, error) {
		return policy, nil
	}, domain.LevelingConfig{}, filepath.Join(t.TempDir(), "role-levels.jsonl"), levelingapp.LevelingInput{
		Role: "ranger", Provider: "CODEX", Mission: "mission-1", Run: "run-2",
	}, 20)
	require.NoError(t, err)
	require.NotNil(t, result.Level.Budget)
	require.Equal(t, "mission-1", result.Level.Budget.MissionID)
	require.Equal(t, "run-2", result.Level.Budget.RunID)
	require.Equal(t, "ranger", result.Level.Budget.Role)
}
