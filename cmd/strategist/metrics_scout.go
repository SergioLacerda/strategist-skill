package main

import (
	"fmt"
	"io"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

type metricsScoutOptions struct {
	Root string
}

var metricsScoutCmd = &cobra.Command{
	Use:   "scout",
	Short: "Report Scout routing metrics",
	Long: `Report metrics computed from .strategist/memory/route-decisions.jsonl and
outcomes.jsonl: fallback_rate, unnecessary_pipeline_rate (Phase 1 —
telemetry.ComputeRouteMetrics). The four reversal-dependent metrics
(route_accuracy, direct_route_reversal_rate, risk_underclassification_rate,
user_override_rate) need a reversal ground-truth source that does not exist
in this workspace yet and are not printed until one does.

Runs cleanly against an empty .strategist/memory/ (no route-decisions.jsonl/
outcomes.jsonl yet), printing sample_size: 0 rather than erroring.`,
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
	return printRouteMetrics(os.Stdout, telemetry.ComputeRouteMetrics(decisions, outcomes))
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

func init() {
	opts := metricsScoutOptions{}
	metricsScoutCmd.Flags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	metricsScoutCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMetricsScout(cmd, opts)
	}
	metricsCmd.AddCommand(metricsScoutCmd)
}
