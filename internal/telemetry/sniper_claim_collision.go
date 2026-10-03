package telemetry

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

// ClaimCollisionSignal reports two or more distinct missions having claimed
// the same target path within the caller-supplied window — the
// claim-collision half of ADR-0008's F3 revisit tripwire.
type ClaimCollisionSignal struct {
	BasePath   string
	TargetPath string
	// MissionIDs is the sorted, deduplicated set of distinct missions that
	// claimed TargetPath within the window.
	MissionIDs []string
}

// DetectClaimCollisions groups records by TargetPath and reports one
// ClaimCollisionSignal per target claimed by two or more distinct
// MissionIDs. A single mission claiming (or re-claiming) the same target
// multiple times is not a collision — only distinct missions targeting the
// same path count, matching ADR-0008's own framing ("two or more distinct
// Sniper sessions claiming the same target").
func DetectClaimCollisions(records []SniperClaimRecord) []ClaimCollisionSignal {
	if len(records) == 0 {
		return nil
	}
	missionsByTarget, basePathByTarget, targetOrder := groupClaimsByTarget(records)
	return collisionSignals(missionsByTarget, basePathByTarget, targetOrder)
}

func groupClaimsByTarget(records []SniperClaimRecord) (map[string]map[string]bool, map[string]string, []string) {
	missionsByTarget := make(map[string]map[string]bool)
	basePathByTarget := make(map[string]string)
	var targetOrder []string
	for _, rec := range records {
		if _, ok := missionsByTarget[rec.TargetPath]; !ok {
			missionsByTarget[rec.TargetPath] = make(map[string]bool)
			basePathByTarget[rec.TargetPath] = rec.BasePath
			targetOrder = append(targetOrder, rec.TargetPath)
		}
		missionsByTarget[rec.TargetPath][rec.MissionID] = true
	}
	return missionsByTarget, basePathByTarget, targetOrder
}

func collisionSignals(missionsByTarget map[string]map[string]bool, basePathByTarget map[string]string, targetOrder []string) []ClaimCollisionSignal {
	var signals []ClaimCollisionSignal
	for _, target := range targetOrder {
		missions := missionsByTarget[target]
		if !ClaimCollisionThresholdMet(len(missions)) {
			continue
		}
		ids := make([]string, 0, len(missions))
		for id := range missions {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		signals = append(signals, ClaimCollisionSignal{
			BasePath:   basePathByTarget[target],
			TargetPath: target,
			MissionIDs: ids,
		})
	}
	return signals
}

// ClaimCollisionThresholdMet reports whether distinctMissionCount meets
// ADR-0008's F3 revisit tripwire threshold for claim-collision attribution:
// two or more distinct missions claiming the same target.
func ClaimCollisionThresholdMet(distinctMissionCount int) bool {
	return distinctMissionCount >= 2
}

// FormatClaimCollisionSignal returns a canonical progress-contract line for a claim collision signal.
func FormatClaimCollisionSignal(s ClaimCollisionSignal) string {
	return fmt.Sprintf(
		"[Strategist] signal=sniper_claim_collision base_path=%s target=%s missions=%s",
		SanitizePath(s.BasePath), SanitizePath(s.TargetPath), strings.Join(s.MissionIDs, ","),
	)
}

// EmitClaimCollisionSignal logs the signal through slog with canonical attributes.
func EmitClaimCollisionSignal(s ClaimCollisionSignal) {
	slog.Info(FormatClaimCollisionSignal(s), AttrBasePath, SanitizePath(s.BasePath), AttrTarget, SanitizePath(s.TargetPath), AttrClaimMissionIDs, strings.Join(s.MissionIDs, ","))
}
