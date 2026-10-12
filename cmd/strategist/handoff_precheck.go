package main

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/cli/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/spf13/cobra"
)

type handoffPrecheckOptions struct {
	Root      string
	MissionID string
}

var handoffPrecheckCmd = &cobra.Command{
	Use:   "precheck",
	Short: "Read the refined package with the provider before the Approval Gate (advisory)",
	Long: `Asks the enabled provider whether the refined package conforms to its contract
criteria, before the Approval Gate is presented. This is the advisory Archivist to
Gate boundary: it records no lifecycle outcome and approves nothing. The reading is
written to the integration call ledger and to the confidence ledger, and the later
handoff evaluate reuses it when the package digest is unchanged, so the provider is
not asked twice for one revision. Without an enabled integration it prints
"delegation: disabled" and changes nothing.`,
	SilenceUsage: true,
}

// runHandoffPrecheck is `handoff precheck`. It never fails a handoff: a provider
// that is unreachable or not confident is reported as a fallback or a signal.
func runHandoffPrecheck(cmd *cobra.Command, opts handoffPrecheckOptions) error {
	if opts.MissionID == "" {
		return fmt.Errorf("handoff precheck: required flag --mission-id is not set")
	}
	root, basePath, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("handoff precheck: %w", err)
	}
	report := gatePrecheck(commandContext(cmd), root, basePath, opts.MissionID)
	if err := printLines(cmd, "transition: "+handoffconsumer.TransitionArchivistToGate); err != nil {
		return err
	}
	if report.Status == handoffconsumer.StatusDisabled {
		return printLines(cmd, "delegation: disabled (no provider reading; main path only)")
	}
	printDelegation(cmd, report)
	return nil
}

func init() {
	opts := handoffPrecheckOptions{}
	handoffPrecheckCmd.Flags().StringVar(&opts.Root, cliutil.FlagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	handoffPrecheckCmd.Flags().StringVar(&opts.MissionID, "mission-id", "", "mission whose refined package is read (required)")
	handoffPrecheckCmd.RunE = func(cmd *cobra.Command, _ []string) error { return runHandoffPrecheck(cmd, opts) }
	handoffCmd.AddCommand(handoffPrecheckCmd)
}
