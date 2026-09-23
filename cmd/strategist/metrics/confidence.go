package metrics

import (
	"fmt"
	"io"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// NewConfidence creates a new Cobra command for reporting confidence metrics.
func NewConfidence(deps Dependencies) *cobra.Command {
	var root, mission string
	cmd := &cobra.Command{Use: "confidence", Short: "Report cross-agent confidence metrics"}
	cmd.Flags().StringVar(&root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().StringVar(&mission, "mission", "", "scope the review to one mission (used at the Approval Gate)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunConfidence(cmd, deps, root, mission) }
	return cmd
}

// RunConfidence executes the confidence reporting logic.
func RunConfidence(cmd *cobra.Command, deps Dependencies, explicitRoot, mission string) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	root, err := deps.ResolveRoot(cmd, "confidence", explicitRoot)
	if err != nil {
		return err
	}
	review, err := telemetry.LoadConfidenceGateReview(root, mission)
	if err != nil {
		return fmt.Errorf("metrics confidence: %w", err)
	}
	if err := PrintConfidenceMetrics(cmd.OutOrStdout(), review); err != nil {
		return err
	}
	return PrintGateOutcome(cmd.OutOrStdout(), root, mission)
}

// PrintGateOutcome prints the gate outcome for the given mission.
func PrintGateOutcome(w io.Writer, root, missionID string) error {
	if missionID == "" {
		return nil
	}
	outcome, err := telemetry.GateOutcomeFor(root, missionID)
	if err != nil {
		return fmt.Errorf("metrics confidence: %w", err)
	}
	if outcome == "" {
		outcome = "none"
	}
	if _, err := fmt.Fprintf(w, "gate_outcome: %s\n", outcome); err != nil {
		return fmt.Errorf("metrics confidence: write output: %w", err)
	}
	return nil
}

// PrintConfidenceMetrics formats and writes confidence metrics output.
func PrintConfidenceMetrics(w io.Writer, review telemetry.ConfidenceGateReview) error {
	m := review.Metrics
	out := fmt.Sprintf("policy_version: %s\nconfidence_distribution.low: %d\nconfidence_distribution.medium: %d\nconfidence_distribution.high: %d\nclaim_kinds.question: %d\nclaim_kinds.assertion: %d\nassertion_evidence_coverage: %.2f\nunsupported_assertion_rate: %.2f\nquestion_preservation_rate: %.2f\ncorrected_high_confidence_claim_rate: %.2f\nsample_size: %d\nground_truth_sample_size: %d\nmissing_records: %d\nrejected_records: %d\nduplicate_records: %d\nreview_required: %t\nconfidence_unavailable: %t\ncalibration_status: %s\n", m.PolicyVersion, m.Distribution["low"], m.Distribution["medium"], m.Distribution["high"], m.ClaimKinds["question"], m.ClaimKinds["assertion"], m.AssertionEvidenceCoverage, m.UnsupportedAssertionRate, m.QuestionPreservationRate, m.CorrectedHighConfidenceClaimRate, m.SampleSize, m.GroundTruthSampleSize, m.MissingRecords, m.RejectedRecords, m.DuplicateRecords, review.ReviewRequired, review.Unavailable, m.CalibrationStatus)
	agents := make([]string, 0, len(m.AgentMetrics))
	for agent := range m.AgentMetrics {
		agents = append(agents, agent)
	}
	sort.Strings(agents)
	for _, agent := range agents {
		a := m.AgentMetrics[agent]
		out += fmt.Sprintf("agent.%s.sample_size: %d\nagent.%s.assertion_evidence_coverage: %.2f\nagent.%s.unsupported_assertion_rate: %.2f\nagent.%s.question_preservation_rate: %.2f\nagent.%s.corrected_high_confidence_claim_rate: %.2f\nagent.%s.ground_truth_sample_size: %d\nagent.%s.calibration_status: %s\n", agent, a.SampleSize, agent, a.AssertionEvidenceCoverage, agent, a.UnsupportedAssertionRate, agent, a.QuestionPreservationRate, agent, a.CorrectedHighConfidenceClaimRate, agent, a.GroundTruthSampleSize, agent, a.CalibrationStatus)
	}
	if _, err := fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("metrics confidence: write output: %w", err)
	}
	return nil
}
