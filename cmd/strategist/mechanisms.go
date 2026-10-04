package main

import (
	"fmt"
	"os"
	"strings"

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
	var role, root, stage string
	var full bool
	cmd := &cobra.Command{
		Use:   "brief --role <role>",
		Short: "Print the Tools, Mechanisms, and Feats available to one role",
		Long: `Prints the role-scoped view of the Mechanisms registry
(contracts/machine/mechanisms.yaml): what each tool is and how to invoke it.
Roles run it from their on_start hook so an agent in a mission knows what is at
its disposal. --stage narrows the view to a canonical Stage and its phases.
--full adds the enforcement tier and when to use each tool.`,
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringVar(&role, "role", "", "role to print the brief for (required)")
	cmd.Flags().StringVar(&stage, "stage", "", "canonical Stage to scope awareness to (FULL, SHORT, or ROSTER)")
	cmd.Flags().StringVar(&root, cliutil.FlagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().BoolVar(&full, "full", false, "include enforcement and when-to-use detail")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMechanismsBriefAtStage(cmd, root, role, stage, full)
	}
	return cmd
}

func runMechanismsBrief(cmd *cobra.Command, root, role string, full bool) error {
	return runMechanismsBriefAtStage(cmd, root, role, "", full)
}

func runMechanismsBriefAtStage(cmd *cobra.Command, root, role, stage string, full bool) error {
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
	brief, err := renderMechanismsBrief(registry, role, stage, full)
	if err != nil {
		return fmt.Errorf("mechanisms brief: %w", err)
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), brief); err != nil {
		return fmt.Errorf("mechanisms brief: write: %w", err)
	}
	return nil
}

func renderMechanismsBrief(registry mechanisms.Registry, role, stage string, full bool) (string, error) {
	if strings.TrimSpace(stage) == "" {
		if full {
			return registry.BriefFull(role), nil
		}
		return registry.Brief(role), nil
	}
	canonicalStage := domain.Stage(strings.ToUpper(strings.TrimSpace(stage)))
	if full {
		brief, err := registry.BriefFullForStage(role, canonicalStage)
		if err != nil {
			return "", fmt.Errorf("render full mechanisms brief for Stage: %w", err)
		}
		return brief, nil
	}
	brief, err := registry.BriefForStage(role, canonicalStage)
	if err != nil {
		return "", fmt.Errorf("render mechanisms brief for Stage: %w", err)
	}
	return brief, nil
}
