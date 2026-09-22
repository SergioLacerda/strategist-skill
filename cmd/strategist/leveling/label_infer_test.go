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

// G1: in automatic mode the role on_start hook (host model, no provider) now
// yields a policy-completed level instead of an empty one.
func TestLabelAutomaticModeInfersTheProviderFromTheHostModel(t *testing.T) {
	calls := 0
	load := func() (internal.Policy, error) { calls++; return inferTestPolicy(t), nil }
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	opts := LabelOptions{Role: "archivist", Mission: "m1", HostModel: "claude-opus-5", HostEffort: "<your-effort>"}

	got, err := LabelRoleWith(domain.DefaultRoleRegistry(), load, domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}, ledger, opts)
	require.NoError(t, err)
	assert.Equal(t, "CLAUDE", got.Level.Provider)
	assert.NotEmpty(t, got.Level.Effort, "the policy fills the effort the hook could not")
	assert.Equal(t, 1, calls)

	manualCalls := 0
	manualLoad := func() (internal.Policy, error) { manualCalls++; return inferTestPolicy(t), nil }
	got, err = LabelRoleWith(domain.DefaultRoleRegistry(), manualLoad, domain.LevelingConfig{Mode: domain.LevelingModeManual}, filepath.Join(t.TempDir(), "l.jsonl"), opts)
	require.NoError(t, err)
	assert.Empty(t, got.Level.Effort, "manual mode is host passthrough")
	assert.Zero(t, manualCalls, "manual mode never reads the policy")
}
