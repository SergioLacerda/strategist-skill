package mission

import (
	"fmt"

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
	_, status, err := deps.Load(root, missionID)
	if err != nil {
		return fmt.Errorf("mission status: %w", err)
	}
	return deps.WriteResult(cmd, asJSON, status)
}
