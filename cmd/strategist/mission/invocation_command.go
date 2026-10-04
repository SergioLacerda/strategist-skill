package mission

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/application"
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

// InvocationBuildInput remains a CLI-package alias for existing adapters and
// tests; the application package owns the request contract.
type InvocationBuildInput = application.InvocationBuildRequest

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

func bindInvocationFlags(cmd *cobra.Command, deps InvocationDependencies) *invocationFlags {
	f := &invocationFlags{}
	cmd.Flags().StringVar(&f.root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().StringVar(&f.missionID, "mission-id", "", "mission identifier (required)")
	cmd.Flags().BoolVar(&f.asJSON, "json", false, "emit machine-readable JSON")
	return f
}
