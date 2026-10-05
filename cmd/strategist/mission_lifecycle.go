package main

import (
	"fmt"
	"os"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// recordMissionLifecycle binds the application marker recorder to the
// telemetry lifecycle ledger under the runtime root.
func recordMissionLifecycle(root, missionID string, markers []domain.LifecycleMarker) error {
	path := telemetry.MissionLifecyclePath(root)
	_, err := application.RecordLifecycleMarkers(root, missionID, markers, application.LifecyclePorts{
		Read: func(_, id string) ([]application.LifecycleRecord, error) {
			recs, err := telemetry.ReadMissionLifecycleFor(path, id)
			if err != nil {
				return nil, fmt.Errorf("read lifecycle ledger: %w", err)
			}
			return lifecycleToApplication(recs), nil
		},
		Append: func(_ string, recs []application.LifecycleRecord) (int, error) {
			n, err := telemetry.AppendMissionLifecycleRecords(path, lifecycleToTelemetry(recs))
			if err != nil {
				return 0, fmt.Errorf("append lifecycle ledger: %w", err)
			}
			return n, nil
		},
		Now: time.Now,
	})
	if err != nil {
		return fmt.Errorf("record mission lifecycle: %w", err)
	}
	return nil
}

// warnMissionLifecycle makes a marker write failure observable without
// failing the mission command that triggered it.
func warnMissionLifecycle(missionID string, err error) {
	//nolint:errcheck // best-effort diagnostic; the mission transition already succeeded
	fmt.Fprintf(os.Stderr, "[Strategist] warning=mission_lifecycle_marker_write_failed mission_id=%s: %v\n", missionID, err)
}

func lifecycleToApplication(recs []telemetry.MissionLifecycleRecord) []application.LifecycleRecord {
	out := make([]application.LifecycleRecord, 0, len(recs))
	for _, r := range recs {
		out = append(out, application.LifecycleRecord{
			MarkerID: r.MarkerID, MarkerKind: r.MarkerKind, MissionID: r.MissionID, Phase: r.Phase,
			PhaseRunID: r.PhaseRunID, EndReason: r.EndReason, Stage: r.Stage, ObservedAt: r.ObservedAt,
		})
	}
	return out
}

func lifecycleToTelemetry(recs []application.LifecycleRecord) []telemetry.MissionLifecycleRecord {
	out := make([]telemetry.MissionLifecycleRecord, 0, len(recs))
	for _, r := range recs {
		out = append(out, telemetry.MissionLifecycleRecord{
			SchemaVersion: telemetry.MissionLifecycleSchemaVersion,
			MarkerID:      r.MarkerID, MarkerKind: r.MarkerKind, MissionID: r.MissionID, Phase: r.Phase,
			PhaseRunID: r.PhaseRunID, EndReason: r.EndReason, Stage: r.Stage, ObservedAt: r.ObservedAt,
		})
	}
	return out
}
