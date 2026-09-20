package main

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

type metricsLabelOptions struct {
	Root, Mission, Subject, Label, Kind, Ref string
}

var metricsLabelCmd = &cobra.Command{
	Use:   "label",
	Short: "Record a reviewed ground-truth label for a mission",
	Long: `Append one reviewed outcome to .strategist/memory/ground-truth-labels.jsonl.

  --subject route                 labels: confirmed, reversed, risk_underclassified, user_override
  --subject handoff_application   labels: applied, not_applied
  --subject gate_outcome          labels: accepted, revision_requested, rejected
  --kind                          user_revision, handoff_validation, downstream_verification

--ref is mandatory: it must point to the review, event or check that decided
the outcome. A label without a source is rejected. The first label per
mission and subject wins; a repeat is reported and not written.`,
}

func runMetricsLabel(cmd *cobra.Command, opts metricsLabelOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	if opts.Subject == telemetry.GroundTruthSubjectGateOutcome {
		return fmt.Errorf("metrics label: gate_outcome is produced only by metrics gate-outcome")
	}
	root, err := resolveMetricsActionRoot(cmd, "label", opts.Root)
	if err != nil {
		return err
	}
	label := telemetry.GroundTruthLabel{
		MissionID: opts.Mission, Subject: opts.Subject, Label: opts.Label,
		Kind: opts.Kind, Ref: opts.Ref, Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	appended, err := telemetry.AppendGroundTruthLabel(telemetry.GroundTruthLabelHistoryPath(root), label)
	if err != nil {
		return fmt.Errorf("metrics label: %w", err)
	}
	status := "label recorded"
	if !appended {
		status = "label already recorded for this mission and subject; nothing written"
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: mission=%s subject=%s label=%s\n", status, opts.Mission, opts.Subject, opts.Label); err != nil {
		return fmt.Errorf("metrics label: write output: %w", err)
	}
	return nil
}

func init() {
	opts := metricsLabelOptions{}
	f := metricsLabelCmd.Flags()
	f.StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.Mission, "mission", "", "mission id the label applies to (required)")
	f.StringVar(&opts.Subject, "subject", "", "route | handoff_application (required)")
	f.StringVar(&opts.Label, "label", "", "reviewed outcome for the subject (required)")
	f.StringVar(&opts.Kind, "kind", "", "user_revision | handoff_validation | downstream_verification (required)")
	f.StringVar(&opts.Ref, "ref", "", "review, event or check that decided the outcome (required)")
	requireFlags(metricsLabelCmd, "mission", "subject", "label", "kind", "ref")
	metricsLabelCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMetricsLabel(cmd, opts)
	}
	metricsCmd.AddCommand(metricsLabelCmd)
}

// requireFlags marks flags required; a missing flag name is a programming error.
func requireFlags(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
}
