package domain

import "fmt"

// SnapshotFromTargets preserves the first-wave durable format as a structural
// migration. It never reads tasks.md or derives additional targets.
func SnapshotFromTargets(missionID string, targets []ExecutionTarget) (ExecutionSnapshot, error) {
	groups := make([]ExecutionGroup, 0, len(targets))
	for index, target := range targets {
		group, err := executionGroupFromTarget(index, target)
		if err != nil {
			return ExecutionSnapshot{}, err
		}
		groups = append(groups, group)
	}
	snapshot := ExecutionSnapshot{Version: ExecutionSnapshotVersion, MissionID: missionID, Groups: groups}
	if len(groups) == 0 {
		return snapshot, nil
	}
	if err := snapshot.Validate(); err != nil {
		return ExecutionSnapshot{}, err
	}
	return snapshot, nil
}

func executionGroupFromTarget(index int, target ExecutionTarget) (ExecutionGroup, error) {
	if err := target.Validate(); err != nil {
		return ExecutionGroup{}, fmt.Errorf("execution_snapshot_invalid: %w", err)
	}
	id := fmt.Sprintf("legacy-group-%03d", index+1)
	metadata := map[string]string(nil)
	if target.Origin == ExecutionTargetOriginRefinedPackage {
		metadata = map[string]string{"completion_source": "refined_tasks"}
	}
	artifact := ExecutionArtifact{ID: id + "-artifact", Role: ExecutionArtifactRolePrimary, Path: target.Path, Kind: target.Kind, Metadata: metadata}
	return ExecutionGroup{
		ID: id, Resolver: ResolverKey{Kind: target.Kind, Origin: target.Origin},
		SideQuestID: target.SideQuestID, Kind: target.Kind, Origin: target.Origin,
		CompletionPolicy: ExecutionCompletionAllRequired, Artifacts: []ExecutionArtifact{artifact},
	}, nil
}
