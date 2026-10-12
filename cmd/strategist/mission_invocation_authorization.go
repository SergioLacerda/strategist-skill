package main

import (
	"fmt"
	"path/filepath"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/mission/refinement"
)

func authorizeSniperEntry(input missionadapter.InvocationBuildInput) error {
	if input.Role != "sniper" || input.Slot != string(domain.SlotExecution) {
		return nil
	}
	_, status, err := loadMission(input.Root, input.MissionID)
	if err != nil {
		return fmt.Errorf("role_invocation_failed: %w", err)
	}
	if status.Stage == domain.StageShort && status.StageFeat == "critical_hit" {
		return authorizeCriticalHitEntry(status)
	}
	if err := requireSniperExecutionTargets(input); err != nil {
		return err
	}
	return requireSniperGateDigest(input)
}

func requireSniperExecutionTargets(input missionadapter.InvocationBuildInput) error {
	snapshot, err := missionadapter.LoadExecutionSnapshot(input.Root, input.BasePath, input.MissionID)
	if err != nil {
		return fmt.Errorf("role_invocation_failed: %w", err)
	}
	if err := snapshot.Validate(); err != nil {
		return fmt.Errorf("role_invocation_failed: %w", err)
	}
	if len(snapshot.Groups) == 0 {
		return fmt.Errorf("role_invocation_failed: execution requires at least one approved execution target")
	}
	return nil
}

func authorizeCriticalHitEntry(status domain.MissionEngineStatus) error {
	if status.State != domain.StateExecution || !status.StageGateApproved {
		return fmt.Errorf("role_invocation_failed: Critical Hit execution requires explicit gate approval")
	}
	expected := domain.CriticalHitStageApprovalDigest(status)
	if expected == "" || status.ApprovalGatePackageDigest != expected {
		return fmt.Errorf("role_invocation_failed: Critical Hit approval evidence is missing or stale")
	}
	return nil
}

func requireSniperTargets(tasksPath string) error {
	hasTargets, err := refinement.HasDocumentationTargets(tasksPath)
	if err != nil {
		return fmt.Errorf("role_invocation_failed: %w", err)
	}
	targets, err := refinement.DocumentationTargetPaths(tasksPath)
	if err != nil {
		return fmt.Errorf("role_invocation_failed: %w", err)
	}
	if !hasTargets || len(targets) == 0 {
		return fmt.Errorf("role_invocation_failed: execution requires at least one documentation_target with an explicit backtick-quoted path")
	}
	return nil
}

func requireSniperGateDigest(input missionadapter.InvocationBuildInput) error {
	_, status, err := loadMission(input.Root, input.MissionID)
	if err != nil {
		return fmt.Errorf("role_invocation_failed: %w", err)
	}
	digest, err := handoff.PackageDigest(filepath.Join(input.BasePath, "refined", input.MissionID))
	if err != nil {
		return fmt.Errorf("role_invocation_failed: %w", err)
	}
	if status.ApprovalGatePackageDigest == "" || digest != status.ApprovalGatePackageDigest {
		return fmt.Errorf("role_invocation_failed: approval_gate_package_changed: accept the current refined package before Sniper execution")
	}
	return nil
}

// authorizeArchivistEntry requires the Ranger-to-Archivist handoff outcome
// before an Archivist refinement request is issued; other roles pass.
func authorizeArchivistEntry(input missionadapter.InvocationBuildInput) error {
	if input.Role != "archivist" || input.Slot != string(domain.SlotRefinement) {
		return nil
	}
	if _, err := missionruntime.AuthorizeRangerToArchivist(input.Root, input.BasePath, input.MissionID); err != nil {
		return fmt.Errorf("role_invocation_failed: %w", err)
	}
	return nil
}
