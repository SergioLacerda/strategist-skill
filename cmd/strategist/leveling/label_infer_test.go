package leveling

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internal "github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func inferTestPolicy(t *testing.T) internal.Policy {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "internal", "embed", "defaults", "leveling.yaml"))
	require.NoError(t, err)
	policy, err := internal.Parse(raw)
	require.NoError(t, err)
	return policy
}

// G1: prefix inference is observable but advisory and cannot supply an
// authority-bearing policy effort.
func TestLabelAutomaticModeInfersTheProviderFromTheHostModel(t *testing.T) {
	calls := 0
	load := func() (internal.Policy, error) { calls++; return inferTestPolicy(t), nil }
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	opts := LabelOptions{Role: "archivist", Mission: "m1", HostModel: "claude-opus-5", HostEffort: "<your-effort>"}

	got, err := LabelRoleWith(domain.DefaultRoleRegistry(), load, domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}, ledger, opts)
	require.NoError(t, err)
	assert.Empty(t, got.Level.Provider)
	assert.Empty(t, got.Level.Effort, "prefix inference must not fill policy effort")
	assert.Equal(t, "prefix_inferred", got.Level.ProviderMatch)
	assert.Equal(t, 1, calls)

	manualCalls := 0
	manualLoad := func() (internal.Policy, error) { manualCalls++; return inferTestPolicy(t), nil }
	got, err = LabelRoleWith(domain.DefaultRoleRegistry(), manualLoad, domain.LevelingConfig{Mode: domain.LevelingModeManual}, filepath.Join(t.TempDir(), "l.jsonl"), opts)
	require.NoError(t, err)
	assert.Empty(t, got.Level.Effort, "manual mode is host passthrough")
	assert.Zero(t, manualCalls, "manual mode never reads the policy")
}
