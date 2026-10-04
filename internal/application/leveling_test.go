package application_test

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	leveling "github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/require"
)

func TestResolveLevelKeepsLedgerBehaviorBehindApplicationBoundary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	result, err := application.ResolveLevel(domain.DefaultRoleRegistry(), func() (leveling.Policy, error) {
		return leveling.Policy{}, nil
	}, domain.LevelingConfig{}, filepath.Join(root, "role-levels.jsonl"), application.LevelingInput{Role: "ranger", Mission: "m-1", HostModel: "opus", HostEffort: "high"}, 20)
	require.NoError(t, err)
	require.Equal(t, "ranger", result.Level.Role)
}
