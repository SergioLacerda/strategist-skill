package application

import (
	"errors"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryLifecycle struct {
	records   []LifecycleRecord
	readErr   error
	appendErr error
}

func (m *memoryLifecycle) read(_, missionID string) ([]LifecycleRecord, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	var out []LifecycleRecord
	for _, r := range m.records {
		if r.MissionID == missionID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memoryLifecycle) appendAll(_ string, recs []LifecycleRecord) (int, error) {
	if m.appendErr != nil {
		return 0, m.appendErr
	}
	seen := map[string]bool{}
	for _, r := range m.records {
		seen[r.MarkerID] = true
	}
	n := 0
	for _, r := range recs {
		if !seen[r.MarkerID] {
			m.records = append(m.records, r)
			seen[r.MarkerID] = true
			n++
		}
	}
	return n, nil
}

func (m *memoryLifecycle) ports(now time.Time) LifecyclePorts {
	return LifecyclePorts{Read: m.read, Append: m.appendAll, Now: func() time.Time { return now }}
}

func marker(kind domain.LifecycleMarkerKind, phase domain.PipelinePhase, reason domain.LifecycleEndReason) domain.LifecycleMarker {
	return domain.LifecycleMarker{Kind: kind, Phase: phase, EndReason: reason}
}

var fixedNow = time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)

func TestRecordLifecycleMarkersNumbersRunsAcrossReentry(t *testing.T) {
	store := &memoryLifecycle{}
	p := store.ports(fixedNow)
	steps := [][]domain.LifecycleMarker{
		{marker(domain.MarkerMissionStarted, domain.PhaseBootstrap, ""), marker(domain.MarkerPhaseStarted, domain.PhaseBootstrap, "")},
		{marker(domain.MarkerPhaseStarted, domain.PhaseRefinement, "")},
		{marker(domain.MarkerPhaseEnded, domain.PhaseRefinement, domain.EndCompleted), marker(domain.MarkerPhaseStarted, domain.PhaseApprovalGate, "")},
		{marker(domain.MarkerPhaseEnded, domain.PhaseApprovalGate, domain.EndRevisionRequested), marker(domain.MarkerPhaseStarted, domain.PhaseRefinement, "")},
	}
	for _, step := range steps {
		_, err := RecordLifecycleMarkers("root", "m-1", step, p)
		require.NoError(t, err)
	}
	var runs []string
	for _, r := range store.records {
		if r.Phase == "REFINEMENT" && r.MarkerKind == "phase_started" {
			runs = append(runs, r.PhaseRunID)
		}
	}
	assert.Equal(t, []string{"r1", "r2"}, runs)
}

func TestRecordLifecycleMarkersReplayIsANoOp(t *testing.T) {
	store := &memoryLifecycle{}
	p := store.ports(fixedNow)
	batch := []domain.LifecycleMarker{marker(domain.MarkerPhaseStarted, domain.PhaseIntake, "")}

	n, err := RecordLifecycleMarkers("root", "m-1", batch, p)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = RecordLifecycleMarkers("root", "m-1", batch, p)
	require.NoError(t, err)
	assert.Equal(t, 0, n, "an open run means the start is a replay")
	assert.Len(t, store.records, 1)
}

func TestRecordLifecycleMarkersEndWithoutOpenRunIsSkipped(t *testing.T) {
	store := &memoryLifecycle{}
	n, err := RecordLifecycleMarkers("root", "m-1",
		[]domain.LifecycleMarker{marker(domain.MarkerPhaseEnded, domain.PhaseDiscovery, domain.EndCompleted)}, store.ports(fixedNow))
	require.NoError(t, err)
	assert.Zero(t, n)
	assert.Empty(t, store.records)
}

func TestRecordLifecycleMarkersEndAndStartInOneBatchShareTheInstant(t *testing.T) {
	store := &memoryLifecycle{}
	p := store.ports(fixedNow)
	_, err := RecordLifecycleMarkers("root", "m-1", []domain.LifecycleMarker{marker(domain.MarkerPhaseStarted, domain.PhaseIntake, "")}, p)
	require.NoError(t, err)
	_, err = RecordLifecycleMarkers("root", "m-1", []domain.LifecycleMarker{
		marker(domain.MarkerPhaseEnded, domain.PhaseIntake, domain.EndCompleted),
		marker(domain.MarkerPhaseStarted, domain.PhaseDiscovery, ""),
	}, p)
	require.NoError(t, err)
	require.Len(t, store.records, 3)
	assert.Equal(t, "r1", store.records[1].PhaseRunID)
	assert.Equal(t, store.records[1].ObservedAt, store.records[2].ObservedAt)
	assert.Equal(t, "2026-10-10T10:00:00Z", store.records[1].ObservedAt)
}

func TestRecordLifecycleMarkersMissionEndedCarriesReasonAndStage(t *testing.T) {
	store := &memoryLifecycle{}
	m := domain.LifecycleMarker{Kind: domain.MarkerMissionEnded, Phase: domain.PhaseDone, EndReason: domain.EndRejected, Stage: domain.StageShort}
	_, err := RecordLifecycleMarkers("root", "m-1", []domain.LifecycleMarker{m}, store.ports(fixedNow))
	require.NoError(t, err)
	require.Len(t, store.records, 1)
	assert.Equal(t, "rejected", store.records[0].EndReason)
	assert.Equal(t, "SHORT", store.records[0].Stage)
	assert.Empty(t, store.records[0].PhaseRunID)
}

func TestRecordLifecycleMarkersPropagatesPortErrors(t *testing.T) {
	batch := []domain.LifecycleMarker{marker(domain.MarkerPhaseStarted, domain.PhaseIntake, "")}

	_, err := RecordLifecycleMarkers("root", "m-1", batch, (&memoryLifecycle{readErr: errors.New("boom")}).ports(fixedNow))
	require.ErrorContains(t, err, "boom")

	_, err = RecordLifecycleMarkers("root", "m-1", batch, (&memoryLifecycle{appendErr: errors.New("disk")}).ports(fixedNow))
	require.ErrorContains(t, err, "disk")

	_, err = RecordLifecycleMarkers("root", "m-1", batch, LifecyclePorts{})
	require.Error(t, err)
	_, err = RecordLifecycleMarkers("root", "", batch, (&memoryLifecycle{}).ports(fixedNow))
	require.Error(t, err)
}

func TestRecordLifecycleMarkersWithNoMarkersTouchesNothing(t *testing.T) {
	n, err := RecordLifecycleMarkers("root", "m-1", nil, LifecyclePorts{})
	require.NoError(t, err)
	assert.Zero(t, n)
}
