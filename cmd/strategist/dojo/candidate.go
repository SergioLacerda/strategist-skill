package dojo

import (
	"fmt"
	"time"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/spf13/cobra"
)

func newCandidate(deps Dependencies) *cobra.Command {
	var root, mission string
	cmd := &cobra.Command{
		Use:   "candidate",
		Short: "Record a regression candidate from a mission where the provider and the gate disagreed",
		Long: `Read a mission's provider readings and its gate outcome and, where they disagreed, record a regression
candidate (content-free: mission, transition, confidence, outcome). A candidate is only flagged for review; it never
counts as golden until it is promoted. The DOJO agent never runs this command.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return runCandidate(cmd, deps, root, mission) },
	}
	cmd.Flags().StringVar(&root, "root", "", rootFlagUsage)
	cmd.Flags().StringVar(&mission, "mission", "", "mission id (required)")
	_ = cmd.MarkFlagRequired("mission") //nolint:errcheck // flag is defined above
	return cmd
}

func runCandidate(cmd *cobra.Command, deps Dependencies, root, mission string) error {
	strategistRoot, basePath, err := resolveRoots(deps, root)
	if err != nil {
		return err
	}
	if err := deps.Calibration.ready(); err != nil {
		return err
	}
	signals, err := deps.Calibration.Signals(strategistRoot, mission)
	if err != nil {
		return fmt.Errorf("dojo candidate: %w", err)
	}
	found := internaldojo.DetectCandidates(mission, signals, time.Now())
	if len(found) == 0 {
		return say(cmd.OutOrStdout(), "no disagreement between the provider and the gate for %s (%d readings)\n", mission, len(signals))
	}
	return recordCandidates(cmd, basePath, mission, found)
}

func recordCandidates(cmd *cobra.Command, basePath, mission string, found []internaldojo.Candidate) error {
	for _, candidate := range found {
		if err := internaldojo.AppendCandidate(basePath, candidate); err != nil {
			return fmt.Errorf("dojo candidate: %w", err)
		}
		if err := say(cmd.OutOrStdout(), "candidate %s: %s at %s (confidence %.2f, threshold %.2f, gate %s)\n", mission, candidate.Kind, candidate.Transition, candidate.Confidence, candidate.Threshold, candidate.Outcome); err != nil {
			return err
		}
	}
	return nil
}
