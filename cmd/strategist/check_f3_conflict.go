package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func emitF3ConflictAttributionSignals(strategistRoot, basePath string, now time.Time) error {
	worktreeRoot := filepath.Dir(strategistRoot)
	conflicted, err := readGitConflictedPaths(worktreeRoot)
	if err != nil {
		return err
	}
	records, err := telemetry.ReadRecentSniperMaterializations(
		telemetry.SniperMaterializationHistoryPath(strategistRoot),
		now,
		telemetry.SniperMaterializationWindow,
	)
	if err != nil {
		return fmt.Errorf("read recent sniper materializations: %w", err)
	}
	for _, signal := range telemetry.SniperConflictSignals(basePath, conflicted, records) {
		telemetry.EmitSniperConflictSignal(signal)
	}
	return nil
}

func readGitConflictedPathsFromWorktree(worktreeRoot string) ([]string, error) {
	//nolint:gosec // G204: read-only fixed git subcommand; worktreeRoot is the discovered workspace root.
	cmd := exec.Command("git", "-C", worktreeRoot, "diff", "--name-only", "--diff-filter=U")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "not a git repository") {
			return nil, nil
		}
		return nil, fmt.Errorf("read git conflicted paths: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return parseGitPathLines(string(out)), nil
}

func parseGitPathLines(out string) []string {
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(path))
		if clean != "." {
			paths = append(paths, clean)
		}
	}
	return paths
}
