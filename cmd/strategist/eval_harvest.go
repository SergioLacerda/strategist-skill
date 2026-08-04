package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
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
// separately (harvestIncludeSource) since they live under archived/, not
// inside the mission directory.
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
	written := 0
	for _, id := range missionIDs {
		n, err := harvestMission(basePath, destRoot, id, includeTypes)
		if err != nil {
			return fmt.Errorf("eval harvest %s: %w", id, err)
		}
		written += n
	}
	fmt.Printf("[Strategist] eval harvest: %d mission(s), %d fixture file(s) written\n", len(missionIDs), written)
	return nil
}

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
	if len(args) != 1 || args[0] == "" {
		return nil, nil, fmt.Errorf("eval harvest: exactly one mission_id is required (or use --all)")
	}
	return []string{args[0]}, nil, nil
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
		if v != "adr" && v != "report" {
			if _, ok := evalHarvestArtifactFiles[v]; !ok {
				return nil, fmt.Errorf("eval harvest: unknown --include value %q (want design, proposal, tasks, adr, report)", v)
			}
		}
		out = append(out, v)
	}
	return out, nil
}

// harvestMission copies the default analysis.md plus every requested
// include type for one mission into destRoot/<mission_id>/, and returns how
// many fixture files were written. Re-running on the same mission
// overwrites the existing copy (DEC-4 — no versioning).
func harvestMission(basePath, destRoot, missionID string, includeTypes []string) (int, error) {
	srcDir, err := missionDir(basePath, missionID)
	if err != nil {
		return 0, err
	}
	destDir := filepath.Join(destRoot, missionID)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return 0, fmt.Errorf("create %s: %w", destDir, err)
	}

	written := 0
	if err := copyHarvestFile(filepath.Join(srcDir, "analysis.md"), filepath.Join(destDir, "analysis.md")); err != nil {
		return written, err
	}
	written++

	for _, t := range includeTypes {
		src, dest := harvestIncludeSource(basePath, srcDir, destDir, missionID, t)
		if err := copyHarvestFile(src, dest); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

// harvestIncludeSource resolves the source/destination pair for one
// --include type. "adr"/"report" live under <base_path>/archived/, never
// inside the mission directory (see 09-response.md § Artifact Contract).
func harvestIncludeSource(basePath, srcDir, destDir, missionID, includeType string) (src, dest string) {
	switch includeType {
	case "adr":
		return filepath.Join(basePath, "archived", missionID+"-adr.md"), filepath.Join(destDir, "adr.md")
	case "report":
		return filepath.Join(basePath, "archived", missionID+"-report.md"), filepath.Join(destDir, "report.md")
	default:
		filename := evalHarvestArtifactFiles[includeType]
		return filepath.Join(srcDir, filename), filepath.Join(destDir, filename)
	}
}

// missionDir resolves a mission's directory under refined/ or done/, in
// that order. treasure.ScannedMission carries no file path, so harvest
// resolves the directory itself.
func missionDir(basePath, missionID string) (string, error) {
	for _, sub := range []string{"refined", "done"} {
		candidate := filepath.Join(basePath, sub, missionID)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("mission %q not found under refined/ or done/", missionID)
}

func copyHarvestFile(src, dest string) error {
	in, err := os.Open(filepath.Clean(src)) //nolint:gosec // G304: harvest source paths are resolved from workspace mission directories under base_path, not raw user input
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	defer in.Close() //nolint:errcheck // read-only file; close error is not actionable

	out, err := os.Create(filepath.Clean(dest)) //nolint:gosec // G304: harvest destination is a computed path under tests/evals/regression/
	if err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}
	defer out.Close() //nolint:errcheck // best-effort close on error paths below; the happy-path close is checked explicitly

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dest, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", dest, err)
	}
	return nil
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
