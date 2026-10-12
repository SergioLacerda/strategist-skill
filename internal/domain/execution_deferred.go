package domain

import (
	"fmt"
	"strings"
)

// RunbookSidecarKind is the artifact kind of the typed `.runbook.yaml` sidecar
// of an accepted runbook. Completion verifies it with the sidecar parser.
const RunbookSidecarKind = "runbook_sidecar"

// DeferredReasonNoExecutionResolver marks an accepted side quest whose kind has
// no execution resolver: it is reported, never executed and never silently
// dropped.
const DeferredReasonNoExecutionResolver = "no_execution_resolver"

// DeferredIntent is an accepted intent that execution deliberately skipped. It
// is part of the immutable snapshot so the notice survives replay and is
// visible to Sniper, the mission view and the report. It is never an artifact
// and never extends the write scope.
type DeferredIntent struct {
	SideQuestID string `json:"side_quest_id"`
	Kind        string `json:"kind"`
	Origin      string `json:"origin"`
	Reason      string `json:"reason"`
}

// Validate checks that the notice identifies the intent and the reason.
func (d DeferredIntent) Validate() error {
	if strings.TrimSpace(d.SideQuestID) == "" || strings.TrimSpace(d.Kind) == "" ||
		strings.TrimSpace(d.Origin) == "" || strings.TrimSpace(d.Reason) == "" {
		return fmt.Errorf("execution_deferred_invalid: side_quest_id, kind, origin and reason are required")
	}
	return nil
}

func validateDeferredIntents(deferred []DeferredIntent) error {
	seen := make(map[string]struct{}, len(deferred))
	for _, d := range deferred {
		if err := d.Validate(); err != nil {
			return err
		}
		if _, exists := seen[d.SideQuestID]; exists {
			return fmt.Errorf("execution_deferred_invalid: duplicate side quest %q", d.SideQuestID)
		}
		seen[d.SideQuestID] = struct{}{}
	}
	return nil
}
