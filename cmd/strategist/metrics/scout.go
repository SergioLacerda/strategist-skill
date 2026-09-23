package metrics

import (
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// NewScout creates a new Cobra command for reporting Scout routing metrics.
func NewScout(deps Dependencies) *cobra.Command {
	var root string
	cmd := &cobra.Command{Use: "scout", Short: "Report Scout routing metrics"}
	cmd.Flags().StringVar(&root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunScout(cmd, deps, root) }
	return cmd
}

// RunScout executes the scout metrics reporting logic.
func RunScout(cmd *cobra.Command, deps Dependencies, explicitRoot string) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	root, err := deps.ResolveRoot(cmd, "scout", explicitRoot)
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
	if err := PrintRouteMetrics(cmd.OutOrStdout(), telemetry.ComputeRouteMetrics(decisions, outcomes)); err != nil {
		return err
	}
	return PrintRouteGroundTruthMetrics(cmd.OutOrStdout(), telemetry.ComputeRouteGroundTruthMetrics(decisions, labels))
}

// PrintRouteGroundTruthMetrics formats and writes route ground truth metrics.
func PrintRouteGroundTruthMetrics(w io.Writer, m telemetry.RouteGroundTruthMetrics) error {
	out := fmt.Sprintf("ground_truth_sample_size: %d\ncalibration_status: %s\n", m.SampleSize, m.CalibrationStatus)
	if m.SampleSize > 0 {
		out += fmt.Sprintf("route_accuracy: %.2f\ndirect_route_reversal_rate: %.2f\nrisk_underclassification_rate: %.2f\nuser_override_rate: %.2f\n", m.RouteAccuracy, m.DirectRouteReversalRate, m.RiskUnderclassificationRate, m.UserOverrideRate)
	}
	if _, err := fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("metrics scout: write output: %w", err)
	}
	return nil
}

// PrintRouteMetrics formats and writes route metrics.
func PrintRouteMetrics(w io.Writer, m telemetry.RouteMetrics) error {
	out := fmt.Sprintf("fallback_rate: %.2f\nunnecessary_pipeline_rate: %.2f\nsample_size: %d\nfull_pipeline_sample_size: %d\n", m.FallbackRate, m.UnnecessaryPipelineRate, m.SampleSize, m.FullPipelineSampleSize)
	if _, err := fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("metrics scout: write output: %w", err)
	}
	return nil
}
