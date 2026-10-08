package mission

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
)

// SideQuestDecision is one user decision about the mission's OA-ADR.
type SideQuestDecision struct {
	StrategistRoot string
	BasePath       string
	MissionID      string
	SideQuestID    string
	Decision       string
	State          domain.MissionState
	// CanonicalPath is active.yaml#adr.canonical_path (empty when absent).
	CanonicalPath string
	Now           time.Time
}

// ResolveDestinationRule validates the configured canonical ADR directory
// (project-relative, inside the workspace) and builds the rule recorded at
// acceptance. The archived fallback is always recorded.
func ResolveDestinationRule(missionID, canonicalPath string) (DestinationRule, error) {
	rule := DestinationRule{Fallback: filepath.ToSlash(filepath.Join("archived", missionID+"-adr.md"))}
	if canonicalPath == "" {
		return rule, nil
	}
	normalized := strings.ReplaceAll(canonicalPath, `\`, "/")
	clean := path.Clean(normalized)
	if isAbsoluteProjectPath(normalized) || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return DestinationRule{}, fmt.Errorf("adr_destination_invalid: adr.canonical_path %q must be a project-relative directory inside the workspace", canonicalPath)
	}
	rule.CanonicalPath = filepath.ToSlash(clean)
	return rule, nil
}

func isAbsoluteProjectPath(value string) bool {
	if path.IsAbs(value) || filepath.IsAbs(value) || strings.HasPrefix(value, "//") {
		return true
	}
	return len(value) >= 3 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':' && value[2] == '/'
}

// DecideSideQuest records the user's acceptance or decline of
// OA-ADR-<mission_id>. The caller must hold the mission lock. It validates
// the id against the package's side_quests_approved list and the mission
// state before writing, so every rejection leaves state unchanged. Repeating
// the same decision is idempotent; the opposite decision is allowed only
// while the record is neither sealed nor reserved.
func DecideSideQuest(req SideQuestDecision) (AcceptedSideQuest, error) {
	if err := validateDecisionRequest(req); err != nil {
		return AcceptedSideQuest{}, err
	}
	existing, found, err := LoadAcceptedSideQuest(req.StrategistRoot, req.MissionID)
	if err != nil {
		return AcceptedSideQuest{}, err
	}
	if rec, done, err := checkExistingDecision(existing, found, req.Decision); done || err != nil {
		return rec, err
	}
	rule, err := decisionRule(req, existing, found)
	if err != nil {
		return AcceptedSideQuest{}, err
	}
	return saveAcceptedSideQuest(req.StrategistRoot, AcceptedSideQuest{
		MissionID: req.MissionID, SideQuestID: req.SideQuestID, Kind: SideQuestKindOAADR,
		Decision: req.Decision, Destination: rule, DecidedAt: nowStamp(req.Now),
	})
}

// checkExistingDecision reports done=true when the same decision is already
// recorded (idempotent repeat) and an error when the opposite decision arrives
// after the record was sealed or its path reserved.
func checkExistingDecision(existing AcceptedSideQuest, found bool, decision string) (rec AcceptedSideQuest, done bool, err error) {
	switch {
	case !found:
		return AcceptedSideQuest{}, false, nil
	case existing.Decision == decision:
		return existing, true, nil
	case existing.Sealed || existing.ReservedPath != "":
		return AcceptedSideQuest{}, false, fmt.Errorf("side_quest_record_sealed: the %s decision for %s can no longer change", existing.Decision, existing.SideQuestID)
	}
	return AcceptedSideQuest{}, false, nil
}

func decisionRule(req SideQuestDecision, existing AcceptedSideQuest, found bool) (DestinationRule, error) {
	if req.Decision == SideQuestAccepted {
		return ResolveDestinationRule(req.MissionID, req.CanonicalPath)
	}
	if found {
		return existing.Destination, nil
	}
	return DestinationRule{}, nil
}

func validateDecisionRequest(req SideQuestDecision) error {
	if req.Decision != SideQuestAccepted && req.Decision != SideQuestDeclined {
		return fmt.Errorf("side_quest_decision_invalid: %q", req.Decision)
	}
	if want := OAADRSideQuestID(req.MissionID); req.SideQuestID != want {
		return fmt.Errorf("side_quest_unknown: only %s can be recorded for this mission, got %q", want, req.SideQuestID)
	}
	quests, err := refinement.ReadSideQuests(req.BasePath, req.MissionID)
	if err != nil {
		return fmt.Errorf("side_quest_unreadable: %w", err)
	}
	if _, ok := refinement.FindSideQuest(quests, req.SideQuestID); !ok {
		return fmt.Errorf("side_quest_unknown: %s is not in the package's side_quests_approved", req.SideQuestID)
	}
	if req.State != domain.StateApprovalGate {
		return fmt.Errorf("side_quest_wrong_state: a side quest is decided at the approval gate, mission is %s", req.State)
	}
	return nil
}

// ErrGateEventConflictsWithSideQuest is the stable rejection of an
// analysis-only terminal event while an OA-ADR is accepted.
var ErrGateEventConflictsWithSideQuest = errors.New("gate_event_conflicts_with_accepted_side_quest")

// RequireNoAcceptedSideQuest rejects the analysis-only terminal events when an
// accepted OA-ADR record exists: the accepted ADR is a documentation target
// that must reach Sniper through the handoff challenge. Every other event, and
// every mission without an accepted record, passes. The caller must hold the
// mission lock.
func RequireNoAcceptedSideQuest(strategistRoot, missionID string, event domain.MissionEngineEvent) error {
	if !domain.MissionEventRequiresNoDocumentationTargets(event) {
		return nil
	}
	rec, found, err := LoadAcceptedSideQuest(strategistRoot, missionID)
	if err != nil {
		return err
	}
	if found && rec.Decision == SideQuestAccepted {
		return fmt.Errorf("%w: %s is accepted — use gate_approved so Sniper materializes the ADR through the handoff challenge, not %s", ErrGateEventConflictsWithSideQuest, rec.SideQuestID, event)
	}
	return nil
}

// SealSideQuestOnGateApproval seals the record when gate_approved commits, so
// the decision can no longer change. Without a record, or for any other
// event, it does nothing. The caller must hold the mission lock.
func SealSideQuestOnGateApproval(strategistRoot, missionID string, event domain.MissionEngineEvent, now time.Time) error {
	if event != domain.MissionEventGateApproved {
		return nil
	}
	rec, found, err := LoadAcceptedSideQuest(strategistRoot, missionID)
	if err != nil || !found || rec.Sealed {
		return err
	}
	rec.Sealed = true
	rec.SealedAt = nowStamp(now)
	_, err = saveAcceptedSideQuest(strategistRoot, rec)
	return err
}
