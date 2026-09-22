package main

import (
	"fmt"

	evaladapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/eval"
	treasure "github.com/SergioLacerda/strategist-skill/treasure-chest"
	"github.com/spf13/cobra"
)

// evalHarvestCmd copies real, persisted mission artifacts into
// tests/evals/regression/ as fixtures for internal/eval's
// TargetArtifactCheck-based content assertions. Design:
// .analysis/refined/20260804-eval-harvest/design.md. Decisions (DEC-1..5):
// .analysis/archived/20260804-eval-harvest-adr.md.
var evalHarvestCmd = newEvalHarvestCommand()

func newEvalHarvestCommand() *cobra.Command {
	return evaladapter.NewHarvest(evalHarvestDependencies())
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
	if err := evaladapter.RunHarvest(cmd, args, evalHarvestDependencies(), evaladapter.HarvestOptions{All: opts.All, Include: opts.Include, Root: opts.Root}); err != nil {
		return fmt.Errorf("run eval harvest: %w", err)
	}
	return nil
}

func evalHarvestDependencies() evaladapter.HarvestDependencies {
	return evaladapter.HarvestDependencies{RootFlag: flagRoot, ResolveRoot: resolveEvalActionRoot, ResolveBasePath: resolveDojoRoots, Select: func(args []string, opts evaladapter.HarvestOptions, base string) ([]string, []treasure.ScanWarning, error) {
		return selectHarvestMissionIDs(args, evalHarvestOptions{All: opts.All, Include: opts.Include, Root: opts.Root}, base)
	}, PrintWarnings: printEvalHarvestWarnings, ParseInclude: parseHarvestInclude, Harvest: harvestMissions, SilenceRun: func(cmd *cobra.Command) {
		if run := telemetryRunFromCmd(cmd); run != nil {
			run.SetSilent()
		}
	}}
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
