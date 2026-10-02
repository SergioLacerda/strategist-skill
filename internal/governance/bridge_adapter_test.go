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

func TestBridge_Evaluate_Allowed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillPath := writeSkillYAML(t, dir, "skill.yaml", map[string]any{
		"compliance": map[string]any{"mandates": []any{"M001", "M002"}},
	})
	b := NewBridge(filepath.Dir(skillPath), filepath.Join(dir, "governance"), validFakeSource())
	decision, err := b.Evaluate(context.Background(), governancebridge.GovernanceRequest{
		MissionID: "m-1", CorrelationID: "corr-1",
	})
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, telemetry.AuthorityExternal("fixture"), decision.Authority)
	assert.Equal(t, "corr-1", decision.CorrelationID)
	assert.Equal(t, "abc123", decision.PolicyID)
}

func TestBridge_Evaluate_NotAllowedWhenMandateMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillPath := writeSkillYAML(t, dir, "skill.yaml", map[string]any{
		"compliance": map[string]any{"mandates": []any{"M001"}},
	})
	source := validFakeSource()
	source.snapshot.ActiveMandates = []string{"M001", "M002"}
	b := NewBridge(filepath.Dir(skillPath), filepath.Join(dir, "governance"), source)
	decision, err := b.Evaluate(context.Background(), governancebridge.GovernanceRequest{})
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "M002")
	assert.Equal(t, "fixture:abc123", decision.CorrelationID)
}

func TestBridge_Evaluate_PropagatesReadError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	b := NewBridge(dir, filepath.Join(dir, "governance"), fakeSource{err: os.ErrNotExist})
	_, err := b.Evaluate(context.Background(), governancebridge.GovernanceRequest{})
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestBridge_Evaluate_NeverMutatesSkillYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillPath := writeSkillYAML(t, dir, "skill.yaml", map[string]any{})
	before, err := os.ReadFile(skillPath) //nolint:gosec // test-controlled temp path
	require.NoError(t, err)
	b := NewBridge(filepath.Dir(skillPath), filepath.Join(dir, "governance"), validFakeSource())
	_, err = b.Evaluate(context.Background(), governancebridge.GovernanceRequest{})
	require.NoError(t, err)
	after, err := os.ReadFile(skillPath) //nolint:gosec // test-controlled temp path
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}
