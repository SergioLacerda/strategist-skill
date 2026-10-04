package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type handoffEvaluateOptions struct {
	Root       string
	MissionID  string
	RiskLevel  string
	Challenges string
	Ack        string
	// ConfidenceSummary is an optional YAML file holding the declared handoff
	// confidence_summary; without it an explicit missing-record is written.
	ConfidenceSummary string
}

var handoffEvaluateCmd = &cobra.Command{
	Use:   "evaluate",
	Short: "Evaluate the Archivist-to-Sniper handoff and record its outcome",
	Long: `Evaluates the Archivist-to-Sniper handoff of a mission: validates its refined
package, derives the policy signals from the package's typed facts, resolves the
policy, runs the semantic challenge when it is required, and durably records a
terminal outcome (passed, failed, or a policy-authorized skip) correlated to the
mission, the package revision and the attempt, and records the handoff confidence
(or an explicit missing-record). A failed outcome returns the mission to refinement,
so a repaired package needs a new Approval Gate acceptance; the last allowed failure
blocks it. A passed or skipped outcome does not enter execution by itself.
The policy is never chosen by the caller. --risk-level is the optional coarse
intake label and can only make the result stricter. The Approval Gate must
already be accepted. Entering execution (mission submit --event
handoff_challenge_satisfied) verifies and consumes this outcome.`,
	SilenceUsage: true,
}

// runArchivistHandoff is `handoff evaluate`: the production boundary for the
// Archivist-to-Sniper handoff. Nothing the caller passes can declare the
// challenge unnecessary.
func runArchivistHandoff(cmd *cobra.Command, opts handoffEvaluateOptions) error {
	silenceHandoffTelemetry(cmd)
	if err := validateArchivistHandoffOptions(opts); err != nil {
		return fmt.Errorf("handoff evaluate: %w", err)
	}
	root, basePath, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("handoff evaluate: %w", err)
	}
	input, err := archivistHandoffInput(opts)
	if err != nil {
		return fmt.Errorf("handoff evaluate: %w", err)
	}
	evaluation, err := recordArchivistHandoffLocked(root, basePath, opts.MissionID, input)
	if err != nil {
		return fmt.Errorf("handoff evaluate: %w", err)
	}
	if err := printArchivistHandoff(cmd, evaluation); err != nil {
		return err
	}
	if evaluation.Outcome.Result == handoff.OutcomeFailed {
		return fmt.Errorf("handoff evaluate: failed (status=%s, critical_failures=%d); the mission returned to refinement and needs a new Approval Gate acceptance", evaluation.Result.Status, evaluation.Result.CriticalFailures)
	}
	return nil
}

// recordArchivistHandoffLocked keeps the command's output-oriented adapter
// while delegating mission lock/load/evaluate/save ordering to the
// application service.
func recordArchivistHandoffLocked(root, basePath, missionID string, input livemission.ArchivistHandoffInput) (livemission.ArchivistHandoffResult, error) {
	var evaluation livemission.ArchivistHandoffResult
	err := application.RecordHandoffLifecycle(root, basePath, missionID, application.HandoffLifecyclePorts{
		Lock: lockMission,
		Load: loadMission,
		Save: saveMission,
		Evaluate: func(root, basePath string, engine *domain.MissionEngine) (domain.MissionEngineStatus, bool, error) {
			var status domain.MissionEngineStatus
			var changed bool
			var evalErr error
			evaluation, status, changed, evalErr = livemission.RecordArchivistHandoff(root, basePath, engine, input)
			if evalErr != nil {
				return status, changed, fmt.Errorf("record Archivist handoff: %w", evalErr)
			}
			return status, changed, nil
		},
	})
	if err != nil {
		return evaluation, fmt.Errorf("record handoff lifecycle: %w", err)
	}
	return evaluation, nil
}

func validateArchivistHandoffOptions(opts handoffEvaluateOptions) error {
	switch {
	case opts.MissionID == "":
		return fmt.Errorf("required flag(s) not set: --mission-id")
	case (opts.Challenges == "") != (opts.Ack == ""):
		return fmt.Errorf("--challenges and --ack must be supplied together")
	}
	return nil
}

func archivistHandoffInput(opts handoffEvaluateOptions) (livemission.ArchivistHandoffInput, error) {
	input := livemission.ArchivistHandoffInput{RiskLevel: opts.RiskLevel}
	if opts.ConfidenceSummary != "" {
		summary, err := loadHandoffConfidenceSummary(opts.ConfidenceSummary)
		if err != nil {
			return livemission.ArchivistHandoffInput{}, err
		}
		input.ConfidenceSummary = &summary
	}
	if opts.Challenges == "" {
		return input, nil
	}
	challenges, err := loadHandoffChallenges(opts.Challenges)
	if err != nil {
		return livemission.ArchivistHandoffInput{}, err
	}
	ack, err := loadHandoffAck(opts.Ack)
	if err != nil {
		return livemission.ArchivistHandoffInput{}, err
	}
	input.Challenges, input.Ack = challenges, ack
	return input, nil
}

func loadHandoffConfidenceSummary(path string) (domain.ConfidenceSummary, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: operator-supplied confidence summary path, same trust level as --challenges/--ack
	if err != nil {
		return domain.ConfidenceSummary{}, fmt.Errorf("read confidence summary file: %w", err)
	}
	var summary domain.ConfidenceSummary
	if err := yaml.Unmarshal(raw, &summary); err != nil {
		return domain.ConfidenceSummary{}, fmt.Errorf("parse confidence summary file: %w", err)
	}
	return summary, nil
}

func printArchivistHandoff(cmd *cobra.Command, evaluation livemission.ArchivistHandoffResult) error {
	outcome := evaluation.Outcome
	var b strings.Builder
	fmt.Fprintf(&b, "outcome: %s\n", outcome.Result)
	fmt.Fprintf(&b, "required: %t\n", outcome.Required)
	fmt.Fprintf(&b, "attempt: %d\n", outcome.Attempt)
	fmt.Fprintf(&b, "package_digest: %s\n", outcome.PackageDigest)
	for _, provenance := range outcome.Provenance {
		fmt.Fprintf(&b, "signal: %s (%s)\n", provenance.Signal, provenance.Source)
	}
	if outcome.Required {
		if err := printHandoffVerifyResult(cmd, evaluation.Result); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), b.String()); err != nil {
		return fmt.Errorf("print handoff outcome: %w", err)
	}
	return nil
}

func init() {
	opts := handoffEvaluateOptions{}
	handoffEvaluateCmd.Flags().StringVar(&opts.Root, cliutil.FlagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	handoffEvaluateCmd.Flags().StringVar(&opts.MissionID, "mission-id", "", "mission whose Archivist-to-Sniper handoff is evaluated (required)")
	handoffEvaluateCmd.Flags().StringVar(&opts.RiskLevel, "risk-level", "", "optional intake risk_level (low, medium, high); can only make the result stricter")
	handoffEvaluateCmd.Flags().StringVar(&opts.Challenges, "challenges", "", "path to a challenges YAML file (needed when the package requires the challenge)")
	handoffEvaluateCmd.Flags().StringVar(&opts.Ack, "ack", "", "path to an acknowledgment YAML file (needed when the package requires the challenge)")
	handoffEvaluateCmd.Flags().StringVar(&opts.ConfidenceSummary, "confidence-summary", "", "optional YAML file with the declared handoff confidence_summary; without it a missing-record is written")
	handoffEvaluateCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runArchivistHandoff(cmd, opts)
	}
	handoffCmd.AddCommand(handoffEvaluateCmd)
}
