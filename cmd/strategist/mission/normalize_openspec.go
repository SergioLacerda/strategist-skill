package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/spf13/cobra"
)

// NormalizeOptions carries the `mission normalize-openspec` flag values.
type NormalizeOptions struct {
	Root, MissionID, ChangeID, RuntimeRoot, Pending string
	// Amend switches to the post-publication amendment mode; Amends and
	// AuthorizationRef are its required companions.
	Amend                    bool
	Amends, AuthorizationRef string
}

// NormalizeDependencies injects mission-id validation and path resolution.
type NormalizeDependencies struct {
	RootFlag         string
	RequireMissionID func(string) error
	ResolvePaths     func(NormalizeOptions) (string, string, string, error)
	// GateLabel returns the mission's gate outcome label ("" when none); it is
	// consulted only by --amend. Nil means no label is known.
	GateLabel func(NormalizeOptions) (string, error)
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
	f.BoolVar(&opts.Amend, "amend", false, "amend an already published package with a new change instead of publishing (analysis.md, the mission status and the original provider_change_id are kept; the previous files are snapshotted under .amendments/)")
	f.StringVar(&opts.Amends, "amends", "", "with --amend: the change being amended (the package's provider_change_id, or the previous amendment's change id)")
	f.StringVar(&opts.AuthorizationRef, "authorization-ref", "", "with --amend: the human authorization for the amendment (a quote or a gate event), recorded verbatim")
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
	if err := validateAmendFlags(cmd, opts); err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	basePath, runtimeRoot, pending, err := deps.ResolvePaths(opts)
	if err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	if opts.Amend {
		return runAmend(cmd, deps, opts, basePath, runtimeRoot)
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
