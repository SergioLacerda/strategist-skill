package mission

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// InvocationDependencies contains the production boundary for the two-phase
// host bridge. The adapter owns file loading and normalization; this package
// owns CLI flags and JSON transport only.
type InvocationDependencies struct {
	RootFlag         string
	RequireMissionID func(string) error
	ResolveBasePath  func(string) (string, string, error)
	LoadMission      func(root, missionID string) (domain.MissionEngineStatus, error)
	Build            func(context.Context, InvocationBuildInput) (domain.MissionInvocationRequest, error)
	Complete         func(context.Context, InvocationCompleteInput) (domain.MissionInvocationOutcome, error)
	ExecuteHost      func(context.Context, string, string, string, domain.MissionInvocationRequest) (domain.MissionInvocationCompletion, error)
	WriteResult      func(*cobra.Command, bool, any) error
	ReadCompletion   func(*cobra.Command) (domain.MissionInvocationCompletion, error)
	// TelemetrySink selects the discovery event sink when a completion runs.
	// Production composition builds it from the existing telemetry
	// configuration; tests inject capture or failing sinks. A nil function
	// leaves the completion without a sink, which the runtime refuses.
	TelemetrySink func() telemetry.EventSink
}

// InvocationBuildInput identifies the immutable Weapon request to build.
type InvocationBuildInput struct {
	Root           string
	BasePath       string
	MissionID      string
	Role           string
	Slot           string
	RequestContext string
}

// InvocationCompleteInput identifies and supplies one host completion.
type InvocationCompleteInput struct {
	Root       string
	BasePath   string
	RequestID  string
	Completion domain.MissionInvocationCompletion
	// Adapter is the execution path the calling CLI command itself owns:
	// `mission complete` is the current-host return channel, `mission invoke
	// --host` is a child. It is never read from the completion.
	Adapter domain.MissionExecutionAdapter
	// Sink receives the provider-owned discovery invocation event. It is
	// required: completion has no sinkless normalization path.
	Sink telemetry.EventSink
}

// sink resolves the configured telemetry sink for one completion.
func (d InvocationDependencies) sink() telemetry.EventSink {
	if d.TelemetrySink == nil {
		return nil
	}
	return d.TelemetrySink()
}

type invocationFlags struct {
	root, missionID, role, slot, requestID, host, requestContext string
	asJSON                                                       bool
}

// NewInvoke builds `mission invoke`, which either emits one immutable JSON
// request or executes it through an explicitly selected host bridge.
func NewInvoke(deps InvocationDependencies) *cobra.Command {
	cmd := &cobra.Command{Use: "invoke", Short: "Invoke one embedded Weapon through a host bridge"}
	f := bindInvocationFlags(cmd, deps)
	cmd.Flags().StringVar(&f.role, "role", "ranger", "Role to invoke")
	cmd.Flags().StringVar(&f.slot, "slot", "", "pipeline slot to invoke (required)")
	cmd.Flags().StringVar(&f.host, "host", "", "execute through host (codex or claude); omit to emit the request")
	cmd.Flags().StringVar(&f.requestContext, "context", "", "original user request supplied to an explicit host bridge")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunInvoke(cmd, deps, f)
	}
	return cmd
}

// RunInvoke validates the request and emits only the bridge envelope.
func RunInvoke(cmd *cobra.Command, deps InvocationDependencies, f *invocationFlags) error {
	if deps.Build == nil || deps.WriteResult == nil {
		return fmt.Errorf("mission invoke: invocation runtime is unavailable")
	}
	root, basePath, err := resolveInvocationRuntime(deps, f)
	if err != nil {
		return err
	}
	request, err := buildInvocationRequest(cmd, deps, f, root, basePath)
	if err != nil {
		return err
	}
	return dispatchInvocation(cmd, deps, f, root, basePath, request)
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
	if deps.LoadMission == nil {
		return "", "", fmt.Errorf("mission invoke: mission lifecycle is unavailable")
	}
	status, err := deps.LoadMission(root, f.missionID)
	if err != nil {
		return "", "", fmt.Errorf("mission invoke: %w", err)
	}
	if err := validateInvocationPhase(status, f.slot); err != nil {
		return "", "", fmt.Errorf("mission invoke: %w", err)
	}
	return root, basePath, nil
}

func buildInvocationRequest(cmd *cobra.Command, deps InvocationDependencies, f *invocationFlags, root, basePath string) (domain.MissionInvocationRequest, error) {
	request, err := deps.Build(cmd.Context(), InvocationBuildInput{
		Root: root, BasePath: basePath, MissionID: f.missionID, Role: f.role, Slot: f.slot, RequestContext: f.requestContext,
	})
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

func validateInvocationPhase(status domain.MissionEngineStatus, slot string) error {
	expected, ok := map[string]domain.PipelinePhase{
		string(domain.SlotDiscovery):  domain.PhaseDiscovery,
		string(domain.SlotRefinement): domain.PhaseRefinement,
	}[slot]
	if !ok {
		return fmt.Errorf("slot %q has no host invocation boundary", slot)
	}
	if status.Phase != expected {
		return fmt.Errorf("slot %q requires phase %q, got %q", slot, expected, status.Phase)
	}
	return nil
}

func bindInvocationFlags(cmd *cobra.Command, deps InvocationDependencies) *invocationFlags {
	f := &invocationFlags{}
	cmd.Flags().StringVar(&f.root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().StringVar(&f.missionID, "mission-id", "", "mission identifier (required)")
	cmd.Flags().BoolVar(&f.asJSON, "json", false, "emit machine-readable JSON")
	return f
}
