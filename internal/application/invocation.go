package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// InvocationCompletionRequest is the adapter-neutral input for consuming one
// host completion. Host normalization and persistence remain behind the
// callback supplied by the composition root.
type InvocationCompletionRequest struct {
	Root       string
	BasePath   string
	RequestID  string
	Completion domain.MissionInvocationCompletion
	Adapter    domain.MissionExecutionAdapter
}

// InvocationBuildRequest identifies the immutable request to build for one
// role and slot before optional host dispatch.
type InvocationBuildRequest struct {
	Root           string
	BasePath       string
	MissionID      string
	Role           string
	Slot           string
	RequestContext string
}

// BuildInvocation delegates request construction through the application
// boundary while keeping provider/runtime assembly in the adapter.
func BuildInvocation(ctx context.Context, request InvocationBuildRequest, build func(context.Context, InvocationBuildRequest) (domain.MissionInvocationRequest, error)) (domain.MissionInvocationRequest, error) {
	if build == nil {
		return domain.MissionInvocationRequest{}, errors.New("mission invoke: invocation runtime is unavailable")
	}
	return build(ctx, request)
}

// CompleteInvocation validates the one-shot completion boundary and delegates
// the concrete runtime operation through an application port.
func CompleteInvocation(
	ctx context.Context,
	request InvocationCompletionRequest,
	complete func(context.Context, InvocationCompletionRequest) (domain.MissionInvocationOutcome, error),
) (domain.MissionInvocationOutcome, error) {
	if complete == nil {
		return domain.MissionInvocationOutcome{}, errors.New("mission complete: invocation runtime is unavailable")
	}
	if request.RequestID == "" {
		return domain.MissionInvocationOutcome{}, errors.New("--request-id is required")
	}
	return complete(ctx, request)
}

// ValidateInvocationDispatch enforces the explicit-host boundary before a
// request can be persisted or executed through a child bridge.
func ValidateInvocationDispatch(host, requestContext, slot string) error {
	if host == "" {
		return nil
	}
	if requestContext == "" {
		return errors.New("--context is required with --host")
	}
	if slot == string(domain.SlotExecution) {
		return errors.New("execution requires the current-host adapter; child host bridges are read-only")
	}
	if _, err := domain.ChildAdapterForHost(host); err != nil {
		return fmt.Errorf("invocation_adapter_unknown: %w", err)
	}
	return nil
}

// ValidateInvocationPhase ensures a slot is invoked only in its mission
// phase, before any runtime request is built.
func ValidateInvocationPhase(status domain.MissionEngineStatus, slot string) error {
	expected, ok := map[string]domain.PipelinePhase{
		string(domain.SlotDiscovery):  domain.PhaseDiscovery,
		string(domain.SlotRefinement): domain.PhaseRefinement,
		string(domain.SlotExecution):  domain.PhaseExecution,
	}[slot]
	if !ok {
		return fmt.Errorf("slot %q has no host invocation boundary", slot)
	}
	if status.Phase != expected {
		return fmt.Errorf("slot %q requires phase %q, got %q", slot, expected, status.Phase)
	}
	return nil
}

// PrepareInvocation loads the mission state and validates the slot boundary
// before a one-shot request is built or persisted.
func PrepareInvocation(root, missionID, slot string, load func(root, missionID string) (domain.MissionEngineStatus, error)) error {
	if load == nil {
		return errors.New("mission lifecycle is unavailable")
	}
	status, err := load(root, missionID)
	if err != nil {
		return err
	}
	return ValidateInvocationPhase(status, slot)
}
