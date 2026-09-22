package main

import (
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

type metricsFallbackOptions struct {
	Root string
}

var metricsFallbackCmd = newMetricsFallbackCommand()

// newMetricsFallbackCommand creates an isolated adapter instance. The command
// delegates all fallback policy and metric computation to internal/telemetry;
// it owns only Cobra binding, root selection and presentation.
func newMetricsFallbackCommand() *cobra.Command {
	opts := metricsFallbackOptions{}
	cmd := &cobra.Command{
		Use:   "fallback",
		Short: "Report provider-fallback (ADR-0028) metrics",
		Long: `Report metrics computed from .strategist/memory/fallback-decisions.jsonl:
auto_native_rate, ask_confirmed_rate — how often each provider_resolution_policy
outcome accounted for a recorded native-role fallback.

Runs cleanly against an empty .strategist/memory/ (no fallback-decisions.jsonl
yet), printing sample_size: 0 rather than erroring.`,
	}
	cmd.Flags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMetricsFallback(cmd, opts)
	}
	return cmd
}

func runMetricsFallback(cmd *cobra.Command, opts metricsFallbackOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	root, err := resolveMetricsActionRoot(cmd, "fallback", opts.Root)
	if err != nil {
		return err
	}
	decisions, err := telemetry.ReadFallbackDecisions(telemetry.FallbackDecisionHistoryPath(root))
	if err != nil {
		return fmt.Errorf("metrics fallback: %w", err)
	}
	return printFallbackMetrics(cmd.OutOrStdout(), telemetry.ComputeFallbackMetrics(decisions))
}

func printFallbackMetrics(w io.Writer, m telemetry.FallbackMetrics) error {
	out := fmt.Sprintf(
		"auto_native_rate: %.2f\n"+
			"ask_confirmed_rate: %.2f\n"+
			"sample_size: %d\n",
		m.AutoNativeRate, m.AskConfirmedRate, m.SampleSize,
	)
	if _, err := fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("metrics fallback: write output: %w", err)
	}
	return nil
}
