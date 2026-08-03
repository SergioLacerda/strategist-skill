package main

import (
	"fmt"
	"io"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// metricsCmd is the parent for Strategist's own runtime-memory reporting
// subcommands, one per metrics domain: "handoff" (this file) and "scout"
// (metrics_scout.go).
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Report metrics computed from Strategist's own runtime memory",
	Long:  "Report metrics computed from .strategist/memory/*.jsonl history. Each subcommand covers one metrics domain.",
}

type metricsHandoffOptions struct {
	Root string
}

var metricsHandoffCmd = &cobra.Command{
	Use:   "handoff",
	Short: "Report Handoff Challenge governance metrics",
	Long: `Report metrics computed from .strategist/memory/handoff-challenges.jsonl:
  handoff_pass_rate, first_attempt_pass_rate, critical_constraint_recall,
  decision_classification_accuracy, scope_violation_rate, handoff_repair_rate,
  semantic_handoff_loss.

Prints all rates as 0 (not an error) when no Handoff Challenge has run yet
in this workspace.`,
}

func runMetricsHandoff(cmd *cobra.Command, opts metricsHandoffOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	root, err := resolveMetricsActionRoot(cmd, "handoff", opts.Root)
	if err != nil {
		return err
	}
	records, err := telemetry.ReadHandoffChallenges(telemetry.HandoffChallengeHistoryPath(root))
	if err != nil {
		return fmt.Errorf("metrics handoff: %w", err)
	}
	return printHandoffMetrics(os.Stdout, telemetry.ComputeHandoffMetrics(records))
}

func resolveMetricsActionRoot(cmd *cobra.Command, action, explicitRoot string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("metrics %s: get cwd: %w", action, err)
	}
	root, _, err := resolveStrategistRoot(stringFlag(cmd, flagRoot, explicitRoot), cwd)
	if err != nil {
		return "", fmt.Errorf("metrics %s: %w", action, err)
	}
	return root, nil
}

func printHandoffMetrics(w io.Writer, m telemetry.HandoffMetrics) error {
	out := fmt.Sprintf(
		"handoff_pass_rate: %.2f\n"+
			"first_attempt_pass_rate: %.2f\n"+
			"critical_constraint_recall: %.2f\n"+
			"decision_classification_accuracy: %.2f\n"+
			"scope_violation_rate: %.2f\n"+
			"handoff_repair_rate: %.2f\n"+
			"semantic_handoff_loss.recall: %.2f\n"+
			"semantic_handoff_loss.classification: %.2f\n"+
			"semantic_handoff_loss.application: %.2f\n"+
			"sample_size: %d\n",
		m.HandoffPassRate, m.FirstAttemptPassRate, m.CriticalConstraintRecall,
		m.DecisionClassificationAccuracy, m.ScopeViolationRate, m.HandoffRepairRate,
		m.SemanticLoss.Recall, m.SemanticLoss.Classification, m.SemanticLoss.Application,
		m.SampleSize,
	)
	if _, err := fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("metrics handoff: write output: %w", err)
	}
	return nil
}

func init() {
	opts := metricsHandoffOptions{}
	metricsHandoffCmd.Flags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	metricsHandoffCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMetricsHandoff(cmd, opts)
	}
	metricsCmd.AddCommand(metricsHandoffCmd)
	rootCmd.AddCommand(metricsCmd)
}
