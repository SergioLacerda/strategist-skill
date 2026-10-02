package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// NewComplete builds `mission complete`, which consumes one raw host response
// from stdin and delegates normalization to the production runtime.
func NewComplete(deps InvocationDependencies) *cobra.Command {
	cmd := &cobra.Command{Use: "complete", Short: "Complete one embedded Weapon invocation"}
	f := bindInvocationFlags(cmd, deps)
	cmd.Flags().StringVar(&f.requestID, "request-id", "", "pending invocation request identifier (required)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunComplete(cmd, deps, f)
	}
	return cmd
}

// RunComplete reads exactly one host completion and returns the Ranger-owned
// outcome. The host never writes the canonical artifact directly.
func RunComplete(cmd *cobra.Command, deps InvocationDependencies, f *invocationFlags) error {
	if deps.Complete == nil || deps.WriteResult == nil || deps.ReadCompletion == nil {
		return fmt.Errorf("mission complete: invocation runtime is unavailable")
	}
	if f.requestID == "" {
		return fmt.Errorf("mission complete: --request-id is required")
	}
	root, basePath, err := deps.ResolveBasePath(f.root)
	if err != nil {
		return fmt.Errorf("mission complete: %w", err)
	}
	completion, err := deps.ReadCompletion(cmd)
	if err != nil {
		return fmt.Errorf("mission complete: %w", err)
	}
	outcome, err := deps.Complete(cmd.Context(), InvocationCompleteInput{Root: root, BasePath: basePath, RequestID: f.requestID, Completion: completion, Adapter: domain.ExecutionAdapterCurrentHost, Sink: deps.sink()})
	if err != nil {
		return fmt.Errorf("mission complete: %w", err)
	}
	return deps.WriteResult(cmd, f.asJSON, outcome)
}
