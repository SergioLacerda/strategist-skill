package main

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

// selectHarvestMissionIDs resolves which missions to harvest: either the
// single positional mission_id, or every mission treasure.ScanMissionsTolerant
// finds when --all is set. The two modes are mutually exclusive (DEC-3).
// The --all branch uses the tolerant scan so one mission with an unparseable
// side_quests_approved: block (see .analysis/archived/20260804-treasure-scan-sq-block-bug-adr.md
// DEC-1) is skipped and reported as a warning instead of aborting the whole run.
func selectHarvestMissionIDs(args []string, opts evalHarvestOptions, basePath string) ([]string, []treasure.ScanWarning, error) {
	if opts.All {
		if len(args) > 0 {
			return nil, nil, fmt.Errorf("eval harvest: mission_id and --all are mutually exclusive")
		}
		return selectAllHarvestMissionIDs(basePath)
	}
	if len(args) != 1 || args[0] == "" {
		return nil, nil, fmt.Errorf("eval harvest: exactly one mission_id is required (or use --all)")
	}
	return []string{args[0]}, nil, nil
}

// selectAllHarvestMissionIDs resolves every mission treasure.ScanMissionsTolerant
// finds under basePath into its ID, passing through any skip warnings.
func selectAllHarvestMissionIDs(basePath string) ([]string, []treasure.ScanWarning, error) {
	missions, warnings, err := treasure.ScanMissionsTolerant(basePath)
	if err != nil {
		return nil, nil, fmt.Errorf("eval harvest: %w", err)
	}
	ids := make([]string, 0, len(missions))
	for _, m := range missions {
		ids = append(ids, m.MissionID)
	}
	return ids, warnings, nil
}

// printEvalHarvestWarnings reports missions skipped by the tolerant scan,
// mirroring treasure_chest_index.go's printTreasureChestIndexWarnings shape.
func printEvalHarvestWarnings(cmd *cobra.Command, warnings []treasure.ScanWarning) {
	for _, warning := range warnings {
		cmd.PrintErrf("Warning: eval harvest: skipped inconsistent mission file: %v\n", warning)
	}
}

// parseHarvestInclude validates and splits the --include flag.
func parseHarvestInclude(include string) ([]string, error) {
	if include == "" {
		return nil, nil
	}
	var out []string
	for _, raw := range strings.Split(include, ",") {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if err := validateHarvestIncludeValue(v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// validateHarvestIncludeValue rejects any --include token that isn't one of
// the recognized artifact types ("adr"/"report", or a key of
// evalHarvestArtifactFiles).
func validateHarvestIncludeValue(v string) error {
	if v == "adr" || v == "report" {
		return nil
	}
	if _, ok := evalHarvestArtifactFiles[v]; ok {
		return nil
	}
	return fmt.Errorf("eval harvest: unknown --include value %q (want design, proposal, tasks, adr, report)", v)
}
