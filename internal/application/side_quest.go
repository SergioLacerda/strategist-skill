package application

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// SideQuestDecisionRequest is the application-owned input for recording a
// mission side-quest decision. Concrete durable records remain adapters.
type SideQuestDecisionRequest struct {
	StrategistRoot string
	BasePath       string
	MissionID      string
	SideQuestID    string
	Decision       string
	State          domain.MissionState
	CanonicalPath  string
	Now            time.Time
}

// DestinationRule is the stable application projection of an ADR destination.
type DestinationRule struct {
	CanonicalPath string
	Fallback      string
}

// SideQuestDecisionResult is the consumer-facing side-quest record.
type SideQuestDecisionResult struct {
	SchemaVersion int
	MissionID     string
	SideQuestID   string
	Kind          string
	Decision      string
	Destination   DestinationRule
	ReservedPath  string
	Claimed       bool
	Sealed        bool
	DecidedAt     string
	SealedAt      string
	Integrity     string
}

// DecideSideQuest loads the mission state before recording a decision. The
// adapter supplies the durable load and decision ports, while this service
// owns their ordering and the boundary DTO.
func DecideSideQuest(
	request SideQuestDecisionRequest,
	load func(root, missionID string) (domain.MissionState, error),
	decide func(SideQuestDecisionRequest) (SideQuestDecisionResult, error),
) (SideQuestDecisionResult, error) {
	if load == nil || decide == nil {
		return SideQuestDecisionResult{}, fmt.Errorf("side quest decision ports are required")
	}
	state, err := load(request.StrategistRoot, request.MissionID)
	if err != nil {
		return SideQuestDecisionResult{}, err
	}
	request.State = state
	return decide(request)
}

// ADRTargetRequest is the application-owned input for resolving an accepted
// OA-ADR destination at execution materialization time.
type ADRTargetRequest struct {
	StrategistRoot string
	BasePath       string
	MissionID      string
	Slug           string
	Now            time.Time
}

// ReserveADRTarget loads the mission state before invoking the durable target
// resolver. The resolver remains injected so filesystem reservation and claim
// persistence stay outside the application package.
func ReserveADRTarget(
	request ADRTargetRequest,
	load func(root, missionID string) (domain.MissionState, error),
	reserve func(ADRTargetRequest, domain.MissionState) (string, error),
) (string, error) {
	if load == nil || reserve == nil {
		return "", fmt.Errorf("ADR target ports are required")
	}
	state, err := load(request.StrategistRoot, request.MissionID)
	if err != nil {
		return "", err
	}
	return reserve(request, state)
}
