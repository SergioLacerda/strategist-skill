package mission

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// usageAppender returns the append port: it writes the record to the usage
// ledger and, for a new record, exports its event. The ledger write is the
// source of truth; a lenient sink swallows an export failure and a strict one
// returns it with the fact that the record was written.
func usageAppender(sinkFor func() telemetry.EventSink) func(string, application.MissionUsageRecord) (bool, error) {
	return func(root string, r application.MissionUsageRecord) (bool, error) {
		rec := toTelemetryUsage(r)
		rec.RecordID = telemetry.MissionUsageRecordID(rec)
		appended, err := telemetry.AppendMissionUsageRecord(telemetry.MissionUsagePath(root), rec)
		if err != nil {
			return false, fmt.Errorf("append mission usage: %w", err)
		}
		if appended {
			if err := exportUsageEvent(sinkFor, rec); err != nil {
				return true, err
			}
		}
		return appended, nil
	}
}

// exportUsageEvent emits the event of a record that was just written. With no
// sink configured nothing is exported.
func exportUsageEvent(sinkFor func() telemetry.EventSink, rec telemetry.MissionUsageRecord) error {
	if sinkFor == nil {
		return nil
	}
	if err := sinkFor().Emit(context.Background(), telemetry.NewMissionUsageEvent(rec)); err != nil {
		return fmt.Errorf("usage recorded in the ledger, telemetry export failed: %w", err)
	}
	return nil
}

func toTelemetryUsage(r application.MissionUsageRecord) telemetry.MissionUsageRecord {
	return telemetry.MissionUsageRecord{
		SchemaVersion: telemetry.MissionUsageSchemaVersion, EventID: r.EventID, MissionID: r.MissionID,
		Phase: r.Phase, PhaseRunID: r.PhaseRunID, Role: r.Role, InvocationID: r.InvocationID,
		Attempt: r.Attempt, UsageScope: r.UsageScope, SessionID: r.SessionID, RecordKind: r.RecordKind,
		TokensInput: r.TokensInput, TokensOutput: r.TokensOutput, TokensCache: r.TokensCache, TokensReasoning: r.TokensReasoning,
		Provider: r.Provider, Model: r.Model, MeasurementSource: r.MeasurementSource, MeasurementQuality: r.MeasurementQuality,
		DedupUnavailable: r.DedupUnavailable, Calls: r.Calls, ObservedAt: r.ObservedAt,
	}
}
