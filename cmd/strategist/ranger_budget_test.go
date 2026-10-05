package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRangerPrecheckWithoutBudgetIsUnavailableAndAudited(t *testing.T) {
	root := t.TempDir()
	report := rangerPrecheck(t.Context(), root, t.TempDir(), "mission-1")
	assert.Nil(t, report.Delegation)

	events, err := leveling.ReadBudgetEvents(filepath.Join(root, domain.DefaultBudgetLedgerPath))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, leveling.BudgetEventDenial, events[0].Event)
	assert.Equal(t, leveling.ReasonLevelingBudgetUnavailable, events[0].Reason)
}

func TestRangerPrecheckWithoutBudgetDoesNotInvokeConfiguredProvider(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, config.Save(filepath.Join(root, config.FileName), config.File{
		SchemaVersion: config.SchemaVersion,
		Providers: map[string]config.Provider{"jev": {
			Enabled: true, Endpoint: "https://127.0.0.1:1", CredentialRef: "env:STRATEGIST_TEST_KEY", Model: "jev-1",
			AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate},
			DataPolicy:          config.DataPolicy{AllowedFields: handoffconsumer.Fields("ranger_to_archivist"), MaxStateBytes: 16384},
		}},
	}))

	report := rangerPrecheck(t.Context(), root, t.TempDir(), "mission-1")
	assert.Equal(t, handoffconsumer.StatusFallback, report.Status)
	assert.NoFileExists(t, filepath.Join(root, "memory", "integration-calls.jsonl"))
}

func TestRangerPrecheckConsumesGrantedBudget(t *testing.T) {
	root := t.TempDir()
	writeRangerActiveConfig(t, root, "")
	writeRangerBudget(t, root, rangerBudget("mission-1", "mission-1", 1))

	basePath := filepath.Join(root, "analysis")
	pending := filepath.Join(basePath, "pending")
	require.NoError(t, os.MkdirAll(pending, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pending, "mission-1-analysis.md"), []byte("---\nmission_id: mission-1\n---\n\n## mission_objective\nEvaluate.\n## known_facts\n- one\n"), 0o644))

	report := rangerPrecheck(t.Context(), root, basePath, "mission-1")
	assert.Nil(t, report.Delegation, "no integration is configured, so the granted budget still ends without a delegation")

	events, err := leveling.ReadBudgetEvents(filepath.Join(root, domain.DefaultBudgetLedgerPath))
	require.NoError(t, err)
	require.Len(t, events, 3)
	assert.Equal(t, leveling.BudgetEventGrant, events[0].Event)
	assert.Equal(t, leveling.BudgetEventConsumption, events[1].Event)
	assert.Equal(t, leveling.BudgetEventProviderUsage, events[2].Event)
	assert.Equal(t, "mission-1", events[0].RunID)
}

func TestRangerPrecheckUsesCustomBudgetLedgerPath(t *testing.T) {
	root := t.TempDir()
	writeRangerActiveConfig(t, root, "audit/ranger-events.jsonl")
	writeRangerBudget(t, root, rangerBudget("mission-1", "mission-1", 0))
	_ = rangerPrecheck(t.Context(), root, t.TempDir(), "mission-1")

	events, err := leveling.ReadBudgetEvents(filepath.Join(root, "audit", "ranger-events.jsonl"))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, leveling.BudgetEventExhaustion, events[0].Event)
}

func TestRangerPrecheckDeniesMismatchedBudgetIdentity(t *testing.T) {
	root := t.TempDir()
	record := rangerBudget("mission-1", "mission-1", 1)
	record.Budget.Role = "archivist"
	writeRangerBudget(t, root, record)

	_ = rangerPrecheck(t.Context(), root, t.TempDir(), "mission-1")

	events, err := leveling.ReadBudgetEvents(filepath.Join(root, domain.DefaultBudgetLedgerPath))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, leveling.BudgetEventDenial, events[0].Event)
	assert.Equal(t, leveling.ReasonLevelingBudgetIdentityMismatch, events[0].Reason)
}

func rangerBudget(missionID, runID string, maxDelegations int) leveling.Record {
	return leveling.Record{MissionID: missionID, Run: runID, Level: leveling.Level{Role: "ranger", Budget: &domain.EffortBudget{
		SchemaVersion: domain.EffortBudgetSchemaVersion, MissionID: missionID, RunID: runID, Role: "ranger", Provider: "jev", Model: "jev-1",
		Capability: "handoff_validate", MaxDelegations: maxDelegations, MaxDepth: 0, Parallelism: 1, ContextBudget: "medium", EvidenceDepth: "standard",
		Permissions: domain.BudgetPermissions{Specialists: true}, ResolutionID: "resolution-1", PolicyVersion: 1, PolicyDigest: "digest",
	}}}
}

func writeRangerBudget(t *testing.T, root string, record leveling.Record) {
	t.Helper()
	require.NoError(t, leveling.AppendRecord(filepath.Join(root, "memory", roleLevelLedger), record))
}

func writeRangerActiveConfig(t *testing.T, root, budgetLedgerPath string) {
	t.Helper()
	contents := "leveling:\n"
	if budgetLedgerPath != "" {
		contents += "  budget_ledger_path: " + budgetLedgerPath + "\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte(contents), 0o600))
}
