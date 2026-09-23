package metrics

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// NewRollout creates a new Cobra command to check confidence enforcement admissibility.
func NewRollout() *cobra.Command {
	var enforcement, reviewFile string
	cmd := &cobra.Command{Use: "rollout-check", Short: "Check whether a confidence enforcement mode is admissible"}
	cmd.Flags().StringVar(&enforcement, "enforcement", "", "advisory | blocking (required)")
	cmd.Flags().StringVar(&reviewFile, "review-file", "", "recorded observe-mode review (YAML)")
	if err := cmd.MarkFlagRequired("enforcement"); err != nil {
		panic(err)
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunRollout(cmd, enforcement, reviewFile) }
	return cmd
}

// RunRollout executes the rollout check logic.
func RunRollout(cmd *cobra.Command, enforcement, reviewFile string) error {
	var review *telemetry.ObserveReview
	if reviewFile != "" {
		loaded, err := telemetry.ReadObserveReview(reviewFile)
		if err != nil {
			return fmt.Errorf("metrics rollout-check: %w", err)
		}
		review = &loaded
	}
	if err := telemetry.ValidateEnforcementChange(enforcement, review); err != nil {
		return fmt.Errorf("metrics rollout-check: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "enforcement %s admissible\n", enforcement); err != nil {
		return fmt.Errorf("metrics rollout-check: write output: %w", err)
	}
	return nil
}
