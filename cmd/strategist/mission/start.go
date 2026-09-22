package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// LifecycleDependencies injects the runtime-specific persistence, path
// resolution and output encoding used by start, status, submit and context.
type LifecycleDependencies struct {
	RootFlag          string
	RequireMissionID  func(string) error
	ResolveBasePath   func(string) (string, string, error)
	RequireNoExisting func(string, string) error
	Save              func(string, domain.MissionEngineStatus) error
	Load              func(string, string) (*domain.MissionEngine, domain.MissionEngineStatus, error)
	WriteResult       func(*cobra.Command, bool, any) error
}

// lifecycleFlags holds the --root/--mission-id/--json values shared by the
// lifecycle subcommands.
type lifecycleFlags struct {
	root, missionID string
	asJSON          bool
}

func bindLifecycleFlags(cmd *cobra.Command, deps LifecycleDependencies) *lifecycleFlags {
	f := &lifecycleFlags{}
	cmd.Flags().StringVar(&f.root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().StringVar(&f.missionID, "mission-id", "", "mission identifier (required)")
	cmd.Flags().BoolVar(&f.asJSON, "json", false, "emit machine-readable JSON")
	return f
}

// NewStart builds `mission start`.
func NewStart(deps LifecycleDependencies) *cobra.Command {
	cmd := &cobra.Command{Use: "start", Short: "Start a mission"}
	f := bindLifecycleFlags(cmd, deps)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunStart(cmd, deps, f.root, f.missionID, f.asJSON) }
	return cmd
}

// RunStart creates and persists a new mission through domain.StartMission.
func RunStart(cmd *cobra.Command, deps LifecycleDependencies, rootInput, missionID string, asJSON bool) error {
	if err := deps.RequireMissionID(missionID); err != nil {
		return err
	}
	root, _, err := deps.ResolveBasePath(rootInput)
	if err != nil {
		return fmt.Errorf("mission start: %w", err)
	}
	if err := deps.RequireNoExisting(root, missionID); err != nil {
		return err
	}
	engine, status, err := domain.StartMission(domain.MissionStartRequest{MissionID: missionID})
	if err != nil {
		return fmt.Errorf("mission start: %w", err)
	}
	if err := deps.Save(root, engine.Status()); err != nil {
		return fmt.Errorf("mission start: %w", err)
	}
	return deps.WriteResult(cmd, asJSON, status)
}
