package mission

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// recordLifecycleBestEffort writes lifecycle markers after the state they
// describe is durable. A failure is reported through LifecycleWarn and never
// returned: the mission transition must not fail or roll back because
// telemetry could not be written (the reader flags the resulting gap).
func recordLifecycleBestEffort(deps LifecycleDependencies, root, missionID string, markers []domain.LifecycleMarker) {
	if deps.RecordLifecycle == nil || len(markers) == 0 {
		return
	}
	if err := deps.RecordLifecycle(root, missionID, markers); err != nil && deps.LifecycleWarn != nil {
		deps.LifecycleWarn(missionID, err)
	}
}
