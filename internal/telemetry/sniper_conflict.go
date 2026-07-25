package telemetry

import (
	"fmt"
	"log/slog"
)

// f3ConflictThreshold is the Git-conflict side of the ADR-0008 F3 revisit
// tripwire: three or more Git conflicts attributed to files Sniper recently
// materialized, within a rolling 30-day window that the caller is
// responsible for tracking (this package does not persist history).
//
// This is the first of two signals named in ADR-0008 § F3 revisit tripwire
// (docs/adr/0008-single-session-assumption.md). The other — two or more
// distinct Sniper sessions claiming the same target — is not instrumented
// here: it would require a cross-session claim registry, which ADR-0008
// itself, and this signal's own approved scope, explicitly rule out. Git
// conflict attribution needs no new persistent state: it classifies
// caller-supplied paths (already-known Git conflicts, already-known recent
// Sniper targets), so it was chosen as the cheaper first-pass signal.
const f3ConflictThreshold = 3

// SniperConflictSignal reports a Git conflict attributed to a documentation
// target Sniper recently materialized — the F3 revisit tripwire's
// conflict-attribution signal.
type SniperConflictSignal struct {
	MissionID     string
	BasePath      string
	TargetPath    string
	ConflictCount int // conflicts attributed to TargetPath within the caller's tracking window
}

// FormatSniperConflictSignal returns a canonical progress-contract line for a Sniper conflict signal.
func FormatSniperConflictSignal(s SniperConflictSignal) string {
	return fmt.Sprintf(
		"[Strategist] signal=sniper_conflict_attributed mission=%s base_path=%s target=%s conflict_count=%d",
		s.MissionID, SanitizePath(s.BasePath), SanitizePath(s.TargetPath), s.ConflictCount,
	)
}

// EmitSniperConflictSignal logs the signal through slog with canonical attributes.
func EmitSniperConflictSignal(s SniperConflictSignal) {
	slog.Info(
		FormatSniperConflictSignal(s),
		AttrMissionID, s.MissionID,
		AttrBasePath, SanitizePath(s.BasePath),
		AttrTarget, SanitizePath(s.TargetPath),
		AttrConflictCount, s.ConflictCount,
	)
}

// ClassifyConflictedTargets returns the subset of conflictedPaths that also
// appear in recentlyMaterialized — the intersection is the set of Git
// conflicts attributable to recent Sniper writes. Detection of each input
// list (reading Git's unmerged paths; tracking what Sniper recently wrote)
// is the caller's responsibility; this function only classifies.
func ClassifyConflictedTargets(conflictedPaths, recentlyMaterialized []string) []string {
	materialized := make(map[string]struct{}, len(recentlyMaterialized))
	for _, p := range recentlyMaterialized {
		materialized[p] = struct{}{}
	}
	var attributed []string
	for _, p := range conflictedPaths {
		if _, ok := materialized[p]; ok {
			attributed = append(attributed, p)
		}
	}
	return attributed
}

// F3ConflictThresholdMet reports whether conflictCount meets ADR-0008's F3
// revisit tripwire threshold for Git-conflict attribution (three or more
// within the caller-tracked rolling 30-day window).
func F3ConflictThresholdMet(conflictCount int) bool {
	return conflictCount >= f3ConflictThreshold
}
