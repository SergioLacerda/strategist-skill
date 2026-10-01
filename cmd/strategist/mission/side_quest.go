package mission

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/spf13/cobra"
)

// NewAcceptSideQuest builds `mission accept-side-quest`.
func NewAcceptSideQuest(deps LifecycleDependencies) *cobra.Command {
	return newSideQuestDecision(deps, "accept-side-quest", livemission.SideQuestAccepted,
		"Record the user's acceptance of the mission's OA-ADR side quest at the Approval Gate")
}

// NewDeclineSideQuest builds `mission decline-side-quest`.
func NewDeclineSideQuest(deps LifecycleDependencies) *cobra.Command {
	return newSideQuestDecision(deps, "decline-side-quest", livemission.SideQuestDeclined,
		"Record the user's decline of the mission's OA-ADR side quest at the Approval Gate")
}

func newSideQuestDecision(deps LifecycleDependencies, use, decision, short string) *cobra.Command {
	var sideQuest string
	cmd := &cobra.Command{Use: use, Short: short}
	f := bindLifecycleFlags(cmd, deps)
	cmd.Flags().StringVar(&sideQuest, "side-quest", "", "side quest id, OA-ADR-<mission_id> (required)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runSideQuestDecision(cmd, deps, f, sideQuest, decision)
	}
	return cmd
}

func runSideQuestDecision(cmd *cobra.Command, deps LifecycleDependencies, f *lifecycleFlags, sideQuest, decision string) error {
	name := cmd.Name()
	if err := deps.RequireMissionID(f.missionID); err != nil {
		return err
	}
	root, basePath, err := deps.ResolveBasePath(f.root)
	if err != nil {
		return fmt.Errorf("mission %s: %w", name, err)
	}
	canonical, err := configuredADRPath(deps, root)
	if err != nil {
		return fmt.Errorf("mission %s: %w", name, err)
	}
	var rec livemission.AcceptedSideQuest
	err = withMissionLock(deps, root, f.missionID, func() error {
		var lockedErr error
		rec, lockedErr = decideLocked(deps, root, basePath, canonical, f.missionID, sideQuest, decision)
		return lockedErr
	})
	if err != nil {
		return fmt.Errorf("mission %s: %w", name, err)
	}
	return deps.WriteResult(cmd, f.asJSON, rec)
}

func configuredADRPath(deps LifecycleDependencies, root string) (string, error) {
	if deps.ADRCanonicalPath == nil {
		return "", nil
	}
	return deps.ADRCanonicalPath(root)
}

// ADRTargetResult is the JSON/text result of `mission adr-target`.
type ADRTargetResult struct {
	MissionID   string `json:"mission_id"`
	SideQuestID string `json:"side_quest_id"`
	Path        string `json:"path"`
}

// NewADRTarget builds `mission adr-target`.
func NewADRTarget(deps LifecycleDependencies) *cobra.Command {
	var slug string
	cmd := &cobra.Command{Use: "adr-target", Short: "Resolve and reserve the accepted OA-ADR destination at materialization time"}
	f := bindLifecycleFlags(cmd, deps)
	cmd.Flags().StringVar(&slug, "slug", "", "title slug used when the canonical ADR directory is numbered (default: mission id)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return runADRTarget(cmd, deps, f, slug) }
	return cmd
}

func runADRTarget(cmd *cobra.Command, deps LifecycleDependencies, f *lifecycleFlags, slug string) error {
	if err := deps.RequireMissionID(f.missionID); err != nil {
		return err
	}
	root, basePath, err := deps.ResolveBasePath(f.root)
	if err != nil {
		return fmt.Errorf("mission adr-target: %w", err)
	}
	var path string
	err = withMissionLock(deps, root, f.missionID, func() error {
		_, status, loadErr := deps.Load(root, f.missionID)
		if loadErr != nil {
			return loadErr
		}
		path, loadErr = reserveTarget(root, basePath, f.missionID, slug, status.State)
		return loadErr
	})
	if err != nil {
		return fmt.Errorf("mission adr-target: %w", err)
	}
	return deps.WriteResult(cmd, f.asJSON, ADRTargetResult{MissionID: f.missionID, SideQuestID: livemission.OAADRSideQuestID(f.missionID), Path: path})
}

func reserveTarget(root, basePath, missionID, slug string, state domain.MissionState) (string, error) {
	path, err := livemission.ReserveADRTarget(livemission.ADRTargetRequest{
		StrategistRoot: root, BasePath: basePath, MissionID: missionID, Slug: slug, State: state, Now: time.Now(),
	})
	if err != nil {
		return "", fmt.Errorf("reserve target: %w", err)
	}
	return path, nil
}

func decideLocked(deps LifecycleDependencies, root, basePath, canonical, missionID, sideQuest, decision string) (livemission.AcceptedSideQuest, error) {
	_, status, err := deps.Load(root, missionID)
	if err != nil {
		return livemission.AcceptedSideQuest{}, err
	}
	rec, err := livemission.DecideSideQuest(livemission.SideQuestDecision{
		StrategistRoot: root, BasePath: basePath, MissionID: missionID, SideQuestID: sideQuest,
		Decision: decision, State: status.State, CanonicalPath: canonical, Now: time.Now(),
	})
	if err != nil {
		return livemission.AcceptedSideQuest{}, fmt.Errorf("record decision: %w", err)
	}
	return rec, nil
}
