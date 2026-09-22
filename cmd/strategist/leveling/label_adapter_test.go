package leveling

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internal "github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDeps(root string, emitted *[]internal.Level) Dependencies {
	return Dependencies{
		LoadPolicy: func() (internal.Policy, string, error) {
			return internal.Policy{}, "", fmt.Errorf("policy should not load")
		},
		WorkspaceRoot: func() (string, error) { return root, nil },
		LoadConfig: func(string) (domain.LevelingConfig, string) {
			return domain.LevelingConfig{}, ""
		},
		LoadRegistry: func(string) (domain.RoleRegistry, string) {
			return domain.DefaultRoleRegistry(), ""
		},
		LedgerName: "role-levels.jsonl",
		RotateMax:  20,
		EmitRoleLevel: func(_ context.Context, _ string, _ string, level internal.Level, _ string) {
			*emitted = append(*emitted, level)
		},
	}
}

func TestNewBuildsCompleteIndependentCommandTree(t *testing.T) {
	var emitted []internal.Level
	cmd := New(testDeps(t.TempDir(), &emitted))
	require.Len(t, cmd.Commands(), 3)
	for _, name := range []string{"validate", "suggest", "label"} {
		found, _, err := cmd.Find([]string{name})
		require.NoError(t, err)
		assert.Equal(t, name, found.Name())
	}

	root := &cobra.Command{Use: "root"}
	Register(root, testDeps(t.TempDir(), &emitted))
	require.Len(t, root.Commands(), 1)
	assert.Equal(t, "leveling", root.Commands()[0].Name())
}

func TestRunLabelUsesInjectedRuntimeDependencies(t *testing.T) {
	root := t.TempDir()
	var emitted []internal.Level
	cmd := NewLabel(testDeps(root, &emitted), &LabelOptions{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	require.NoError(t, cmd.Flags().Set("role", "ranger"))
	require.NoError(t, cmd.Flags().Set("mission", "m1"))
	require.NoError(t, cmd.Flags().Set("host-model", "Sonnet"))
	require.NoError(t, cmd.Flags().Set("host-effort", "high"))
	require.NoError(t, cmd.Flags().Set("message", "ready"))
	require.NoError(t, cmd.Flags().Set("width", "80"))
	require.NoError(t, cmd.RunE(cmd, nil))

	assert.Equal(t, "Ranger(Sonnet-High) - ready\n", out.String())
	require.Len(t, emitted, 1)
	assert.Equal(t, "ranger", emitted[0].Role)
	record, ok, err := internal.LatestRecord(filepath.Join(root, "memory", "role-levels.jsonl"), "m1", "ranger")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "Sonnet-High", record.Label())
}
