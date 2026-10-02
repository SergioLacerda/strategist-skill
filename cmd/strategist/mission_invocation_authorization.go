package main

import (
	"fmt"
	"path/filepath"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
)

func authorizeSniperEntry(input missionadapter.InvocationBuildInput) error {
	if input.Role != "sniper" || input.Slot != string(domain.SlotExecution) {
		return nil
	}
	tasksPath := filepath.Join(input.BasePath, "refined", input.MissionID, "tasks.md")
	if err := requireSniperTargets(tasksPath); err != nil {
		return err
	}
	return requireSniperGateDigest(input)
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
