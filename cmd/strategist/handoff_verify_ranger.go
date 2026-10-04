package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/spf13/cobra"
)

type rangerHandoffEvaluateOptions struct {
	Root, MissionID, Challenges, Ack string
}

var rangerHandoffEvaluateCmd = &cobra.Command{
	Use:          "evaluate-ranger",
	Short:        "Evaluate and persist the Ranger-to-Archivist lifecycle outcome",
	SilenceUsage: true,
}

func runRangerHandoffEvaluate(cmd *cobra.Command, opts rangerHandoffEvaluateOptions) error {
	if err := validateRangerHandoffOptions(opts); err != nil {
		return err
	}
	root, basePath, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("handoff evaluate-ranger: %w", err)
	}
	input, err := rangerHandoffInput(opts)
	if err != nil {
		return fmt.Errorf("handoff evaluate-ranger: %w", err)
	}
	report := rangerPrecheck(commandContext(cmd), root, basePath, opts.MissionID)
	input.Delegation = report.Delegation
	evaluation, err := evaluateRangerLocked(root, basePath, opts.MissionID, input)
	if err != nil {
		return fmt.Errorf("handoff evaluate-ranger: %w", err)
	}
	printDelegation(cmd, report)
	if err := printRangerHandoffEvaluation(cmd, evaluation); err != nil {
		return err
	}
	if evaluation.Outcome.Result == handoff.OutcomeFailed {
		return fmt.Errorf("handoff evaluate-ranger: failed (status=%s, critical_failures=%d); Archivist entry remains blocked", evaluation.Result.Status, evaluation.Result.CriticalFailures)
	}
	return nil
}

func validateRangerHandoffOptions(opts rangerHandoffEvaluateOptions) error {
	switch {
	case opts.MissionID == "":
		return fmt.Errorf("handoff evaluate-ranger: required flag --mission-id is not set")
	case (opts.Challenges == "") != (opts.Ack == ""):
		return fmt.Errorf("handoff evaluate-ranger: --challenges and --ack must be supplied together")
	}
	return nil
}

// evaluateRangerLocked records the Ranger-to-Archivist outcome under the
// mission lock the other mission commands share.
func evaluateRangerLocked(root, basePath, missionID string, input livemission.RangerHandoffInput) (livemission.RangerHandoffResult, error) {
	var evaluation livemission.RangerHandoffResult
	err := lockMission(root, missionID, func() error {
		artifact := filepath.Join(basePath, "pending", missionID+"-analysis.md")
		var evalErr error
		evaluation, evalErr = livemission.EvaluateRangerToArchivist(root, artifact, missionID, input)
		if evalErr != nil {
			return fmt.Errorf("evaluate Ranger-to-Archivist handoff: %w", evalErr)
		}
		return nil
	})
	return evaluation, err
}

func rangerHandoffInput(opts rangerHandoffEvaluateOptions) (livemission.RangerHandoffInput, error) {
	if opts.Challenges == "" {
		return livemission.RangerHandoffInput{}, nil
	}
	challenges, err := loadHandoffChallenges(opts.Challenges)
	if err != nil {
		return livemission.RangerHandoffInput{}, err
	}
	ack, err := loadHandoffAck(opts.Ack)
	if err != nil {
		return livemission.RangerHandoffInput{}, err
	}
	return livemission.RangerHandoffInput{Challenges: challenges, Ack: ack}, nil
}

func printRangerHandoffEvaluation(cmd *cobra.Command, evaluation livemission.RangerHandoffResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "transition: %s\n", handoff.TransitionRangerToArchivist)
	fmt.Fprintf(&b, "outcome: %s\n", evaluation.Outcome.Result)
	fmt.Fprintf(&b, "required: %t\n", evaluation.Outcome.Required)
	fmt.Fprintf(&b, "attempt: %d\n", evaluation.Outcome.Attempt)
	fmt.Fprintf(&b, "artifact_digest: %s\n", evaluation.Outcome.ArtifactDigest)
	if evaluation.Outcome.Required {
		if err := printHandoffVerifyResult(cmd, evaluation.Result); err != nil {
			return fmt.Errorf("print Ranger handoff outcome: %w", err)
		}
	}
	_, err := fmt.Fprint(cmd.OutOrStdout(), b.String())
	if err != nil {
		return fmt.Errorf("print Ranger handoff evaluation: %w", err)
	}
	return nil
}

func init() {
	opts := rangerHandoffEvaluateOptions{}
	rangerHandoffEvaluateCmd.Flags().StringVar(&opts.Root, cliutil.FlagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	rangerHandoffEvaluateCmd.Flags().StringVar(&opts.MissionID, "mission-id", "", "mission whose Ranger-to-Archivist handoff is evaluated (required)")
	rangerHandoffEvaluateCmd.Flags().StringVar(&opts.Challenges, "challenges", "", "path to a challenges YAML file when the typed facts require a challenge")
	rangerHandoffEvaluateCmd.Flags().StringVar(&opts.Ack, "ack", "", "path to an acknowledgment YAML file when the typed facts require a challenge")
	rangerHandoffEvaluateCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runRangerHandoffEvaluate(cmd, opts)
	}
	handoffCmd.AddCommand(rangerHandoffEvaluateCmd)
}
