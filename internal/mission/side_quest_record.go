package mission

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// Side-quest decisions recorded for an Opportunity Attack ADR.
const (
	SideQuestAccepted = "accepted"
	SideQuestDeclined = "declined"

	// SideQuestKindOAADR records an ADR opportunity decision.
	SideQuestKindOAADR = "oa_adr"
	// SideQuestKindOARunbook records a Runbook opportunity decision.
	SideQuestKindOARunbook = "oa_runbook"
	// SideQuestKindOAChest records a Treasure Chest opportunity decision.
	SideQuestKindOAChest = "oa_treasure_chest"

	sideQuestSchemaVersion = 1
)

// DestinationRule is the ADR destination rule resolved at acceptance time.
// The final file name depends on a directory scan, so only the rule is fixed
// here and the path is reserved later (see ReserveADRTarget).
type DestinationRule struct {
	// CanonicalPath is the validated project-relative ADR directory, or empty
	// when the archived fallback applies.
	CanonicalPath string `json:"canonical_path,omitempty"`
	// Fallback is the base_path-relative archived destination.
	Fallback string `json:"fallback"`
}

// AcceptedSideQuest is the durable, integrity-protected record of the user's
// decision about the mission's OA-ADR side quest. It lives at
// missions/side-quests/<mission_id>.json and is only written under the
// mission lock.
type AcceptedSideQuest struct {
	SchemaVersion int             `json:"schema_version"`
	MissionID     string          `json:"mission_id"`
	SideQuestID   string          `json:"side_quest_id"`
	Kind          string          `json:"kind"`
	Decision      string          `json:"decision"`
	Destination   DestinationRule `json:"destination"`
	// ReservedPath is the workspace-relative ADR path reserved once at
	// materialization time.
	ReservedPath string `json:"reserved_path,omitempty"`
	// Claimed is set once the Sniper claim for ReservedPath is recorded, so an
	// interrupted materialization retries only the missing step.
	Claimed bool `json:"claimed,omitempty"`
	// Sealed is set by gate_approved; a sealed record never changes decision.
	Sealed    bool   `json:"sealed,omitempty"`
	DecidedAt string `json:"decided_at"`
	SealedAt  string `json:"sealed_at,omitempty"`
	// Integrity is the digest of every other field.
	Integrity string `json:"integrity"`
}

// OAADRSideQuestID is the id the package must carry for the mission's ADR.
func OAADRSideQuestID(missionID string) string { return "OA-ADR-" + missionID }

// OARunbookSideQuestID is the independent Runbook opportunity id.
func OARunbookSideQuestID(missionID string) string { return "OA-RUNBOOK-" + missionID }

// OAChestSideQuestID is the independent Treasure Chest opportunity id.
func OAChestSideQuestID(missionID string) string { return "OA-CHEST-" + missionID }

func (r AcceptedSideQuest) digest() (string, error) {
	r.Integrity = ""
	raw, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("side_quest_record_invalid: encode record: %w", err)
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), nil
}

func (r AcceptedSideQuest) sealIntegrity() (AcceptedSideQuest, error) {
	r.SchemaVersion = sideQuestSchemaVersion
	digest, err := r.digest()
	if err != nil {
		return AcceptedSideQuest{}, err
	}
	r.Integrity = digest
	return r, nil
}

// VerifyIntegrity reports whether the record still matches its digest.
func (r AcceptedSideQuest) VerifyIntegrity() error {
	want, err := r.digest()
	if err != nil {
		return err
	}
	if r.Integrity == "" || r.Integrity != want {
		return fmt.Errorf("side_quest_record_tampered: integrity digest does not match the record for mission %q", r.MissionID)
	}
	return nil
}
