package main

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

type metricsGateOutcomeOptions struct {
	Root, Mission, Outcome, Ref string
}

var metricsGateOutcomeCmd = &cobra.Command{
	Use:   "gate-outcome",
	Short: "Record the human Approval Gate outcome as ground truth",
	Long: `Record what the human decided at the Approval Gate. This is the only automatic
ground-truth source: the gate is the immutable point of human action. Confidence
an agent fills between handoffs is a claim and is never written as a label.

  --outcome accepted | revision_requested | rejected
  --ref     the gate event that recorded the human decision (required)

The first outcome per mission wins; a repeat is reported and not written.`,
}

func runMetricsGateOutcome(cmd *cobra.Command, opts metricsGateOutcomeOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	root, err := resolveMetricsActionRoot(cmd, "gate-outcome", opts.Root)
	if err != nil {
		return err
	}
	appended, err := telemetry.RecordGateOutcome(root, opts.Mission, opts.Outcome, opts.Ref)
	if err != nil {
		return fmt.Errorf("metrics gate-outcome: %w", err)
	}
	status := "gate outcome recorded"
	if !appended {
		status = "gate outcome already recorded for this mission; nothing written"
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: mission=%s outcome=%s\n", status, opts.Mission, opts.Outcome); err != nil {
		return fmt.Errorf("metrics gate-outcome: write output: %w", err)
	}
	return nil
}

func init() {
	opts := metricsGateOutcomeOptions{}
	f := metricsGateOutcomeCmd.Flags()
	f.StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.Mission, "mission", "", "mission id (required)")
	f.StringVar(&opts.Outcome, "outcome", "", "accepted | revision_requested | rejected (required)")
	f.StringVar(&opts.Ref, "ref", "", "gate event that recorded the human decision (required)")
	requireFlags(metricsGateOutcomeCmd, "mission", "outcome", "ref")
	metricsGateOutcomeCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMetricsGateOutcome(cmd, opts)
	}
	metricsCmd.AddCommand(metricsGateOutcomeCmd)
}
