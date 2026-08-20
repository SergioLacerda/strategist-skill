package governance

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/governancebridge"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSDDBridge_Evaluate_Allowed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeSddFixtures(t, dir, []string{"M001"})
	skillPath := writeSkillYAML(t, dir, "skill.yaml", map[string]any{
		"compliance": map[string]any{"mandates": []any{"M001"}},
	})
	skillRoot := filepath.Dir(skillPath)

	b := NewSDDBridge(skillRoot, filepath.Join(dir, ".sdd"))
	decision, err := b.Evaluate(context.Background(), governancebridge.GovernanceRequest{
		MissionID: "m-1", CorrelationID: "corr-1",
	})
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, telemetry.AuthorityExternal("sdd"), decision.Authority)
	assert.Equal(t, "corr-1", decision.CorrelationID)
	assert.Equal(t, "abc123", decision.PolicyID)
}

func TestSDDBridge_Evaluate_NotAllowedWhenMandateMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeSddFixtures(t, dir, []string{"M001", "M002"})
	skillPath := writeSkillYAML(t, dir, "skill.yaml", map[string]any{
		"compliance": map[string]any{"mandates": []any{"M001"}},
	})
	skillRoot := filepath.Dir(skillPath)

	b := NewSDDBridge(skillRoot, filepath.Join(dir, ".sdd"))
	decision, err := b.Evaluate(context.Background(), governancebridge.GovernanceRequest{})
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "M002")
}

func TestSDDBridge_Evaluate_PropagatesReadError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	b := NewSDDBridge(dir, filepath.Join(dir, ".sdd")) // no fixtures written
	_, err := b.Evaluate(context.Background(), governancebridge.GovernanceRequest{})
	require.Error(t, err)
}

// TestSDDBridge_Evaluate_NeverMutatesSkillYAML is acceptance check 6.7: a
// GovernanceBridge decision must be read-only — Evaluate must never write
// skill.yaml as a side effect, unlike RunSync(dryRun=false).
func TestSDDBridge_Evaluate_NeverMutatesSkillYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeSddFixtures(t, dir, []string{"M001"})
	// Deliberately omit compliance.mandates so applyMissingFields/computeComplianceGaps
	// would have something to change if Evaluate ever ran with dryRun=false.
	skillPath := writeSkillYAML(t, dir, "skill.yaml", map[string]any{})
	skillRoot := filepath.Dir(skillPath)

	before, err := os.ReadFile(skillPath) //nolint:gosec // G304: test temp path
	require.NoError(t, err)

	b := NewSDDBridge(skillRoot, filepath.Join(dir, ".sdd"))
	_, err = b.Evaluate(context.Background(), governancebridge.GovernanceRequest{})
	require.NoError(t, err)

	after, err := os.ReadFile(skillPath) //nolint:gosec // G304: test temp path
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after), "Evaluate must never write skill.yaml")
}

var _ governancebridge.GovernanceBridge = SDDBridge{}
