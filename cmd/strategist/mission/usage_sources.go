package mission

import (
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// usageSources are the read-side inputs shared by `mission view` and
// `mission usage`, so both build the same timeline.
type usageSources struct {
	lifecycle     []telemetry.MissionLifecycleRecord
	lifecycleErr  error
	usage         []telemetry.MissionUsageRecord
	usageErr      error
	invocationIDs []string
}

// loadUsageSources reads the lifecycle ledger, the usage ledger and the ids of
// the invocations Strategist issued for the mission. An unreadable invocation
// store only drops the coverage counters; it never fails the read.
func loadUsageSources(root, missionID string) usageSources {
	var s usageSources
	s.lifecycle, s.lifecycleErr = telemetry.ReadMissionLifecycleFor(telemetry.MissionLifecyclePath(root), missionID)
	s.usage, s.usageErr = telemetry.ReadMissionUsageFor(telemetry.MissionUsagePath(root), missionID)
	listing, err := livemission.NewInvocationStore(root).List(missionID)
	if err != nil {
		return s
	}
	for _, r := range listing.Requests {
		s.invocationIDs = append(s.invocationIDs, r.RequestID)
	}
	return s
}
