package main

import (
	"encoding/json"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/gatecriteria"
	"github.com/spf13/cobra"
)

type handoffCriteriaOptions struct {
	Root      string
	MissionID string
	JSON      bool
}

var handoffCriteriaCmd = &cobra.Command{
	Use:   "criteria",
	Short: "Report the quality criteria already computed for a mission (read-only)",
	Long: `Prints the quality criteria the Approval Gate shows, gathered only from records earlier
stages already wrote: route and claim confidence, the confidence review, the deterministic
handoff outcomes, the critic and mission-quality results, the refinement handoff metrics and
the provider (JEV) readings. Every row is always present and filled: a value, "não avaliado"
(the stage did not run or was not enabled) or "não sei" (data exists but cannot support a
statement). The command never calls the provider and never recomputes a criterion.`,
	SilenceUsage: true,
}

func runHandoffCriteria(cmd *cobra.Command, opts handoffCriteriaOptions) error {
	if opts.MissionID == "" {
		return fmt.Errorf("handoff criteria: required flag --mission-id is not set")
	}
	root, basePath, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("handoff criteria: %w", err)
	}
	report := gatecriteria.Build(gatecriteria.Input{Root: root, BasePath: basePath, MissionID: opts.MissionID})
	if opts.JSON {
		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("handoff criteria: %w", err)
		}
		return printLines(cmd, string(raw))
	}
	lines := make([]string, 0, len(report.Rows))
	for _, row := range report.Rows {
		lines = append(lines, fmt.Sprintf("%s: %s  [%s; %s; %s]", row.Criterion, row.Value, row.State, row.Source, row.Detail))
	}
	return printLines(cmd, lines...)
}

func init() {
	opts := handoffCriteriaOptions{}
	handoffCriteriaCmd.Flags().StringVar(&opts.Root, cliutil.FlagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	handoffCriteriaCmd.Flags().StringVar(&opts.MissionID, "mission-id", "", "mission whose criteria are reported (required)")
	handoffCriteriaCmd.Flags().BoolVar(&opts.JSON, "json", false, "emit machine-readable JSON")
	handoffCriteriaCmd.RunE = func(cmd *cobra.Command, _ []string) error { return runHandoffCriteria(cmd, opts) }
	handoffCmd.AddCommand(handoffCriteriaCmd)
}
