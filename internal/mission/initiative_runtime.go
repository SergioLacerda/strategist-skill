package mission

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/initiative"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	jsonlsink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/jsonl"
)

// InitiativeRuntime wires the existing INITIATIVE domain to the Scout entry
// consultation, its independent append-only ledger, and diagnostic telemetry.
type InitiativeRuntime struct {
	core      initiative.Runtime
	eventSink telemetry.EventSink
}

// NewInitiativeRuntime creates the internal runtime with the supplied policy.
func NewInitiativeRuntime(strategistRoot string, policy initiative.Policy) (InitiativeRuntime, error) {
	return NewInitiativeRuntimeWithSink(strategistRoot, policy, jsonlsink.New(telemetry.InitiativeEventHistoryPath(strategistRoot)))
}

// NewInitiativeRuntimeWithSink creates the runtime with an explicit
// authoritative event sink. Tests and host orchestration can inject a sink;
// the default constructor uses the independent local INITIATIVE JSONL stream.
func NewInitiativeRuntimeWithSink(strategistRoot string, policy initiative.Policy, eventSink telemetry.EventSink) (InitiativeRuntime, error) {
	core, err := initiative.NewRuntime(strategistRoot, policy)
	if err != nil {
		return InitiativeRuntime{}, fmt.Errorf("initiative runtime: create core: %w", err)
	}
	if eventSink == nil {
		return InitiativeRuntime{}, fmt.Errorf("initiative runtime: event sink is required")
	}
	return InitiativeRuntime{core: core, eventSink: eventSink}, nil
}

// NewDefaultInitiativeRuntime creates the runtime using the built-in policy.
func NewDefaultInitiativeRuntime(strategistRoot string) (InitiativeRuntime, error) {
	raw, err := (embed.Extractor{}).ReadFile("initiative.yaml")
	if err != nil {
		return InitiativeRuntime{}, fmt.Errorf("initiative runtime: read canonical policy: %w", err)
	}
	policy, err := initiative.Parse(raw)
	if err != nil {
		return InitiativeRuntime{}, fmt.Errorf("initiative runtime: parse canonical policy: %w", err)
	}
	mirrorPath := filepath.Join(strategistRoot, "initiative.yaml")
	if mirror, readErr := os.ReadFile(mirrorPath); readErr == nil { //nolint:gosec // mirrorPath is rooted at the selected Strategist workspace
		if !bytes.Equal(bytes.TrimSpace(raw), bytes.TrimSpace(mirror)) {
			return InitiativeRuntime{}, fmt.Errorf("initiative runtime: canonical policy mirror drift at %s", mirrorPath)
		}
	} else if !os.IsNotExist(readErr) {
		return InitiativeRuntime{}, fmt.Errorf("initiative runtime: read canonical policy mirror: %w", readErr)
	}
	return NewInitiativeRuntime(strategistRoot, policy)
}

// EnterRole resolves one advice envelope at role entry and emits one
// diagnostic event. Repeated entry for the same mission/role/run reuses the
// existing advice identity and emits a reuse marker without appending a new
// advice record.
func (r InitiativeRuntime) EnterRole(input InitiativeRoleEntry) (initiative.Advice, error) {
	advice, reused, err := r.core.EnterRole(initiative.AdviceInput{
		MissionID: input.MissionID, Role: input.Role, RunID: input.RunID,
		Trigger: initiative.TriggerInitial, Observed: input.Observed, Leveling: input.Leveling,
	})
	if err != nil {
		return initiative.Advice{}, fmt.Errorf("initiative runtime: enter role: %w", err)
	}
	if err := r.emitAdvice(advice, reused); err != nil {
		return initiative.Advice{}, err
	}
	return advice, nil
}
