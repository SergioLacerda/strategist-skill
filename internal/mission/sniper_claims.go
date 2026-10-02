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
	tasksPath := filepath.Join(basePath, "refined", missionID, "tasks.md")
	targets, err := refinement.DocumentationTargetPaths(tasksPath)
	if err != nil {
		return 0, fmt.Errorf("record sniper claims: %w", err)
	}
	if len(targets) == 0 {
		return 0, nil
	}
	claimPath := telemetry.SniperClaimHistoryPath(strategistRoot)
	packageDigest := ""
	if digest, digestErr := handoff.PackageDigest(filepath.Join(basePath, "refined", missionID)); digestErr == nil {
		packageDigest = digest
	}
	for _, target := range targets {
		rec := telemetry.SniperClaimRecord{
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
