package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// RecordSniperClaims records one telemetry.SniperClaimRecord per
// documentation_target path declared in the mission's approved
// <basePath>/refined/<missionID>/tasks.md — the write side of ADR-0008's F3
// revisit tripwire claim-collision signal, which internal/check's
// emitF3ClaimCollisionSignals reads back. It is called from
// cmd/strategist/mission's RunSubmit at handoff_challenge_satisfied, deriving
// target paths from the approved package rather than adding an agent
// obligation that can be forgotten (ADR-0057 § A1 / design.md Batch A).
//
// A tasks.md with no documentation_target, or no tasks.md at all, records
// nothing and is not an error: an analysis-only accepted package has nothing
// for Sniper to claim.
func RecordSniperClaims(strategistRoot, basePath, missionID string, now time.Time) (int, error) {
	return RecordSniperClaimsForEntry(strategistRoot, basePath, missionID, "", nil, now)
}

// RecordSniperClaimsForEntry projects the immutable target snapshot of one
// execution entry. It never re-parses tasks when targets are supplied.
func RecordSniperClaimsForEntry(strategistRoot, basePath, missionID, entryID string, targets []string, now time.Time) (int, error) {
	var err error
	targets, err = resolveSniperClaimTargets(basePath, missionID, targets)
	if err != nil {
		return 0, err
	}
	if len(targets) == 0 {
		return 0, nil
	}
	claimPath := telemetry.SniperClaimHistoryPath(strategistRoot)
	packageDigest := sniperClaimPackageDigest(basePath, missionID)
	return appendSniperClaims(claimPath, entryID, missionID, basePath, targets, packageDigest, now)
}

func resolveSniperClaimTargets(basePath, missionID string, targets []string) ([]string, error) {
	if targets != nil {
		return targets, nil
	}
	tasksPath := filepath.Join(basePath, "refined", missionID, "tasks.md")
	resolved, err := refinement.DocumentationTargetPaths(tasksPath)
	if err != nil {
		return nil, fmt.Errorf("record sniper claims: %w", err)
	}
	return resolved, nil
}

func sniperClaimPackageDigest(basePath, missionID string) string {
	refinedPath := filepath.Join(basePath, "refined", missionID)
	if digest, err := handoff.PackageDigest(refinedPath); err == nil {
		return digest
	}
	return ""
}

func appendSniperClaims(claimPath, entryID, missionID, basePath string, targets []string, packageDigest string, now time.Time) (int, error) {
	for _, target := range targets {
		rec := telemetry.SniperClaimRecord{ExecutionEntryID: entryID,
			MissionID: missionID, BasePath: basePath, TargetPath: target, PackageDigest: packageDigest, ClaimedAt: now,
		}
		if err := telemetry.AppendSniperClaim(claimPath, rec); err != nil {
			return 0, fmt.Errorf("record sniper claims: %w", err)
		}
	}
	return len(targets), nil
}

// ReadMissionPhase reads the persisted phase of a mission by id directly,
// without domain.RestoreMission's full validation pass — a caller filtering
// claims by whether the owning mission is still in flight only needs the
// phase. A missing or unparseable state file reports found=false rather than
// erroring: the conservative default is to treat an unreadable mission as
// still live, so a possible collision is surfaced rather than silently
// dropped (see internal/check's emitF3ClaimCollisionSignals).
func ReadMissionPhase(strategistRoot, missionID string) (phase domain.PipelinePhase, found bool) {
	data, err := os.ReadFile(filepath.Join(strategistRoot, "missions", missionID+".json")) //nolint:gosec // strategistRoot is the discovered/validated runtime root; missionID is validated by the CLI layer
	if err != nil {
		return "", false
	}
	var status domain.MissionEngineStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return "", false
	}
	return status.Phase, true
}

// Terminal reports whether phase is one of the mission engine's
// terminal phases (domain.PhaseDone) — the release predicate for a Sniper
// claim (ADR-0057 § A1 / design.md task 2.4): a mission that has completed
// released its claim on any target it materialized, since SniperClaimWindow's
// own 30-day age cutoff cannot distinguish "still working" from "finished
// yesterday". domain.PhaseBlocked is deliberately not terminal here: a
// blocked mission may still resume and its claim should keep counting.
func Terminal(phase domain.PipelinePhase) bool {
	return phase == domain.PhaseDone
}
