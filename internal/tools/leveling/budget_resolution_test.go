package leveling_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resolutionBudget() domain.EffortBudget {
	return domain.EffortBudget{
		SchemaVersion: domain.EffortBudgetSchemaVersion, MissionID: "mission-1", RunID: "run-1", Role: "ranger",
		Provider: "CLAUDE", Model: "claude-sonnet", Capability: "reasoning", MaxDelegations: 2, Parallelism: 1,
		ContextBudget: "medium", EvidenceDepth: "standard", PolicyVersion: 1, PolicyDigest: "sha256:policy",
	}
}

func TestBaseResolutionIDIsDeterministicAndCoversIdentity(t *testing.T) {
	t.Parallel()
	base := resolutionBudget()
	id := leveling.BaseResolutionID(base)

	require.NotEmpty(t, id)
	assert.Equal(t, id, leveling.BaseResolutionID(resolutionBudget()), "the same envelope always has the same id")

	for name, mutate := range map[string]func(*domain.EffortBudget){
		"mission":  func(b *domain.EffortBudget) { b.MissionID = "mission-2" },
		"run":      func(b *domain.EffortBudget) { b.RunID = "run-2" },
		"role":     func(b *domain.EffortBudget) { b.Role = "archivist" },
		"provider": func(b *domain.EffortBudget) { b.Provider = "CODEX" },
		"policy":   func(b *domain.EffortBudget) { b.PolicyDigest = "sha256:other" },
		"model":    func(b *domain.EffortBudget) { b.Model = "claude-opus" },
	} {
		changed := resolutionBudget()
		mutate(&changed)
		assert.NotEqual(t, id, leveling.BaseResolutionID(changed), "%s is part of the identity", name)
	}
}

func TestBaseResolutionIDIgnoresTheLimitsAndEscalationFields(t *testing.T) {
	t.Parallel()
	base := resolutionBudget()
	other := resolutionBudget()
	other.MaxDelegations = 9
	other.EscalationLevel = 1
	other.ParentResolutionID = "parent"
	other.ResolutionID = "already-set"

	assert.Equal(t, leveling.BaseResolutionID(base), leveling.BaseResolutionID(other),
		"the base id names the resolution, not the budget that came out of it, so it is stable when limits change")
}
