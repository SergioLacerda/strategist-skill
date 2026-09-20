package main

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

type metricsConfidenceOptions struct {
	Root string
}

var metricsConfidenceCmd = &cobra.Command{
	Use:   "confidence",
	Short: "Report cross-agent confidence metrics",
	Long:  "Report comparable claim metrics from .strategist/memory/confidence-records.jsonl.\nThe confidence percentage is policy data; calibration_status and ground_truth_sample_size\nshow whether empirical evidence exists. Empty history reports no_sample.",
}

func runMetricsConfidence(cmd *cobra.Command, opts metricsConfidenceOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	root, err := resolveMetricsActionRoot(cmd, "confidence", opts.Root)
	if err != nil {
		return err
	}
	records, diagnostics, err := telemetry.ReadConfidenceRecordsWithDiagnostics(telemetry.ConfidenceHistoryPath(root))
	if err != nil {
		return fmt.Errorf("metrics confidence: %w", err)
	}
	review := telemetry.BuildConfidenceGateReview(records, diagnostics)
	return printConfidenceMetrics(os.Stdout, review)
}

func printConfidenceMetrics(w io.Writer, review telemetry.ConfidenceGateReview) error {
	m := review.Metrics
	out := fmt.Sprintf(
		"policy_version: %s\n"+
			"confidence_distribution.low: %d\n"+
			"confidence_distribution.medium: %d\n"+
			"confidence_distribution.high: %d\n"+
			"claim_kinds.question: %d\n"+
			"claim_kinds.assertion: %d\n"+
			"assertion_evidence_coverage: %.2f\n"+
			"unsupported_assertion_rate: %.2f\n"+
			"question_preservation_rate: %.2f\n"+
			"corrected_high_confidence_claim_rate: %.2f\n"+
			"sample_size: %d\n"+
			"ground_truth_sample_size: %d\n"+
			"missing_records: %d\n"+
			"rejected_records: %d\n"+
			"duplicate_records: %d\n"+
			"review_required: %t\n"+
			"confidence_unavailable: %t\n"+
			"calibration_status: %s\n",
		m.PolicyVersion,
		m.Distribution["low"], m.Distribution["medium"], m.Distribution["high"],
		m.ClaimKinds["question"], m.ClaimKinds["assertion"],
		m.AssertionEvidenceCoverage, m.UnsupportedAssertionRate,
		m.QuestionPreservationRate, m.CorrectedHighConfidenceClaimRate,
		m.SampleSize, m.GroundTruthSampleSize, m.MissingRecords,
		m.RejectedRecords, m.DuplicateRecords, review.ReviewRequired,
		review.Unavailable, m.CalibrationStatus,
	)
	agents := make([]string, 0, len(m.AgentMetrics))
	for agent := range m.AgentMetrics {
		agents = append(agents, agent)
	}
	sort.Strings(agents)
	for _, agent := range agents {
		agentMetrics := m.AgentMetrics[agent]
		out += fmt.Sprintf(
			"agent.%s.sample_size: %d\n"+
				"agent.%s.assertion_evidence_coverage: %.2f\n"+
				"agent.%s.unsupported_assertion_rate: %.2f\n"+
				"agent.%s.question_preservation_rate: %.2f\n"+
				"agent.%s.corrected_high_confidence_claim_rate: %.2f\n"+
				"agent.%s.ground_truth_sample_size: %d\n"+
				"agent.%s.calibration_status: %s\n",
			agent, agentMetrics.SampleSize,
			agent, agentMetrics.AssertionEvidenceCoverage,
			agent, agentMetrics.UnsupportedAssertionRate,
			agent, agentMetrics.QuestionPreservationRate,
			agent, agentMetrics.CorrectedHighConfidenceClaimRate,
			agent, agentMetrics.GroundTruthSampleSize,
			agent, agentMetrics.CalibrationStatus,
		)
	}
	if _, err := fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("metrics confidence: write output: %w", err)
	}
	return nil
}

func init() {
	opts := metricsConfidenceOptions{}
	metricsConfidenceCmd.Flags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	metricsConfidenceCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMetricsConfidence(cmd, opts)
	}
	metricsCmd.AddCommand(metricsConfidenceCmd)
}
