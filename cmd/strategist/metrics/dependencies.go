// Package metrics contains Cobra adapters for Strategist runtime-memory metrics.
// It delegates history and aggregation behavior to internal/telemetry.
package metrics

import "github.com/spf13/cobra"

// Dependencies defines injected dependencies for metrics commands.
type Dependencies struct {
	RootFlag    string
	ResolveRoot func(cmd *cobra.Command, action, explicitRoot string) (string, error)
	SilenceRun  func(cmd *cobra.Command)
	// ResolveBasePath returns the workspace artifact root (active.yaml's
	// base_path) for a resolved .strategist/ root. It returns "" with a nil
	// error when the workspace declares none. Optional: a nil resolver
	// disables the claim-location guard in `metrics record`.
	ResolveBasePath func(strategistRoot string) (string, error)
}
