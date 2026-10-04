package mission

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/application"
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
		var lockedErr error
		path, lockedErr = resolveADRTarget(deps, root, basePath, f.missionID, slug)
		if lockedErr != nil {
			return fmt.Errorf("reserve ADR target: %w", lockedErr)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("mission adr-target: %w", err)
	}
	return deps.WriteResult(cmd, f.asJSON, ADRTargetResult{MissionID: f.missionID, SideQuestID: livemission.OAADRSideQuestID(f.missionID), Path: path})
}

func resolveADRTarget(deps LifecycleDependencies, root, basePath, missionID, slug string) (string, error) {
	path, err := application.ReserveADRTarget(application.ADRTargetRequest{
		StrategistRoot: root, BasePath: basePath, MissionID: missionID, Slug: slug, Now: time.Now(),
	}, func(loadRoot, loadMissionID string) (domain.MissionState, error) {
		return loadMissionState(deps, loadRoot, loadMissionID)
	}, func(request application.ADRTargetRequest, state domain.MissionState) (string, error) {
		return reserveTarget(request.StrategistRoot, request.BasePath, request.MissionID, request.Slug, state, request.Now)
	})
	if err != nil {
		return "", fmt.Errorf("reserve ADR target: %w", err)
	}
	return path, nil
}

func loadMissionState(deps LifecycleDependencies, root, missionID string) (domain.MissionState, error) {
	_, status, err := deps.Load(root, missionID)
	if err != nil {
		return "", err
	}
	return status.State, nil
}

func reserveTarget(root, basePath, missionID, slug string, state domain.MissionState, now time.Time) (string, error) {
	path, err := livemission.ReserveADRTarget(livemission.ADRTargetRequest{
		StrategistRoot: root, BasePath: basePath, MissionID: missionID, Slug: slug, State: state, Now: now,
	})
	if err != nil {
		return "", fmt.Errorf("reserve target: %w", err)
	}
	return path, nil
}

func decideLocked(deps LifecycleDependencies, root, basePath, canonical, missionID, sideQuest, decision string) (livemission.AcceptedSideQuest, error) {
	result, err := application.DecideSideQuest(application.SideQuestDecisionRequest{
		StrategistRoot: root, BasePath: basePath, MissionID: missionID, SideQuestID: sideQuest,
		Decision: decision, CanonicalPath: canonical, Now: time.Now(),
	}, func(loadRoot, loadMissionID string) (domain.MissionState, error) {
		return loadMissionState(deps, loadRoot, loadMissionID)
	}, func(request application.SideQuestDecisionRequest) (application.SideQuestDecisionResult, error) {
		rec, decideErr := livemission.DecideSideQuest(livemission.SideQuestDecision{
			StrategistRoot: request.StrategistRoot, BasePath: request.BasePath, MissionID: request.MissionID,
			SideQuestID: request.SideQuestID, Decision: request.Decision, State: request.State,
			CanonicalPath: request.CanonicalPath, Now: request.Now,
		})
		if decideErr != nil {
			return application.SideQuestDecisionResult{}, fmt.Errorf("decide side quest: %w", decideErr)
		}
		return application.SideQuestDecisionResult{
			SchemaVersion: rec.SchemaVersion, MissionID: rec.MissionID, SideQuestID: rec.SideQuestID,
			Kind: rec.Kind, Decision: rec.Decision,
			Destination:  application.DestinationRule{CanonicalPath: rec.Destination.CanonicalPath, Fallback: rec.Destination.Fallback},
			ReservedPath: rec.ReservedPath, Claimed: rec.Claimed, Sealed: rec.Sealed,
			DecidedAt: rec.DecidedAt, SealedAt: rec.SealedAt, Integrity: rec.Integrity,
		}, nil
	})
	if err != nil {
		return livemission.AcceptedSideQuest{}, fmt.Errorf("record decision: %w", err)
	}
	return livemission.AcceptedSideQuest{
		SchemaVersion: result.SchemaVersion, MissionID: result.MissionID, SideQuestID: result.SideQuestID,
		Kind: result.Kind, Decision: result.Decision,
		Destination:  livemission.DestinationRule{CanonicalPath: result.Destination.CanonicalPath, Fallback: result.Destination.Fallback},
		ReservedPath: result.ReservedPath, Claimed: result.Claimed, Sealed: result.Sealed,
		DecidedAt: result.DecidedAt, SealedAt: result.SealedAt, Integrity: result.Integrity,
	}, nil
}
