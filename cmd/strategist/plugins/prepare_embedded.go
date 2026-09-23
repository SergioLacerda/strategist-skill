package plugins

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/spf13/cobra"
)

// NewPrepareEmbedded creates `plugins prepare-embedded`.
func NewPrepareEmbedded() *cobra.Command {
	opts := install.PrepareEmbeddedOptions{}
	var check bool
	cmd := &cobra.Command{
		Use:   "prepare-embedded",
		Short: "Ingest external-skills-source/ into the embedded plugin catalog",
		Long: `Resolves every ORKA-shaped package under --source (each subdirectory
paired with a strategist.yaml project-adapter sidecar declaring
canonical_role/risk_score), verifies it, and merges the result into
internal/embed/defaults/plugins/catalog.yaml plus generated
internal/embed/defaults/skills/<id>/skill.yaml mirrors, recording exactly
what was embedded in a committed lock file.

This is a maintainer/CI operation on this repository's own embedded
defaults (ADR-0032's pre-build ingestion pattern, generalized from Treasure
Chest to role-slot skills — see
.analysis/refined/20260913-embedded-skill-directory-catalog). It is not a
per-workspace end-user command and does not touch .strategist/.

Use --check in CI to fail non-zero on drift instead of writing.`,
	}
	cmd.Flags().StringVar(&opts.Source, "source", "external-skills-source", "directory of ORKA-shaped skill packages to ingest")
	cmd.Flags().StringVar(&opts.DefaultsRoot, "defaults-root", "internal/embed/defaults", "internal/embed/defaults root to generate catalog.yaml and skills/<id>/skill.yaml into")
	cmd.Flags().StringVar(&opts.LockPath, "lock", install.EmbeddedSkillLockFileName, "path to the committed embedded-skill lock file")
	cmd.Flags().BoolVar(&check, "check", false, "fail non-zero on drift instead of writing (for CI)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunPrepareEmbedded(cmd.OutOrStdout(), opts, check)
	}
	return cmd
}

// RunPrepareEmbedded ingests (or, with check, verifies) the embedded catalog.
func RunPrepareEmbedded(out io.Writer, opts install.PrepareEmbeddedOptions, check bool) error {
	if check {
		return checkPrepareEmbedded(out, opts)
	}
	report, err := install.PrepareEmbedded(opts)
	if err != nil {
		return fmt.Errorf("prepare-embedded: %w", err)
	}
	if err := printPrepareEmbeddedReport(out, report); err != nil {
		return err
	}
	return writeReport(fmt.Fprintf(out, "prepare-embedded: wrote %s, %d mirror(s), and %s\n", filepath.Join(opts.DefaultsRoot, "plugins", "catalog.yaml"), len(report.Ingested), opts.LockPath))
}

func printPrepareEmbeddedReport(out io.Writer, report install.PrepareEmbeddedReport) error {
	for _, rejection := range report.Rejected {
		if err := writeReport(fmt.Fprintf(out, "prepare-embedded: rejected %s: %s\n", rejection.ID, rejection.Reason)); err != nil {
			return err
		}
	}
	for _, skill := range report.Ingested {
		if err := writeReport(fmt.Fprintf(out, "prepare-embedded: ingested %s (digest=%s)\n", skill.ID, skill.Package.Digest)); err != nil {
			return err
		}
	}
	return nil
}

func writeReport(_ int, err error) error {
	if err != nil {
		return fmt.Errorf("prepare-embedded: write report: %w", err)
	}
	return nil
}

// checkPrepareEmbedded is the --check branch of RunPrepareEmbedded: it reports
// what would be ingested and fails on drift instead of writing.
func checkPrepareEmbedded(out io.Writer, opts install.PrepareEmbeddedOptions) error {
	report, drift, err := install.CheckEmbeddedDrift(opts)
	if printErr := printPrepareEmbeddedReport(out, report); printErr != nil {
		return printErr
	}
	if err != nil {
		return fmt.Errorf("prepare-embedded --check: %w", err)
	}
	if drift {
		return fmt.Errorf("prepare-embedded --check: drift detected — run `strategist plugins prepare-embedded` and commit the result")
	}
	return writeReport(fmt.Fprintln(out, "prepare-embedded --check: no drift"))
}
