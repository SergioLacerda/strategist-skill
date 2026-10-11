package mission

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
)

// LoadExecutionTargetSnapshot returns the immutable target set captured when
// the mission entered execution. Legacy entries without metadata are projected
// as ordinary refined-package documentation targets. A pre-entry fixture may
// still read explicit targets from tasks.md so existing current-host tests and
// non-side-quest callers retain their boundary behavior.
func LoadExecutionTargetSnapshot(root, basePath, missionID string) ([]domain.ExecutionTarget, error) {
	snapshot, err := LoadExecutionSnapshot(root, basePath, missionID)
	if err != nil {
		return nil, err
	}
	return executionTargetsFromSnapshot(snapshot), nil
}

// LoadExecutionSnapshot returns the immutable hierarchical snapshot. An
// existing entry is authoritative; tasks.md is only a pre-entry compatibility
// source for callers that have no durable execution entry yet.
func LoadExecutionSnapshot(root, basePath, missionID string) (domain.ExecutionSnapshot, error) {
	entry, err := loadExecutionEntry(executionEntryPath(root, missionID))
	if err == nil {
		return loadSnapshotFromExecutionEntry(missionID, entry)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return domain.ExecutionSnapshot{}, err
	}
	return loadSnapshotFromRefinedPackage(basePath, missionID)
}

func loadSnapshotFromExecutionEntry(missionID string, entry executionEntry) (domain.ExecutionSnapshot, error) {
	if entry.ExecutionSnapshot != nil {
		return *entry.ExecutionSnapshot, nil
	}
	snapshot, err := domain.SnapshotFromTargets(missionID, executionTargetsForEntry(&entry))
	if err != nil {
		return domain.ExecutionSnapshot{}, fmt.Errorf("load execution snapshot from legacy entry: %w", err)
	}
	return snapshot, nil
}

func loadSnapshotFromRefinedPackage(basePath, missionID string) (domain.ExecutionSnapshot, error) {
	paths, err := refinement.DocumentationTargetPaths(filepath.Join(basePath, "refined", missionID, "tasks.md"))
	if err != nil {
		return domain.ExecutionSnapshot{}, fmt.Errorf("load execution snapshot from refined package: %w", err)
	}
	snapshot, err := domain.SnapshotFromTargets(missionID, executionTargetsFromPaths(paths))
	if err != nil {
		return domain.ExecutionSnapshot{}, fmt.Errorf("load execution snapshot from refined targets: %w", err)
	}
	return snapshot, nil
}

func executionTargetsForEntry(entry *executionEntry) []domain.ExecutionTarget {
	if entry.ExecutionSnapshot != nil {
		return executionTargetsFromSnapshot(*entry.ExecutionSnapshot)
	}
	if len(entry.TargetMetadata) > 0 {
		return append([]domain.ExecutionTarget(nil), entry.TargetMetadata...)
	}
	return executionTargetsFromPaths(entry.Targets)
}

func executionTargetsFromSnapshot(snapshot domain.ExecutionSnapshot) []domain.ExecutionTarget {
	targets := make([]domain.ExecutionTarget, 0)
	for _, group := range snapshot.Groups {
		for _, artifact := range group.Artifacts {
			targets = append(targets, domain.ExecutionTarget{Path: artifact.Path, Kind: artifact.Kind, Origin: group.Origin, SideQuestID: group.SideQuestID})
		}
	}
	return targets
}

func executionTargetsFromPaths(paths []string) []domain.ExecutionTarget {
	targets := make([]domain.ExecutionTarget, 0, len(paths))
	for _, path := range paths {
		targets = append(targets, domain.ExecutionTarget{
			Path: path, Kind: domain.ExecutionTargetKindDocumentation,
			Origin: domain.ExecutionTargetOriginRefinedPackage,
		})
	}
	return targets
}
