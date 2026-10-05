package levelingapp_test

import (
	"path/filepath"
	"testing"

	levelingapp "github.com/SergioLacerda/strategist-skill/internal/application/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
	leveling "github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func embeddedPolicyLoader(t *testing.T) leveling.PolicyLoader {
	t.Helper()
	raw, err := (embed.Extractor{}).ReadFile("leveling.yaml")
	require.NoError(t, err)
	policy, err := leveling.Parse(raw)
	require.NoError(t, err)
	return func() (leveling.Policy, error) { return policy, nil }
}

func resolveFor(t *testing.T, ledger string, input levelingapp.LevelingInput) levelingapp.LevelingResult {
	t.Helper()
	result, err := levelingapp.ResolveLevel(domain.DefaultRoleRegistry(), embeddedPolicyLoader(t), domain.LevelingConfig{}, ledger, input, 50)
	require.NoError(t, err)
	return result
}

func rangerInput(reason string) levelingapp.LevelingInput {
	return levelingapp.LevelingInput{Role: "ranger", Provider: "CLAUDE", Mission: "mission-1", Run: "mission-1", HostModel: "claude-sonnet-5-5", HostEffort: "medium", Reason: reason}
}

func TestResolveLevelAssignsADeterministicBaseResolutionID(t *testing.T) {
	t.Parallel()
	first := resolveFor(t, filepath.Join(t.TempDir(), "a.jsonl"), rangerInput(""))
	second := resolveFor(t, filepath.Join(t.TempDir(), "b.jsonl"), rangerInput(""))

	require.NotNil(t, first.Level.Budget)
	require.NotEmpty(t, first.Level.Budget.ResolutionID)
	assert.Equal(t, first.Level.Budget.ResolutionID, second.Level.Budget.ResolutionID)
	assert.Equal(t, 0, first.Level.Budget.EscalationLevel)
	assert.Empty(t, first.Level.Budget.ParentResolutionID)
	assert.False(t, first.Escalation.Attempted, "a plain resolution never escalates")
}

func TestResolveLevelReplayKeepsTheBudgetAndItsID(t *testing.T) {
	t.Parallel()
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	first := resolveFor(t, ledger, rangerInput(""))
	replay := resolveFor(t, ledger, rangerInput(""))

	assert.True(t, replay.Reused)
	require.NotNil(t, replay.Level.Budget)
	assert.Equal(t, first.Level.Budget.ResolutionID, replay.Level.Budget.ResolutionID)
}

func TestEscalatedReasonLinksAChildEnvelopeToTheParentResolution(t *testing.T) {
	t.Parallel()
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	parent := resolveFor(t, ledger, rangerInput(""))

	child := resolveFor(t, ledger, rangerInput("escalated"))

	require.NotNil(t, child.Level.Budget)
	assert.True(t, child.Escalation.Attempted)
	assert.True(t, child.Escalation.Granted)
	assert.Equal(t, parent.Level.Budget.ResolutionID, child.Level.Budget.ParentResolutionID)
	assert.Equal(t, 1, child.Level.Budget.EscalationLevel)
	assert.Equal(t, "escalated", child.Level.Budget.EscalationReason)
	assert.NotEqual(t, parent.Level.Budget.ResolutionID, child.Level.Budget.ResolutionID)
	assert.Equal(t, parent.Level.Budget.MaxDelegations+1, child.Level.Budget.MaxDelegations)
	assert.Equal(t, child.Level.Budget.ResolutionID, child.Escalation.ResolutionID)

	record, found, err := leveling.LatestRunRecord(ledger, "mission-1", "ranger", "mission-1")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, child.Level.Budget.ResolutionID, record.Budget.ResolutionID, "the escalated tuple is what the next consumer reads")
}

func TestEscalationIsExhaustedExplicitlyAndKeepsTheEffectiveEnvelope(t *testing.T) {
	t.Parallel()
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	_ = resolveFor(t, ledger, rangerInput(""))
	first := resolveFor(t, ledger, rangerInput("escalated"))
	require.True(t, first.Escalation.Granted)

	second := resolveFor(t, ledger, rangerInput("escalated"))

	assert.True(t, second.Escalation.Attempted)
	assert.False(t, second.Escalation.Granted)
	assert.Equal(t, leveling.ReasonLevelingBudgetEscalationExhausted, second.Escalation.Reason)
	require.NotNil(t, second.Level.Budget)
	assert.Equal(t, first.Level.Budget.ResolutionID, second.Level.Budget.ResolutionID, "the effective envelope is the last granted one")
	assert.Equal(t, 1, second.Level.Budget.EscalationLevel)
	assert.Equal(t, first.Level.Budget.MaxDelegations, second.Level.Budget.MaxDelegations)
}

func TestEscalationWithoutAPriorBudgetIsNotAnAttempt(t *testing.T) {
	t.Parallel()
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")

	result := resolveFor(t, ledger, rangerInput("escalated"))

	assert.False(t, result.Escalation.Attempted, "there is nothing to escalate")
	require.NotNil(t, result.Level.Budget)
	assert.Equal(t, 0, result.Level.Budget.EscalationLevel)
}

func TestEscalationWhenTheNewLevelHasNoBudgetIsExplicitlyUnavailable(t *testing.T) {
	t.Parallel()
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	parent := resolveFor(t, ledger, rangerInput(""))
	require.NotNil(t, parent.Level.Budget)

	hostOnly := rangerInput("escalated")
	hostOnly.Provider = ""
	result := resolveFor(t, ledger, hostOnly)

	assert.True(t, result.Escalation.Attempted)
	assert.False(t, result.Escalation.Granted)
	assert.Equal(t, leveling.ReasonLevelingBudgetUnavailable, result.Escalation.Reason,
		"a missing budget is told apart from a denied or exhausted escalation")
	assert.Nil(t, result.Level.Budget)
}
