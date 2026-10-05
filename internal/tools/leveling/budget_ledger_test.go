package leveling_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBudgetLedgerAppendsEventsInOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-effort", "budget-events.jsonl")
	budget := testBudget()
	for _, event := range []leveling.BudgetEventKind{leveling.BudgetEventGrant, leveling.BudgetEventConsumption} {
		record := leveling.NewBudgetEventRecord(budget, event, "ok", "unverified", 1, 0)
		require.NoError(t, leveling.AppendBudgetEvent(path, record))
	}

	records, err := leveling.ReadBudgetEvents(path)
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, leveling.BudgetLedgerSchemaVersion, records[0].SchemaVersion)
	assert.Equal(t, leveling.BudgetEventGrant, records[0].Event)
	assert.Equal(t, leveling.BudgetEventConsumption, records[1].Event)
	assert.Equal(t, budget.ResolutionID, records[0].ResolutionID)
	assert.Equal(t, budget.PolicyDigest, records[0].PolicyDigest)
}

func TestBudgetLedgerIgnoresMalformedAndForeignSchemas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	require.NoError(t, os.WriteFile(path, []byte("not-json\n{\"schema_version\":\"other/v1\"}\n"), 0o600))
	records, err := leveling.ReadBudgetEvents(path)
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestBudgetLedgerRotatesToRecentEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	budget := testBudget()
	for i := 0; i < 4; i++ {
		record := leveling.NewBudgetEventRecord(budget, leveling.BudgetEventGrant, string(rune('a'+i)), "unverified", 1, 0)
		require.NoError(t, leveling.AppendBudgetEvent(path, record))
	}
	dropped, err := leveling.RotateBudgetEvents(path, 2)
	require.NoError(t, err)
	assert.Equal(t, 2, dropped)
	records, err := leveling.ReadBudgetEvents(path)
	require.NoError(t, err)
	assert.Len(t, records, 2)
	assert.Equal(t, "c", records[0].Reason)
	assert.Equal(t, "d", records[1].Reason)
}

func testBudget() domain.EffortBudget {
	return domain.EffortBudget{
		SchemaVersion: domain.EffortBudgetSchemaVersion, MissionID: "mission-1", RunID: "mission-1", Role: "ranger",
		Provider: "jev", Model: "jev-1", Capability: "handoff_validate", MaxDelegations: 2, MaxDepth: 0,
		Parallelism: 1, ContextBudget: "medium", EvidenceDepth: "standard", Permissions: domain.BudgetPermissions{Specialists: true},
		ResolutionID: "resolution-1", ParentResolutionID: "resolution-0", PolicyVersion: 1, PolicyDigest: "digest",
	}
}
