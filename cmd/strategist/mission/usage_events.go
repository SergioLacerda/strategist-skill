package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// sourcesFromEvents replaces the lifecycle and usage records with the ones
// rebuilt from an exported event stream, keeping the invocation ids Strategist
// issued (they are not part of the stream). The same builders then produce the
// report, so a valid stream reconstructs the same figures as the ledgers.
func sourcesFromEvents(base usageSources, path, missionID string) (usageSources, error) {
	events, malformed, err := telemetry.ReadEventLines(path)
	if err != nil {
		return usageSources{}, fmt.Errorf("read event stream: %w", err)
	}
	base.lifecycle, base.usage = recordsFromEvents(events, missionID)
	base.lifecycleErr, base.usageErr = nil, nil
	if malformed > 0 {
		base.usageErr = fmt.Errorf("%d unreadable line(s) in the event stream", malformed)
	}
	return base, nil
}

// recordsFromEvents decodes the lifecycle and usage events of one mission.
func recordsFromEvents(events []telemetry.Event, missionID string) ([]telemetry.MissionLifecycleRecord, []telemetry.MissionUsageRecord) {
	var lifecycle []telemetry.MissionLifecycleRecord
	var usage []telemetry.MissionUsageRecord
	for _, e := range events {
		if rec, ok := telemetry.MissionLifecycleRecordFromEvent(e); ok && rec.MissionID == missionID {
			lifecycle = append(lifecycle, rec)
		}
		if rec, ok := telemetry.MissionUsageRecordFromEvent(e); ok && rec.MissionID == missionID {
			usage = append(usage, rec)
		}
	}
	return lifecycle, usage
}
