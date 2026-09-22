package main

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/spf13/cobra"
)

type missionNormalizeOpenSpecOptions struct {
	Root        string
	MissionID   string
	ChangeID    string
	RuntimeRoot string
	Pending     string
}

var missionNormalizeOpenSpecCmd = newMissionNormalizeOpenSpecCommand()

func newMissionNormalizeOpenSpecCommand() *cobra.Command {
	opts := missionNormalizeOpenSpecOptions{}
	cmd := &cobra.Command{
		Use:   "normalize-openspec",
		Short: "Publish a completed OpenSpec change into the refined mission package",
		Long: `Validates a private OpenSpec change and atomically promotes its proposal,
design, tasks, and the mission analysis into <base_path>/refined/<mission_id>.
OpenSpec specs and archive history remain private provider scratch.`,
	}
	cmd.Flags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().StringVar(&opts.MissionID, "mission-id", "", "canonical Strategist mission id (required)")
	cmd.Flags().StringVar(&opts.ChangeID, "change-id", "", "completed OpenSpec change id (required)")
	cmd.Flags().StringVar(&opts.RuntimeRoot, "runtime-root", "", "OpenSpec runtime root (default: <strategist-root>/openspec)")
	cmd.Flags().StringVar(&opts.Pending, "pending-analysis", "", "pending analysis path (default: <base_path>/pending/<mission-id>-analysis.md)")
	if err := cmd.MarkFlagRequired("change-id"); err != nil {
		panic(err)
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runMissionNormalizeOpenSpec(cmd, opts)
	}
	return cmd
}

func runMissionNormalizeOpenSpec(cmd *cobra.Command, opts missionNormalizeOpenSpecOptions) error {
	if !validNormalizeMissionID(opts.MissionID) {
		return fmt.Errorf("mission normalize-openspec: --mission-id must use lowercase letters, digits, and hyphens")
	}
	basePath, runtimeRoot, pending, err := resolveNormalizePaths(opts)
	if err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	result, err := refinement.NormalizeOpenSpec(refinement.OpenSpecInput{
		MissionID: opts.MissionID, BasePath: basePath, RuntimeRoot: runtimeRoot,
		ChangeID: opts.ChangeID, PendingAnalysisPath: pending,
	})
	if err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "mission_id=%s provider_change_id=%s refined=%s status=archivist_done\n", opts.MissionID, result.ProviderChangeID, result.RefinedPath)
	if err != nil {
		return fmt.Errorf("mission normalize-openspec: write output: %w", err)
	}
	return nil
}

func validNormalizeMissionID(missionID string) bool {
	return missionID != "" && missionIDPattern.MatchString(missionID)
}

func resolveNormalizePaths(opts missionNormalizeOpenSpecOptions) (string, string, string, error) {
	strategistRoot, basePath, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve active base path: %w", err)
	}
	projectRoot := filepath.Dir(strategistRoot)
	return basePath, resolvePath(opts.RuntimeRoot, filepath.Join(strategistRoot, "openspec"), projectRoot),
		resolvePath(opts.Pending, filepath.Join(basePath, "pending", opts.MissionID+"-analysis.md"), projectRoot), nil
}

func resolvePath(value, fallback, projectRoot string) string {
	if value == "" {
		return fallback
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(projectRoot, value)
}
