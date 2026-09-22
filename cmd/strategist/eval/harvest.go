package eval

import (
	"fmt"
	"path/filepath"

	treasure "github.com/SergioLacerda/strategist-skill/treasure-chest"
	"github.com/spf13/cobra"
)

// HarvestOptions holds command line flags for the eval harvest command.
type HarvestOptions struct {
	All           bool
	Include, Root string
}

// HarvestDependencies defines injected dependencies for the eval harvest command.
type HarvestDependencies struct {
	RootFlag        string
	ResolveRoot     func(*cobra.Command, string, string) (string, string, error)
	SilenceRun      func(*cobra.Command)
	ResolveBasePath func(string) (string, string, error)
	Select          func([]string, HarvestOptions, string) ([]string, []treasure.ScanWarning, error)
	PrintWarnings   func(*cobra.Command, []treasure.ScanWarning)
	ParseInclude    func(string) ([]string, error)
	Harvest         func(string, string, []string, []string) (int, error)
}

// NewHarvest creates a new Cobra command for harvesting real mission artifacts.
func NewHarvest(deps HarvestDependencies) *cobra.Command {
	opts := HarvestOptions{}
	cmd := &cobra.Command{Use: "harvest [mission_id]", Short: "Copy real mission artifacts into tests/evals/regression/ as fixtures"}
	f := cmd.Flags()
	f.BoolVar(&opts.All, "all", false, "harvest every mission found")
	f.StringVar(&opts.Include, "include", "", "comma-separated extra artifact types")
	f.StringVar(&opts.Root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return RunHarvest(cmd, args, deps, opts) }
	return cmd
}

// RunHarvest executes the harvest command logic.
func RunHarvest(cmd *cobra.Command, args []string, deps HarvestDependencies, opts HarvestOptions) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	strategistRoot, projectRoot, err := deps.ResolveRoot(cmd, "harvest", opts.Root)
	if err != nil {
		return err
	}
	_, basePath, err := deps.ResolveBasePath(strategistRoot)
	if err != nil {
		return fmt.Errorf("eval harvest: %w", err)
	}
	missionIDs, warnings, err := deps.Select(args, opts, basePath)
	if err != nil {
		return err
	}
	deps.PrintWarnings(cmd, warnings)
	includeTypes, err := deps.ParseInclude(opts.Include)
	if err != nil {
		return err
	}
	written, err := deps.Harvest(basePath, filepath.Join(projectRoot, "tests", "evals", "regression"), missionIDs, includeTypes)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "[Strategist] eval harvest: %d mission(s), %d fixture file(s) written\n", len(missionIDs), written); err != nil {
		return fmt.Errorf("eval harvest: write output: %w", err)
	}
	return nil
}
