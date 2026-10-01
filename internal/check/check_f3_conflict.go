package check

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
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
	return emitF3ClaimCollisionSignals(strategistRoot, now)
}

// emitF3ClaimCollisionSignals surfaces ADR-0008 F3's other tripwire signal —
// two or more distinct missions claiming the same target — from recorded
// claim history (memory/sniper-claims.jsonl). This is the Git-conflict
// signal's sibling above, following the same "strategist check reads
// recorded history and emits" pattern. Claims are now written at
// handoff_challenge_satisfied (internal/mission.RecordSniperClaims, ADR-0057 §
// A1), so this reader is no longer permanently starved.
//
// Claims belonging to a mission that has already reached a terminal phase are
// excluded before collision detection (ADR-0057 § A2 / design.md task 2.4):
// SniperClaimWindow's 30-day age cutoff cannot by itself distinguish "still
// working" from "finished yesterday", so two sequential, fully completed
// missions touching the same target inside that window would otherwise
// report a false collision forever. A mission whose state cannot be read is
// conservatively treated as still live — an unreadable mission is a reason to
// keep surfacing a possible collision, not to suppress it.
func emitF3ClaimCollisionSignals(strategistRoot string, now time.Time) error {
	claims, err := telemetry.ReadRecentSniperClaims(
		telemetry.SniperClaimHistoryPath(strategistRoot),
		now,
		telemetry.SniperClaimWindow,
	)
	if err != nil {
		return fmt.Errorf("read recent sniper claims: %w", err)
	}
	live := liveClaimsOnly(strategistRoot, claims)
	for _, signal := range telemetry.DetectClaimCollisions(live) {
		telemetry.EmitClaimCollisionSignal(signal)
	}
	return nil
}

// liveClaimsOnly filters out claims whose owning mission has already reached
// a terminal phase, per emitF3ClaimCollisionSignals's doc comment. It caches
// one phase lookup per distinct mission id so a claim history with many
// records from the same mission reads that mission's state file once.
func liveClaimsOnly(strategistRoot string, claims []telemetry.SniperClaimRecord) []telemetry.SniperClaimRecord {
	terminal := make(map[string]bool)
	live := make([]telemetry.SniperClaimRecord, 0, len(claims))
	for _, claim := range claims {
		done, cached := terminal[claim.MissionID]
		if !cached {
			phase, found := livemission.ReadMissionPhase(strategistRoot, claim.MissionID)
			done = found && livemission.Terminal(phase)
			terminal[claim.MissionID] = done
		}
		if !done {
			live = append(live, claim)
		}
	}
	return live
}

func readGitConflictedPathsFromWorktree(worktreeRoot string) ([]string, error) {
	//nolint:gosec // G204: read-only fixed git subcommand; worktreeRoot is the discovered workspace root.
	probe := exec.Command("git", "-C", worktreeRoot, "rev-parse", "--is-inside-work-tree")
	if err := probe.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// git ran and deterministically said "not inside a work tree" —
			// nothing to check, and no locale/version-dependent text to match.
			return nil, nil
		}
		// probe could not even run (invalid path, git not found, etc.) — a real
		// operational error, not a "no git repo here" signal. Propagate it.
		return nil, fmt.Errorf("read git conflicted paths: probe worktree: %w", err)
	}
	//nolint:gosec // G204: read-only fixed git subcommand; worktreeRoot is the discovered workspace root.
	cmd := exec.Command("git", "-C", worktreeRoot, "diff", "--name-only", "--diff-filter=U")
	out, err := cmd.CombinedOutput()
	if err != nil {
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
