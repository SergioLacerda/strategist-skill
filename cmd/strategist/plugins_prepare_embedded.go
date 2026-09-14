package main

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/spf13/cobra"
)

var pluginsPrepareEmbeddedCmd = &cobra.Command{
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

func runPluginsPrepareEmbedded(opts install.PrepareEmbeddedOptions, check bool) error {
	if check {
		report, drift, err := install.CheckEmbeddedDrift(opts)
		printPrepareEmbeddedReport(report)
		if err != nil {
			return fmt.Errorf("prepare-embedded --check: %w", err)
		}
		if drift {
			return fmt.Errorf("prepare-embedded --check: drift detected — run `strategist plugin prepare-embedded` and commit the result")
		}
		fmt.Println("prepare-embedded --check: no drift")
		return nil
	}

	report, err := install.PrepareEmbedded(opts)
	if err != nil {
		return fmt.Errorf("prepare-embedded: %w", err)
	}
	printPrepareEmbeddedReport(report)
	fmt.Printf("prepare-embedded: wrote %s, %d mirror(s), and %s\n", filepath.Join(opts.DefaultsRoot, "plugins", "catalog.yaml"), len(report.Ingested), opts.LockPath)
	return nil
}

func printPrepareEmbeddedReport(report install.PrepareEmbeddedReport) {
	for _, rejection := range report.Rejected {
		fmt.Printf("prepare-embedded: rejected %s: %s\n", rejection.ID, rejection.Reason)
	}
	for _, skill := range report.Ingested {
		fmt.Printf("prepare-embedded: ingested %s (digest=%s)\n", skill.ID, skill.Package.Digest)
	}
}

func init() {
	opts := install.PrepareEmbeddedOptions{}
	var check bool
	pluginsPrepareEmbeddedCmd.Flags().StringVar(&opts.Source, "source", "external-skills-source", "directory of ORKA-shaped skill packages to ingest")
	pluginsPrepareEmbeddedCmd.Flags().StringVar(&opts.DefaultsRoot, "defaults-root", filepath.Join("internal", "embed", "defaults"), "internal/embed/defaults root to generate catalog.yaml and skills/<id>/skill.yaml into")
	pluginsPrepareEmbeddedCmd.Flags().StringVar(&opts.LockPath, "lock", install.EmbeddedSkillLockFileName, "path to the committed embedded-skill lock file")
	pluginsPrepareEmbeddedCmd.Flags().BoolVar(&check, "check", false, "fail non-zero on drift instead of writing (for CI)")
	pluginsPrepareEmbeddedCmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runPluginsPrepareEmbedded(opts, check)
	}
	pluginsCmd.AddCommand(pluginsPrepareEmbeddedCmd)
}
