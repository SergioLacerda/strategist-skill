package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// RunInvoke validates the request and emits only the bridge envelope.
func RunInvoke(cmd *cobra.Command, deps InvocationDependencies, f *invocationFlags) error {
	if deps.Build == nil || deps.WriteResult == nil {
		return fmt.Errorf("mission invoke: invocation runtime is unavailable")
	}
	root, basePath, err := resolveInvocationRuntime(deps, f)
	if err != nil {
		return err
	}
	if err := validateInvocationDispatch(f); err != nil {
		return fmt.Errorf("mission invoke: %w", err)
	}
	request, err := buildInvocationRequest(cmd, deps, f, root, basePath)
	if err != nil {
		return err
	}
	return dispatchInvocation(cmd, deps, f, root, basePath, request)
}

// validateInvocationDispatch runs before Build persists a single-use request.
// Invalid child-host dispatch must not leave an orphan pending invocation.
// The application service owns the rule; this wrapper preserves the adapter's
// focused test seam and error contract.
func validateInvocationDispatch(f *invocationFlags) error {
	if err := application.ValidateInvocationDispatch(f.host, f.requestContext, f.slot); err != nil {
		return fmt.Errorf("validate invocation dispatch: %w", err)
	}
	return nil
}

func resolveInvocationRuntime(deps InvocationDependencies, f *invocationFlags) (string, string, error) {
	if err := deps.RequireMissionID(f.missionID); err != nil {
		return "", "", err
	}
	if f.slot == "" {
		return "", "", fmt.Errorf("mission invoke: --slot is required")
	}
	root, basePath, err := deps.ResolveBasePath(f.root)
	if err != nil {
		return "", "", fmt.Errorf("mission invoke: %w", err)
	}
	if err := application.PrepareInvocation(root, f.missionID, f.slot, deps.LoadMission); err != nil {
		return "", "", fmt.Errorf("mission invoke: %w", err)
	}
	return root, basePath, nil
}

func validateInvocationPhase(status domain.MissionEngineStatus, slot string) error {
	if err := application.ValidateInvocationPhase(status, slot); err != nil {
		return fmt.Errorf("validate invocation phase: %w", err)
	}
	return nil
}

func buildInvocationRequest(cmd *cobra.Command, deps InvocationDependencies, f *invocationFlags, root, basePath string) (domain.MissionInvocationRequest, error) {
	request, err := application.BuildInvocation(cmd.Context(), application.InvocationBuildRequest{
		Root: root, BasePath: basePath, MissionID: f.missionID, Role: f.role, Slot: f.slot, RequestContext: f.requestContext,
	}, deps.Build)
	if err != nil {
		return domain.MissionInvocationRequest{}, fmt.Errorf("mission invoke: %w", err)
	}
	return request, nil
}

func dispatchInvocation(cmd *cobra.Command, deps InvocationDependencies, f *invocationFlags, root, basePath string, request domain.MissionInvocationRequest) error {
	if f.host != "" {
		return executeHostInvocation(cmd, deps, f, root, basePath, request)
	}
	return deps.WriteResult(cmd, f.asJSON, request)
}

func executeHostInvocation(cmd *cobra.Command, deps InvocationDependencies, f *invocationFlags, root, basePath string, request domain.MissionInvocationRequest) error {
	if f.requestContext == "" {
		return fmt.Errorf("mission invoke: --context is required with --host")
	}
	if deps.ExecuteHost == nil {
		return fmt.Errorf("mission invoke: host bridge is unavailable")
	}
	adapter, err := domain.ChildAdapterForHost(f.host)
	if err != nil {
		return fmt.Errorf("mission invoke: invocation_adapter_unknown: %w", err)
	}
	completion, err := deps.ExecuteHost(cmd.Context(), root, f.host, f.requestContext, request)
	if err != nil {
		return fmt.Errorf("mission invoke: role_invocation_failed: %w", err)
	}
	outcome, err := deps.Complete(cmd.Context(), InvocationCompleteInput{Root: root, BasePath: basePath, RequestID: request.RequestID, Completion: completion, Adapter: adapter, Sink: deps.sink()})
	if err != nil {
		return fmt.Errorf("mission invoke: %w", err)
	}
	return deps.WriteResult(cmd, f.asJSON, outcome)
}
