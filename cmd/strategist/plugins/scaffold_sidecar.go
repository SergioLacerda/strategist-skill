package plugins

import (
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/spf13/cobra"
)

// NewScaffoldSidecar creates `plugins scaffold-sidecar`.
func NewScaffoldSidecar() *cobra.Command {
	opts := install.SidecarScaffoldOptions{}
	cmd := &cobra.Command{
		Use:   "scaffold-sidecar <package-dir>",
		Short: "Generate a package's strategist.yaml sidecar deterministically",
		Long: `Generates <package-dir>/strategist.yaml from the operator's declaration and
the package (ADR-0061). --role and --slot are required and are the human
confirmation; they are never inferred. version comes from SKILL.md
metadata.version, or from --version when the package declares none; risk_score, category and the ranger weapon_contract are
derived by fixed rules. upstream_* and license are filled only from an explicit
--provenance file. openspec_root additionally needs the package's runtime/
bundle, runtime.lock.yaml and a runtime: block in the provenance file.

An existing sidecar is never overwritten without --force. --check writes
nothing and fails non-zero when the file is missing or differs from the
regenerated one. The command touches only <package-dir>/strategist.yaml.`,
		Args: cobra.ExactArgs(1),
	}
	cmd.Flags().StringArrayVar(&opts.Roles, "role", nil, "Role the Weapon serves (repeatable, required)")
	cmd.Flags().StringArrayVar(&opts.Slots, "slot", nil, "slot the Weapon supports (repeatable, required)")
	cmd.Flags().StringVar(&opts.Version, "version", "", "package version, used only when SKILL.md carries no metadata.version")
	cmd.Flags().StringVar(&opts.Runtime, "runtime", "embedded", "runtime kind: embedded or openspec_root")
	cmd.Flags().StringVar(&opts.ProvenancePath, "provenance", "", "explicit provenance file supplying upstream_*, license and the openspec_root runtime block")
	cmd.Flags().BoolVar(&opts.Default, "default", false, "mark the Weapon as the default for its slot")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "overwrite an existing, differing sidecar")
	cmd.Flags().BoolVar(&opts.Check, "check", false, "fail non-zero on drift instead of writing (for CI)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts.PackageDir = args[0]
		return RunScaffoldSidecar(cmd.OutOrStdout(), opts)
	}
	return cmd
}

// RunScaffoldSidecar generates (or, with Check, verifies) one package sidecar.
func RunScaffoldSidecar(out io.Writer, opts install.SidecarScaffoldOptions) error {
	result, err := install.ScaffoldSidecar(opts)
	if err != nil {
		return fmt.Errorf("scaffold-sidecar: %w", err)
	}
	if _, err := fmt.Fprintf(out, "scaffold-sidecar: %s %s\n", result.Status, result.Path); err != nil {
		return fmt.Errorf("scaffold-sidecar: write report: %w", err)
	}
	return nil
}
