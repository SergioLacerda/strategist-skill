package domain

import (
	"fmt"
	"strings"
)

// ExecutionTargetKind identifies the documentation artifact kind authorized
// for Sniper execution.
const (
	ExecutionTargetKindDocumentation = "documentation"
	ExecutionTargetKindADR           = "adr"
)

// ExecutionTargetOrigin identifies how an execution target entered the
// approved mission package.
const (
	ExecutionTargetOriginRefinedPackage    = "refined_package"
	ExecutionTargetOriginAcceptedSideQuest = "accepted_side_quest"
)

// ExecutionTarget is the immutable, provider-neutral target passed from the
// approved mission boundary to Sniper.
type ExecutionTarget struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Origin      string `json:"origin"`
	SideQuestID string `json:"side_quest_id,omitempty"`
}

// Validate checks the path and required metadata before a target crosses an
// execution boundary. Kind values remain extensible for future Weapons; the
// current bridge decides which kinds it can resolve and materialize.
func (t ExecutionTarget) Validate() error {
	if err := ValidateProjectRelativePath(t.Path, "execution_target.path"); err != nil {
		return err
	}
	if strings.TrimSpace(t.Kind) == "" {
		return fmt.Errorf("execution_target.kind is required")
	}
	if strings.TrimSpace(t.Origin) == "" {
		return fmt.Errorf("execution_target.origin is required")
	}
	if t.Origin == ExecutionTargetOriginAcceptedSideQuest && strings.TrimSpace(t.SideQuestID) == "" {
		return fmt.Errorf("execution_target.side_quest_id is required for accepted side-quest targets")
	}
	return nil
}
