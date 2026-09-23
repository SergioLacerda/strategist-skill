// Package metrics contains Cobra adapters for Strategist runtime-memory metrics.
// It delegates history and aggregation behavior to internal/telemetry.
package metrics

import (
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// Dependencies defines injected dependencies for metrics commands.
type Dependencies struct {
	RootFlag    string
	ResolveRoot func(cmd *cobra.Command, action, explicitRoot string) (string, error)
	SilenceRun  func(cmd *cobra.Command)
	// ResolveBasePath returns the workspace artifact root (active.yaml's
	// base_path) for a resolved .strategist/ root. It returns "" with a nil
	// error when the workspace declares none. Optional: a nil resolver
	// disables the claim-location guard in `metrics record`.
	ResolveBasePath func(strategistRoot string) (string, error)
}

// NewFallback creates a new Cobra command for reporting provider fallback metrics.
func NewFallback(deps Dependencies) *cobra.Command {
	opts := fallbackOptions{}
	cmd := &cobra.Command{Use: "fallback", Short: "Report provider-fallback (ADR-0028) metrics"}
	cmd.Flags().StringVar(&opts.Root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if deps.SilenceRun != nil {
			deps.SilenceRun(cmd)
		}
		root, err := deps.ResolveRoot(cmd, "fallback", opts.Root)
		if err != nil {
			return err
		}
		decisions, err := telemetry.ReadFallbackDecisions(telemetry.FallbackDecisionHistoryPath(root))
		if err != nil {
			return fmt.Errorf("metrics fallback: %w", err)
		}
		return PrintFallbackMetrics(cmd.OutOrStdout(), telemetry.ComputeFallbackMetrics(decisions))
	}
	return cmd
}

type fallbackOptions struct{ Root string }

// PrintFallbackMetrics formats and writes fallback metrics to the writer.
func PrintFallbackMetrics(w io.Writer, m telemetry.FallbackMetrics) error {
	out := fmt.Sprintf("auto_native_rate: %.2f\nask_confirmed_rate: %.2f\nsample_size: %d\n", m.AutoNativeRate, m.AskConfirmedRate, m.SampleSize)
	if _, err := fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("metrics fallback: write output: %w", err)
	}
	return nil
}
