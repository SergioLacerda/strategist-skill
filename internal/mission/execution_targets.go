package mission

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// UnsupportedExecutionSideQuestError remains as a compatibility error type for
// callers that imported the first-wave bridge. New resolution reports the
// generic execution_resolver_not_registered error instead.
type UnsupportedExecutionSideQuestError struct{ ID string }

func (e UnsupportedExecutionSideQuestError) Error() string {
	return fmt.Sprintf("execution_target_unsupported_side_quest: %s is not supported by the current execution bridge", e.ID)
}

// ResolveExecutionSnapshot assembles the immutable, provider-neutral
// execution contract. Accepted side quests are enumerated generically and
// dispatched through the resolver registry by domain kind and origin.
func ResolveExecutionSnapshot(strategistRoot, basePath, missionID string, explicit []string, now time.Time) (domain.ExecutionSnapshot, error) {
	registry, err := builtinResolverRegistry(strategistRoot, basePath, now)
	if err != nil {
		return domain.ExecutionSnapshot{}, err
	}
	explicitGroups, err := resolveExplicitExecutionGroups(registry, missionID, explicit)
	if err != nil {
		return domain.ExecutionSnapshot{}, err
	}
	sideQuestGroups, deferred, err := resolveAcceptedSideQuestGroups(registry, strategistRoot, missionID)
	if err != nil {
		return domain.ExecutionSnapshot{}, err
	}
	groups := explicitGroups
	groups = append(groups, sideQuestGroups...)
	if len(groups) == 0 {
		return domain.ExecutionSnapshot{Version: domain.ExecutionSnapshotVersion, MissionID: missionID, Deferred: deferred}, nil
	}
	snapshot := domain.ExecutionSnapshot{Version: domain.ExecutionSnapshotVersion, MissionID: missionID, Groups: groups, Deferred: deferred}
	if err := snapshot.Validate(); err != nil {
		return domain.ExecutionSnapshot{}, fmt.Errorf("execution_snapshot_invalid: %w", err)
	}
	return snapshot, nil
}

func resolveExplicitExecutionGroups(registry *ResolverRegistry, missionID string, explicit []string) ([]domain.ExecutionGroup, error) {
	paths, err := normalizeExplicitExecutionPaths(explicit)
	if err != nil {
		return nil, err
	}
	if len(paths) > 0 {
		resolution, err := registry.Resolve(context.Background(), ExecutionResolveInput{
			MissionID: missionID, Kind: domain.ExecutionTargetKindDocumentation, Origin: domain.ExecutionTargetOriginRefinedPackage,
			Payload: map[string]any{"paths": paths},
		})
		if err != nil {
			return nil, err
		}
		return resolution.Groups, nil
	}
	return nil, nil
}

func normalizeExplicitExecutionPaths(explicit []string) ([]string, error) {
	seenPaths := make(map[string]struct{}, len(explicit))
	paths := make([]string, 0, len(explicit))
	for _, path := range explicit {
		if strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("execution_target_invalid: target path is empty")
		}
		if _, exists := seenPaths[path]; exists {
			continue
		}
		seenPaths[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths, nil
}

// resolveAcceptedSideQuestGroups resolves every accepted side quest whose kind
// has a registered resolver. A kind without one (Treasure Chest maintenance is
// provider-owned through JEWELER, not by Sniper) is returned as an explicit
// deferred notice instead of failing the snapshot; every other resolution error
// still fails closed.
func resolveAcceptedSideQuestGroups(registry *ResolverRegistry, strategistRoot, missionID string) ([]domain.ExecutionGroup, []domain.DeferredIntent, error) {
	records, err := LoadMissionSideQuests(strategistRoot, missionID)
	if err != nil {
		return nil, nil, err
	}
	groups := make([]domain.ExecutionGroup, 0, len(records))
	var deferred []domain.DeferredIntent
	for _, record := range acceptedRecords(records) {
		resolved, resolveErr := resolveAcceptedRecord(registry, missionID, record)
		if resolveErr != nil {
			return nil, nil, resolveErr
		}
		groups = append(groups, resolved.groups...)
		deferred = append(deferred, resolved.deferred...)
	}
	return groups, deferred, nil
}

func acceptedRecords(records []AcceptedSideQuest) []AcceptedSideQuest {
	accepted := make([]AcceptedSideQuest, 0, len(records))
	for _, record := range records {
		if record.Decision == SideQuestAccepted {
			accepted = append(accepted, record)
		}
	}
	return accepted
}

// acceptedResolution is what one accepted record contributes to the snapshot:
// resolver groups, or a deferred notice when its kind has no resolver.
type acceptedResolution struct {
	groups   []domain.ExecutionGroup
	deferred []domain.DeferredIntent
}

func resolveAcceptedRecord(registry *ResolverRegistry, missionID string, record AcceptedSideQuest) (acceptedResolution, error) {
	kind := executionKindForSideQuest(record.Kind)
	if !registry.Has(kind, domain.ExecutionTargetOriginAcceptedSideQuest) {
		return acceptedResolution{deferred: []domain.DeferredIntent{{
			SideQuestID: record.SideQuestID, Kind: record.Kind,
			Origin: domain.ExecutionTargetOriginAcceptedSideQuest, Reason: domain.DeferredReasonNoExecutionResolver,
		}}}, nil
	}
	resolution, err := registry.Resolve(context.Background(), ExecutionResolveInput{
		MissionID: missionID, SideQuestID: record.SideQuestID, Kind: kind,
		Origin: domain.ExecutionTargetOriginAcceptedSideQuest,
	})
	if err != nil {
		return acceptedResolution{}, err
	}
	return acceptedResolution{groups: resolution.Groups}, nil
}

func executionKindForSideQuest(kind string) string {
	switch kind {
	case SideQuestKindOAADR:
		return domain.ExecutionTargetKindADR
	case SideQuestKindOARunbook:
		return "runbook"
	default:
		return kind
	}
}

// ResolveExecutionTargets projects the hierarchical snapshot for first-wave
// callers. New callers must retain the snapshot so group atomicity is not lost.
func ResolveExecutionTargets(strategistRoot, basePath, missionID string, explicit []string, now time.Time) ([]domain.ExecutionTarget, error) {
	snapshot, err := ResolveExecutionSnapshot(strategistRoot, basePath, missionID, explicit, now)
	if err != nil {
		return nil, err
	}
	targets := make([]domain.ExecutionTarget, 0)
	for _, group := range snapshot.Groups {
		for _, artifact := range group.Artifacts {
			targets = append(targets, domain.ExecutionTarget{Path: artifact.Path, Kind: artifact.Kind, Origin: group.Origin, SideQuestID: group.SideQuestID})
		}
	}
	return targets, nil
}

// ExecutionTargetPaths projects the immutable target snapshot to the path
// representation consumed by existing telemetry ledgers.
func ExecutionTargetPaths(targets []domain.ExecutionTarget) []string {
	paths := make([]string, 0, len(targets))
	for _, target := range targets {
		paths = append(paths, target.Path)
	}
	return paths
}

// hasExecutionResolver reports whether an accepted side quest of the given
// record kind can be resolved into execution groups. It asks the same builtin
// registry execution uses, so the analysis-only guard and the resolver cannot
// disagree.
func hasExecutionResolver(recordKind string) bool {
	registry, err := builtinResolverRegistry("", "", time.Time{})
	if err != nil {
		return true
	}
	return registry.Has(executionKindForSideQuest(recordKind), domain.ExecutionTargetOriginAcceptedSideQuest)
}
