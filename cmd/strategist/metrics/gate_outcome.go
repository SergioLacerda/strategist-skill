package metrics

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// NewGateOutcome creates a new Cobra command to record human Approval Gate outcome.
func NewGateOutcome(deps Dependencies) *cobra.Command {
	var root, mission, outcome, ref string
	cmd := &cobra.Command{Use: "gate-outcome", Short: "Record the human Approval Gate outcome as ground truth"}
	f := cmd.Flags()
	f.StringVar(&root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&mission, "mission", "", "mission id (required)")
	f.StringVar(&outcome, "outcome", "", "accepted | revision_requested | rejected (required)")
	f.StringVar(&ref, "ref", "", "gate event that recorded the human decision (required)")
	for _, name := range []string{"mission", "outcome", "ref"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunGateOutcome(cmd, deps, root, mission, outcome, ref)
	}
	return cmd
}

// RunGateOutcome executes the gate outcome recording logic.
func RunGateOutcome(cmd *cobra.Command, deps Dependencies, explicitRoot, mission, outcome, ref string) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	root, err := deps.ResolveRoot(cmd, "gate-outcome", explicitRoot)
	if err != nil {
		return err
	}
	appended, err := telemetry.RecordGateOutcome(root, mission, outcome, ref)
	if err != nil {
		return fmt.Errorf("metrics gate-outcome: %w", err)
	}
	status := "gate outcome recorded"
	if !appended {
		status = "gate outcome already recorded for this mission; nothing written"
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: mission=%s outcome=%s\n", status, mission, outcome); err != nil {
		return fmt.Errorf("metrics gate-outcome: write output: %w", err)
	}
	return nil
}
