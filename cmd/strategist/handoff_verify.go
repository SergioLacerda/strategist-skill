package main

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/spf13/cobra"
)

// handoffCmd is the parent for Strategist's Handoff Challenge subcommands.
var handoffCmd = &cobra.Command{
	Use:   "handoff",
	Short: "Run and record Handoff Challenge verification",
}

type handoffVerifyOptions struct {
	Root       string
	Transition string
	Policy     string
	Challenges string
	Ack        string
	MissionID  string
	Attempt    int
}

var handoffVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify a Handoff Challenge acknowledgment and record the result",
	Long: `Runs internal/handoff.Verify deterministically against a challenges file and
an acknowledgment file, prints the result, and appends a ChallengeRecord to
.strategist/memory/handoff-challenges.jsonl.

This gives the LLM agent embodying a Strategist role (Ranger, Archivist,
Sniper) a scriptable, deterministic tool to invoke instead of reasoning
through a Handoff Challenge unaided — see docs/strategist-concepts.md §
Handoff Challenge "Known Limitations" for why this command exists.

Exits non-zero when verification fails, so callers can gate on it directly.`,
}

func runHandoffVerify(cmd *cobra.Command, opts handoffVerifyOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	if err := validateHandoffVerifyOptions(opts); err != nil {
		return fmt.Errorf("handoff verify: %w", err)
	}
	policy, err := resolveHandoffPolicy(opts)
	if err != nil {
		return fmt.Errorf("handoff verify: %w", err)
	}
	challenges, err := loadHandoffChallenges(opts.Challenges)
	if err != nil {
		return fmt.Errorf("handoff verify: %w", err)
	}
	ack, err := loadHandoffAck(opts.Ack)
	if err != nil {
		return fmt.Errorf("handoff verify: %w", err)
	}

	result := handoff.Verify(policy, challenges, ack)
	if err := printHandoffVerifyResult(cmd, result); err != nil {
		return fmt.Errorf("handoff verify: %w", err)
	}

	if err := recordHandoffVerify(cmd, opts, result); err != nil {
		return fmt.Errorf("handoff verify: %w", err)
	}

	if !result.Passed {
		return fmt.Errorf("handoff verify: failed (status=%s, critical_failures=%d)", result.Status, result.CriticalFailures)
	}
	return nil
}

// validateHandoffVerifyOptions checks the flags this command treats as
// required. Not delegated to cobra's MarkFlagRequired so the error path
// stays consistent with this file's own %w-wrapped error style.
func validateHandoffVerifyOptions(opts handoffVerifyOptions) error {
	var missing []string
	if opts.Challenges == "" {
		missing = append(missing, "--challenges")
	}
	if opts.Ack == "" {
		missing = append(missing, "--ack")
	}
	if opts.MissionID == "" {
		missing = append(missing, "--mission-id")
	}
	if len(missing) > 0 {
		return fmt.Errorf("required flag(s) not set: %s", strings.Join(missing, ", "))
	}
	return nil
}

// resolveHandoffPolicy loads --policy if given, otherwise the built-in
// default for --transition.
func resolveHandoffPolicy(opts handoffVerifyOptions) (handoff.Policy, error) {
	if opts.Policy != "" {
		return loadHandoffPolicy(opts.Policy)
	}
	switch opts.Transition {
	case handoff.TransitionArchivistToSniper:
		return handoff.DefaultPolicy(), nil
	case handoff.TransitionRangerToArchivist:
		p := handoff.RangerToArchivistPolicy()
		p.Enabled = true // CLI invocation is an explicit request to verify — advisory default doesn't apply here
		return p, nil
	case handoff.TransitionSniperToValidation:
		p := handoff.SniperToValidationPolicy()
		p.Enabled = true
		return p, nil
	default:
		return handoff.Policy{}, fmt.Errorf("unknown --transition %q (want %s, %s, or %s; or pass --policy)",
			opts.Transition, handoff.TransitionArchivistToSniper, handoff.TransitionRangerToArchivist, handoff.TransitionSniperToValidation)
	}
}

func init() {
	opts := handoffVerifyOptions{}
	handoffVerifyCmd.Flags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	handoffVerifyCmd.Flags().StringVar(&opts.Transition, "transition", "", "handoff transition (archivist_to_sniper, ranger_to_archivist, sniper_to_validation) — ignored if --policy is set")
	handoffVerifyCmd.Flags().StringVar(&opts.Policy, "policy", "", "path to a policy YAML file, overriding the built-in default for --transition")
	handoffVerifyCmd.Flags().StringVar(&opts.Challenges, "challenges", "", "path to a challenges YAML file (required)")
	handoffVerifyCmd.Flags().StringVar(&opts.Ack, "ack", "", "path to an acknowledgment YAML file (required)")
	handoffVerifyCmd.Flags().StringVar(&opts.MissionID, "mission-id", "", "mission_id to record against (required)")
	handoffVerifyCmd.Flags().IntVar(&opts.Attempt, "attempt", 1, "attempt number, for repair-loop tracking")
	handoffVerifyCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runHandoffVerify(cmd, opts)
	}
	handoffCmd.AddCommand(handoffVerifyCmd)
	rootCmd.AddCommand(handoffCmd)
}
