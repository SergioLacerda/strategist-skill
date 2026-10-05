package selectrunbook

import (
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
	"github.com/stretchr/testify/assert"
)

func TestSelectionMatchingAndPolicyEdgeBranches(t *testing.T) {
	assert.Zero(t, countByRole([]Selection{{Role: RolePrimary}}, RoleSupporting))

	assert.False(t, declaredSignalMatches("provider", MissionSignals{""}))
	assert.Empty(t, matchAppliesWhen([]string{"", "unrelated"}, MissionSignals{"signal"}))
	assert.False(t, triggerMatchesAnySignal("provider", MissionSignals{""}))
	assert.False(t, rawTriggerMatches("provider", ""))
	assert.False(t, containsAtWordBoundary("provider", "missing"))
	assert.True(t, boundaryBefore("provider", 0, "provider"))
	assert.True(t, boundaryBefore("--", 1, "-"))

	assert.True(t, meetsMinTrust("unknown", "T0"))
	assert.True(t, isStale(Runbook{Metadata: runbook.Metadata{ReviewedAt: "not-a-date"}}, SelectionPolicy{MaxAge: time.Hour}))
	assert.False(t, isStale(Runbook{Metadata: runbook.Metadata{ReviewedAt: time.Now().UTC().Format("2006-01-02")}}, SelectionPolicy{
		MaxAge: 48 * time.Hour,
	}))
}
