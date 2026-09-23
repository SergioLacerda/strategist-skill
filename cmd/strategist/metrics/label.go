package metrics

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// NewLabel creates a new Cobra command to record ground truth labels.
func NewLabel(deps Dependencies) *cobra.Command {
	var root, mission, subject, label, kind, ref string
	cmd := &cobra.Command{Use: "label", Short: "Record a reviewed ground-truth label for a mission"}
	f := cmd.Flags()
	f.StringVar(&root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&mission, "mission", "", "mission id the label applies to (required)")
	f.StringVar(&subject, "subject", "", "route | handoff_application (required)")
	f.StringVar(&label, "label", "", "reviewed outcome for the subject (required)")
	f.StringVar(&kind, "kind", "", "user_revision | handoff_validation | downstream_verification (required)")
	f.StringVar(&ref, "ref", "", "review, event or check that decided the outcome (required)")
	for _, name := range []string{"mission", "subject", "label", "kind", "ref"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunLabel(cmd, deps, root, mission, subject, label, kind, ref)
	}
	return cmd
}

// RunLabel executes the ground truth label recording logic.
func RunLabel(cmd *cobra.Command, deps Dependencies, explicitRoot, mission, subject, label, kind, ref string) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	if subject == telemetry.GroundTruthSubjectGateOutcome {
		return fmt.Errorf("metrics label: gate_outcome is produced only by metrics gate-outcome")
	}
	root, err := deps.ResolveRoot(cmd, "label", explicitRoot)
	if err != nil {
		return err
	}
	appended, err := telemetry.AppendGroundTruthLabel(telemetry.GroundTruthLabelHistoryPath(root), telemetry.GroundTruthLabel{MissionID: mission, Subject: subject, Label: label, Kind: kind, Ref: ref, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	if err != nil {
		return fmt.Errorf("metrics label: %w", err)
	}
	status := "label recorded"
	if !appended {
		status = "label already recorded for this mission and subject; nothing written"
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: mission=%s subject=%s label=%s\n", status, mission, subject, label); err != nil {
		return fmt.Errorf("metrics label: write output: %w", err)
	}
	return nil
}
