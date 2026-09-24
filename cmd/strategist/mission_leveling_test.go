package main

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/initiative"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/require"
)

func TestResolveMissionLevelingPersistsAndReusesTheResolutionEvent(t *testing.T) {
	root := t.TempDir()
	first, err := resolveMissionLeveling(root, "mission-level", "scout", "run-1")
	require.NoError(t, err)
	require.Equal(t, initiative.ObservationUnavailable, first.State)
	require.NotEmpty(t, first.EventID)

	path := filepath.Join(root, "memory", roleLevelLedger)
	records, err := leveling.ReadRecords(path)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "mission_start", records[0].Reason)

	second, err := resolveMissionLeveling(root, "mission-level", "scout", "run-1")
	require.NoError(t, err)
	require.Equal(t, first, second)
	records, err = leveling.ReadRecords(path)
	require.NoError(t, err)
	require.Len(t, records, 1, "a repeated role entry must not append another LEVELING event")
}

func TestStartInitiativeConsultationUsesThePersistedLevelingSnapshot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, startInitiativeConsultation(root, "mission-consult"))

	levelRecords, err := leveling.ReadRecords(filepath.Join(root, "memory", roleLevelLedger))
	require.NoError(t, err)
	require.Len(t, levelRecords, 1)
	initiativeRecords, err := initiative.ReadRecords(initiative.LedgerPath(root))
	require.NoError(t, err)
	require.Len(t, initiativeRecords, 1)
	require.NotNil(t, initiativeRecords[0].Advice.Leveling)
	require.Equal(t, levelRecords[0].MissionID, initiativeRecords[0].Advice.MissionID)
	require.Equal(t, levelRecords[0].Role, initiativeRecords[0].Advice.Leveling.Role)
	require.Equal(t, "leveling-mission-consult-mission-consult-"+levelRecords[0].Timestamp, initiativeRecords[0].Advice.Leveling.EventID)
}
