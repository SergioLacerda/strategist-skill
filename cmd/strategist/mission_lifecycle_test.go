package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordMissionLifecycleWritesNumberedIdempotentMarkers(t *testing.T) {
	root := t.TempDir()
	start := domain.LifecycleMarkersForStart(domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseBootstrap, State: domain.StateInit})
	require.NoError(t, recordMissionLifecycle(root, "m-1", start))
	require.NoError(t, recordMissionLifecycle(root, "m-1", start), "a replay is a no-op")

	before := domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseBootstrap, State: domain.StateInit}
	after := domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseIntake, State: domain.StateInit}
	require.NoError(t, recordMissionLifecycle(root, "m-1",
		domain.LifecycleMarkersForTransition(domain.MissionEventBootstrapDone, before, after)))

	got, err := telemetry.ReadMissionLifecycleFor(telemetry.MissionLifecyclePath(root), "m-1")
	require.NoError(t, err)
	require.Len(t, got, 4)
	assert.Equal(t, "mission_started", got[0].MarkerKind)
	assert.Equal(t, "r1", got[1].PhaseRunID)
	assert.Equal(t, "phase_ended", got[2].MarkerKind)
	assert.Equal(t, "completed", got[2].EndReason)
	assert.Equal(t, "INTAKE", got[3].Phase)
	assert.Equal(t, telemetry.MissionLifecycleSchemaVersion, got[0].SchemaVersion)
}

func TestRecordMissionLifecycleReportsAnUnwritableLedger(t *testing.T) {
	root := t.TempDir()
	// a file where the memory directory must be makes the ledger unwritable
	require.NoError(t, os.WriteFile(filepath.Join(root, "memory"), []byte("x"), 0o600))
	err := recordMissionLifecycle(root, "m-1",
		domain.LifecycleMarkersForStart(domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseBootstrap}))
	require.Error(t, err)
}

func TestWarnMissionLifecycleDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() { warnMissionLifecycle("m-1", errors.New("boom")) })
}
