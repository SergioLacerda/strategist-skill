package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

// evalHarvestCmd copies real, persisted mission artifacts into
// tests/evals/regression/ as fixtures for internal/eval's
// TargetArtifactCheck-based content assertions. Design:
// .analysis/refined/20260804-eval-harvest/design.md. Decisions (DEC-1..5):
// .analysis/archived/20260804-eval-harvest-adr.md.
var evalHarvestCmd = &cobra.Command{
	Use:   "harvest [mission_id]",
	Short: "Copy real mission artifacts into tests/evals/regression/ as fixtures",
	Long: `Copy persisted artifacts from completed Strategist missions (analysis.md by
default) into tests/evals/regression/<mission_id>/, for use as real fixture content by
internal/eval's TargetArtifactCheck-based content assertions.

Mission discovery reuses treasure.ScanMissionsTolerant — the same mechanism
behind "strategist treasure-chest index" — so only missions with a tasks.md
under <base_path>/refined/ or <base_path>/done/ are eligible. A mission
whose tasks.md fails to parse is skipped with a warning rather than
aborting the run.

No route_decision fixture type is produced: Scout's route_decision is never persisted
to disk anywhere in this codebase (see .analysis/archived/20260804-eval-harvest-adr.md
DEC-5).`,
}

type evalHarvestOptions struct {
	All     bool
	Include string
	Root    string
}

// evalHarvestArtifactFiles maps an --include value to the filename it reads
// from inside the mission directory. "adr" and "report" are handled
// separately (harvestIncludeSource, eval_harvest_copy.go) since they live
// under archived/, not inside the mission directory.
var evalHarvestArtifactFiles = map[string]string{
	"design":   "design.md",
	"proposal": "proposal.md",
	"tasks":    "tasks.md",
}

func runEvalHarvest(cmd *cobra.Command, args []string, opts evalHarvestOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.All = boolFlag(cmd, "all", opts.All)
	opts.Include = stringFlag(cmd, "include", opts.Include)

	strategistRoot, projectRoot, err := resolveEvalActionRoot(cmd, "harvest", opts.Root)
	if err != nil {
		return err
	}
	_, basePath, err := resolveDojoRoots(strategistRoot)
	if err != nil {
		return fmt.Errorf("eval harvest: %w", err)
	}

	missionIDs, warnings, err := selectHarvestMissionIDs(args, opts, basePath)
	if err != nil {
		return err
	}
	printEvalHarvestWarnings(cmd, warnings)
	includeTypes, err := parseHarvestInclude(opts.Include)
	if err != nil {
		return err
	}

	destRoot := filepath.Join(projectRoot, "tests", "evals", "regression")
	written, err := harvestMissions(basePath, destRoot, missionIDs, includeTypes)
	if err != nil {
		return err
	}
	fmt.Printf("[Strategist] eval harvest: %d mission(s), %d fixture file(s) written\n", len(missionIDs), written)
	return nil
}

// harvestMissions harvests every mission in missionIDs, returning the total
// fixture file count written across all of them.
func harvestMissions(basePath, destRoot string, missionIDs, includeTypes []string) (int, error) {
	written := 0
	for _, id := range missionIDs {
		n, err := harvestMission(basePath, destRoot, id, includeTypes)
		if err != nil {
			return written, fmt.Errorf("eval harvest %s: %w", id, err)
		}
		written += n
	}
	return written, nil
}

func init() {
	opts := evalHarvestOptions{}
	evalHarvestCmd.Flags().BoolVar(&opts.All, "all", false, "harvest every mission found by treasure.ScanMissionsTolerant")
	evalHarvestCmd.Flags().StringVar(&opts.Include, "include", "", "comma-separated extra artifact types: design,proposal,tasks,adr,report")
	evalHarvestCmd.Flags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	evalHarvestCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runEvalHarvest(cmd, args, opts)
	}
	evalCmd.AddCommand(evalHarvestCmd)
}
