package application

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// LifecycleRecord is one lifecycle ledger line as the application layer sees
// it; adapters map it to the telemetry record.
type LifecycleRecord struct {
	MarkerID, MarkerKind, MissionID, Phase string
	PhaseRunID, EndReason, Stage           string
	ObservedAt                             string
}

// LifecyclePorts are the ledger and clock boundaries of marker recording.
type LifecyclePorts struct {
	Read   func(root, missionID string) ([]LifecycleRecord, error)
	Append func(root string, recs []LifecycleRecord) (int, error)
	Now    func() time.Time
}

// RecordLifecycleMarkers turns domain markers into ledger records and appends
// them. It assigns deterministic run numbers from the mission's own history:
// a start while the phase has an open run is a replay and is dropped, and an
// end without an open run is dropped (the reader reports the resulting gap).
// It reports how many records were appended.
func RecordLifecycleMarkers(root, missionID string, markers []domain.LifecycleMarker, ports LifecyclePorts) (int, error) {
	if len(markers) == 0 {
		return 0, nil
	}
	if err := validateLifecyclePorts(missionID, ports); err != nil {
		return 0, err
	}
	history, err := ports.Read(root, missionID)
	if err != nil {
		return 0, fmt.Errorf("mission lifecycle: read ledger: %w", err)
	}
	batch := buildLifecycleBatch(missionID, markers, ports.Now().UTC().Format(time.RFC3339), history)
	if len(batch) == 0 {
		return 0, nil
	}
	n, err := ports.Append(root, batch)
	if err != nil {
		return 0, fmt.Errorf("mission lifecycle: append ledger: %w", err)
	}
	return n, nil
}

func validateLifecyclePorts(missionID string, ports LifecyclePorts) error {
	if missionID == "" {
		return errors.New("mission lifecycle: mission id is required")
	}
	if ports.Read == nil || ports.Append == nil || ports.Now == nil {
		return errors.New("mission lifecycle: ledger and clock ports are required")
	}
	return nil
}

func buildLifecycleBatch(missionID string, markers []domain.LifecycleMarker, at string, history []LifecycleRecord) []LifecycleRecord {
	known := append([]LifecycleRecord(nil), history...)
	var batch []LifecycleRecord
	for _, m := range markers {
		rec, ok := lifecycleRecordFor(missionID, m, at, known)
		if !ok {
			continue
		}
		batch = append(batch, rec)
		known = append(known, rec)
	}
	return batch
}

func lifecycleRecordFor(missionID string, m domain.LifecycleMarker, at string, history []LifecycleRecord) (LifecycleRecord, bool) {
	rec := LifecycleRecord{
		MarkerKind: string(m.Kind), MissionID: missionID, Phase: string(m.Phase),
		EndReason: string(m.EndReason), Stage: string(m.Stage), ObservedAt: at,
	}
	switch m.Kind {
	case domain.MarkerPhaseStarted:
		open, starts := phaseRuns(history, rec.Phase)
		if open != "" {
			return LifecycleRecord{}, false
		}
		rec.PhaseRunID = "r" + strconv.Itoa(starts+1)
	case domain.MarkerPhaseEnded:
		open, _ := phaseRuns(history, rec.Phase)
		if open == "" {
			return LifecycleRecord{}, false
		}
		rec.PhaseRunID = open
	case domain.MarkerMissionStarted, domain.MarkerMissionEnded:
	}
	rec.MarkerID = lifecycleMarkerID(rec)
	return rec, true
}

// phaseRuns returns the open run id of a phase (empty when none) and how many
// runs of it have started, folding the mission's history in file order.
func phaseRuns(history []LifecycleRecord, phase string) (open string, starts int) {
	for _, r := range history {
		if r.Phase == phase {
			open, starts = foldPhaseRun(r, open, starts)
		}
	}
	return open, starts
}

func foldPhaseRun(r LifecycleRecord, open string, starts int) (string, int) {
	switch {
	case r.MarkerKind == string(domain.MarkerPhaseStarted):
		return r.PhaseRunID, starts + 1
	case r.MarkerKind == string(domain.MarkerPhaseEnded) && r.PhaseRunID == open:
		return "", starts
	}
	return open, starts
}

func lifecycleMarkerID(r LifecycleRecord) string {
	run := r.PhaseRunID
	if run == "" {
		run = "-"
	}
	id := r.MissionID + "|" + r.Phase + "|" + run + "|" + r.MarkerKind
	if r.MarkerKind == string(domain.MarkerMissionEnded) {
		id += "|" + r.EndReason
	}
	return id
}
