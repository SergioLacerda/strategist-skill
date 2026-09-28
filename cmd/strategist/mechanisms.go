package main

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/mechanisms"
	"github.com/spf13/cobra"
)

var mechanismsCmd = newMechanismsCmd()

// newMechanismsCmd builds the `mechanisms` family. It is a constructor so the
// tests can run it in isolation.
func newMechanismsCmd() *cobra.Command {
	parent := &cobra.Command{
		Use:   "mechanisms",
		Short: "Inspect the Mechanisms registry",
	}
	parent.AddCommand(newMechanismsBriefCmd())
	return parent
}

func newMechanismsBriefCmd() *cobra.Command {
	var role, root string
	var full bool
	cmd := &cobra.Command{
		Use:   "brief --role <role>",
		Short: "Print the tools (Mechanisms and Abilities) available to one role",
		Long: `Prints the role-scoped view of the Mechanisms registry
(contracts/machine/mechanisms.yaml): what each tool is and how to invoke it.
Roles run it from their on_start hook so an agent in a mission knows what is at
its disposal. --full adds the enforcement tier and when to use each tool.`,
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringVar(&role, "role", "", "role to print the brief for (required)")
	cmd.Flags().StringVar(&root, cliutil.FlagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().BoolVar(&full, "full", false, "include enforcement and when-to-use detail")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMechanismsBrief(cmd, root, role, full)
	}
	return cmd
}

func runMechanismsBrief(cmd *cobra.Command, root, role string, full bool) error {
	if run := cliutil.TelemetryRunFromCmd(cmd); run != nil {
		run.SetSilent() // the brief is read by an agent at role start: no telemetry banner around it
	}
	if !domain.DefaultRoleRegistry().Has(role) {
		return fmt.Errorf("mechanisms brief: --role %q is not a Strategist role", role)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("mechanisms brief: get cwd: %w", err)
	}
	strategistRoot, _, err := cliutil.ResolveStrategistRoot(root, cwd)
	if err != nil {
		return fmt.Errorf("mechanisms brief: %w", err)
	}
	registry, err := mechanisms.Load(strategistRoot)
	if err != nil {
		return fmt.Errorf("mechanisms brief: %w", err)
	}
	brief := registry.Brief(role)
	if full {
		brief = registry.BriefFull(role)
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), brief); err != nil {
		return fmt.Errorf("mechanisms brief: write: %w", err)
	}
	return nil
}
