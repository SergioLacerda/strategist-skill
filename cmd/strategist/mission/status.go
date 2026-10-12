package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/spf13/cobra"
)

// NewStatus builds `mission status`.
func NewStatus(deps LifecycleDependencies) *cobra.Command {
	cmd := &cobra.Command{Use: "status", Short: "Inspect a mission status"}
	f := bindLifecycleFlags(cmd, deps)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunStatus(cmd, deps, f.root, f.missionID, f.asJSON) }
	return cmd
}

// RunStatus prints the persisted status of one mission.
func RunStatus(cmd *cobra.Command, deps LifecycleDependencies, rootInput, missionID string, asJSON bool) error {
	if err := deps.RequireMissionID(missionID); err != nil {
		return err
	}
	root, _, err := deps.ResolveBasePath(rootInput)
	if err != nil {
		return fmt.Errorf("mission status: %w", err)
	}
	status, err := application.LoadMissionStatus(deps.Load, root, missionID)
	if err != nil {
		return fmt.Errorf("mission status: %w", err)
	}
	if err := deps.WriteResult(cmd, asJSON, status); err != nil {
		return err
	}
	return printBindingChanges(cmd, deps, root, missionID)
}

// printBindingChanges writes one advisory line per slot whose binding changed
// since the mission started to stderr, so stdout (the status JSON) is unchanged.
func printBindingChanges(cmd *cobra.Command, deps LifecycleDependencies, root, missionID string) error {
	if deps.BindingChanges == nil {
		return nil
	}
	for _, line := range deps.BindingChanges(root, missionID) {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "[Strategist] mission status: %s\n", line); err != nil {
			return fmt.Errorf("mission status: write advisory: %w", err)
		}
	}
	return nil
}
