package main

import (
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

type metricsScoutOptions struct {
	Root string
}

var metricsScoutCmd = newMetricsScoutCommand()

func newMetricsScoutCommand() *cobra.Command {
	opts := metricsScoutOptions{}
	cmd := &cobra.Command{
		Use:   "scout",
		Short: "Report Scout routing metrics",
		Long: `Report metrics computed from .strategist/memory/route-decisions.jsonl and
outcomes.jsonl: fallback_rate, unnecessary_pipeline_rate (Phase 1 —
telemetry.ComputeRouteMetrics). The four reversal-dependent metrics
(route_accuracy, direct_route_reversal_rate, risk_underclassification_rate,
user_override_rate) are computed from reviewed labels in
.strategist/memory/ground-truth-labels.jsonl and are printed only once at
least one decision has a label; otherwise calibration_status is no_sample.

Runs cleanly against an empty .strategist/memory/ (no route-decisions.jsonl/
outcomes.jsonl yet), printing sample_size: 0 rather than erroring.`,
	}
	cmd.Flags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMetricsScout(cmd, opts)
	}
	return cmd
}

func runMetricsScout(cmd *cobra.Command, opts metricsScoutOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	root, err := resolveMetricsActionRoot(cmd, "scout", opts.Root)
	if err != nil {
		return err
	}
	decisions, err := telemetry.ReadRouteDecisions(telemetry.RouteDecisionHistoryPath(root))
	if err != nil {
		return fmt.Errorf("metrics scout: %w", err)
	}
	outcomes, err := telemetry.ReadOutcomes(telemetry.OutcomeHistoryPath(root))
	if err != nil {
		return fmt.Errorf("metrics scout: %w", err)
	}
	labels, err := telemetry.ReadGroundTruthLabels(telemetry.GroundTruthLabelHistoryPath(root), telemetry.GroundTruthSubjectRoute)
	if err != nil {
		return fmt.Errorf("metrics scout: %w", err)
	}
	if err := printRouteMetrics(cmd.OutOrStdout(), telemetry.ComputeRouteMetrics(decisions, outcomes)); err != nil {
		return err
	}
	return printRouteGroundTruthMetrics(cmd.OutOrStdout(), telemetry.ComputeRouteGroundTruthMetrics(decisions, labels))
}

func printRouteGroundTruthMetrics(w io.Writer, m telemetry.RouteGroundTruthMetrics) error {
	out := fmt.Sprintf("ground_truth_sample_size: %d\ncalibration_status: %s\n", m.SampleSize, m.CalibrationStatus)
	if m.SampleSize > 0 {
		out += fmt.Sprintf(
			"route_accuracy: %.2f\n"+
				"direct_route_reversal_rate: %.2f\n"+
				"risk_underclassification_rate: %.2f\n"+
				"user_override_rate: %.2f\n",
			m.RouteAccuracy, m.DirectRouteReversalRate, m.RiskUnderclassificationRate, m.UserOverrideRate,
		)
	}
	if _, err := fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("metrics scout: write output: %w", err)
	}
	return nil
}

func printRouteMetrics(w io.Writer, m telemetry.RouteMetrics) error {
	out := fmt.Sprintf(
		"fallback_rate: %.2f\n"+
			"unnecessary_pipeline_rate: %.2f\n"+
			"sample_size: %d\n"+
			"full_pipeline_sample_size: %d\n",
		m.FallbackRate, m.UnnecessaryPipelineRate, m.SampleSize, m.FullPipelineSampleSize,
	)
	if _, err := fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("metrics scout: write output: %w", err)
	}
	return nil
}
