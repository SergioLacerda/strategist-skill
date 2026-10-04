package metrics

import (
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// NewHandoff creates a new Cobra command for handoff governance metrics.
func NewHandoff(deps Dependencies) *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "handoff",
		Short: "Report Handoff Challenge governance metrics",
		Long:  "Report Handoff Challenge governance metrics from .strategist/memory/handoff-challenges.jsonl.",
	}
	cmd.Flags().StringVar(&root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunHandoff(cmd, deps, root)
	}
	return cmd
}

// RunHandoff executes the handoff metrics reporting logic.
func RunHandoff(cmd *cobra.Command, deps Dependencies, explicitRoot string) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	root, err := deps.ResolveRoot(cmd, "handoff", explicitRoot)
	if err != nil {
		return err
	}
	report, err := application.ReportHandoffMetrics(root, handoffMetricsReportPorts())
	if err != nil {
		return fmt.Errorf("metrics handoff: %w", err)
	}
	return PrintHandoffMetrics(cmd.OutOrStdout(), telemetryHandoffMetrics(report))
}

func telemetryHandoffMetrics(report application.HandoffMetricsReport) telemetry.HandoffMetrics {
	return telemetry.HandoffMetrics{
		HandoffPassRate: report.HandoffPassRate, FirstAttemptPassRate: report.FirstAttemptPassRate,
		CriticalConstraintRecall: report.CriticalConstraintRecall, DecisionClassificationAccuracy: report.DecisionClassificationAccuracy,
		ScopeViolationRate: report.ScopeViolationRate, HandoffRepairRate: report.HandoffRepairRate,
		SemanticLoss: telemetry.SemanticHandoffLoss{
			Recall: report.SemanticLoss.Recall, Classification: report.SemanticLoss.Classification, Application: report.SemanticLoss.Application,
		},
		SampleSize: report.SampleSize, ApplicationSampleSize: report.ApplicationSampleSize,
	}
}

// PrintHandoffMetrics formats and writes handoff metrics to the writer.
func PrintHandoffMetrics(w io.Writer, m telemetry.HandoffMetrics) error {
	out := fmt.Sprintf(
		"handoff_pass_rate: %.2f\nfirst_attempt_pass_rate: %.2f\ncritical_constraint_recall: %.2f\ndecision_classification_accuracy: %.2f\nscope_violation_rate: %.2f\nhandoff_repair_rate: %.2f\nsemantic_handoff_loss.recall: %.2f\nsemantic_handoff_loss.classification: %.2f\nsemantic_handoff_loss.application: %.2f\nsample_size: %d\napplication_sample_size: %d\n",
		m.HandoffPassRate, m.FirstAttemptPassRate, m.CriticalConstraintRecall, m.DecisionClassificationAccuracy, m.ScopeViolationRate, m.HandoffRepairRate, m.SemanticLoss.Recall, m.SemanticLoss.Classification, m.SemanticLoss.Application, m.SampleSize, m.ApplicationSampleSize,
	)
	if _, err := fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("metrics handoff: write output: %w", err)
	}
	return nil
}
