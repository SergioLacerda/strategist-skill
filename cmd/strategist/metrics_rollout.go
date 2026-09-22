package main

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

type metricsRolloutOptions struct {
	Enforcement, ReviewFile string
}

var metricsRolloutCmd = newMetricsRolloutCommand()

func newMetricsRolloutCommand() *cobra.Command {
	opts := metricsRolloutOptions{}
	cmd := &cobra.Command{
		Use:   "rollout-check",
		Short: "Check whether a confidence enforcement mode is admissible",
		Long: `Check a requested confidence enforcement mode against the recorded observe-mode review.

  --enforcement advisory   always admissible; rollback preserves records and the human gate
  --enforcement blocking   admissible only with a valid --review-file (see
                           docs/runbooks/confidence-observe-review.md)

Exits non-zero and reports the reason when blocking is refused; enforcement then stays advisory.`,
	}
	f := cmd.Flags()
	f.StringVar(&opts.Enforcement, "enforcement", "", "advisory | blocking (required)")
	f.StringVar(&opts.ReviewFile, "review-file", "", "recorded observe-mode review (YAML)")
	requireFlags(cmd, "enforcement")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMetricsRollout(cmd, opts)
	}
	return cmd
}

func runMetricsRollout(cmd *cobra.Command, opts metricsRolloutOptions) error {
	var review *telemetry.ObserveReview
	if opts.ReviewFile != "" {
		loaded, err := telemetry.ReadObserveReview(opts.ReviewFile)
		if err != nil {
			return fmt.Errorf("metrics rollout-check: %w", err)
		}
		review = &loaded
	}
	if err := telemetry.ValidateEnforcementChange(opts.Enforcement, review); err != nil {
		return fmt.Errorf("metrics rollout-check: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "enforcement %s admissible\n", opts.Enforcement); err != nil {
		return fmt.Errorf("metrics rollout-check: write output: %w", err)
	}
	return nil
}
