package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/spf13/cobra"
)

// NormalizeOptions carries the `mission normalize-openspec` flag values.
type NormalizeOptions struct{ Root, MissionID, ChangeID, RuntimeRoot, Pending string }

// NormalizeDependencies injects mission-id validation and path resolution.
type NormalizeDependencies struct {
	RootFlag         string
	RequireMissionID func(string) error
	ResolvePaths     func(NormalizeOptions) (string, string, string, error)
}

// NewNormalizeOpenSpec builds `mission normalize-openspec`.
func NewNormalizeOpenSpec(deps NormalizeDependencies) *cobra.Command {
	opts := NormalizeOptions{}
	cmd := &cobra.Command{
		Use:   "normalize-openspec",
		Short: "Publish a completed OpenSpec change into the refined mission package",
		Long: `Validates a private OpenSpec change and atomically promotes its proposal,
design, tasks, and the mission analysis into <base_path>/refined/<mission_id>.
OpenSpec specs and archive history remain private provider scratch.`,
	}
	f := cmd.Flags()
	f.StringVar(&opts.Root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.MissionID, "mission-id", "", "canonical Strategist mission id (required)")
	f.StringVar(&opts.ChangeID, "change-id", "", "completed OpenSpec change id (required)")
	f.StringVar(&opts.RuntimeRoot, "runtime-root", "", "OpenSpec runtime root (default: <strategist-root>/openspec)")
	f.StringVar(&opts.Pending, "pending-analysis", "", "pending analysis path (default: <base_path>/pending/<mission-id>-analysis.md)")
	if err := cmd.MarkFlagRequired("change-id"); err != nil {
		panic(err)
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunNormalizeOpenSpec(cmd, deps, opts) }
	return cmd
}

// RunNormalizeOpenSpec publishes a completed OpenSpec change through
// refinement.NormalizeOpenSpec.
func RunNormalizeOpenSpec(cmd *cobra.Command, deps NormalizeDependencies, opts NormalizeOptions) error {
	if err := deps.RequireMissionID(opts.MissionID); err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	basePath, runtimeRoot, pending, err := deps.ResolvePaths(opts)
	if err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	result, err := refinement.NormalizeOpenSpec(refinement.OpenSpecInput{MissionID: opts.MissionID, BasePath: basePath, RuntimeRoot: runtimeRoot, ChangeID: opts.ChangeID, PendingAnalysisPath: pending})
	if err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "mission_id=%s provider_change_id=%s refined=%s status=archivist_done\n", opts.MissionID, result.ProviderChangeID, result.RefinedPath); err != nil {
		return fmt.Errorf("mission normalize-openspec: write output: %w", err)
	}
	return nil
}
