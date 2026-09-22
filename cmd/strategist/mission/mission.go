// Package mission contains Cobra adapters for Strategist mission operations.
// State transitions remain owned by internal/domain.
package mission

import "github.com/spf13/cobra"

// NewParent builds the bare `mission` parent command.
func NewParent() *cobra.Command {
	return &cobra.Command{
		Use:   "mission",
		Short: "Report and inspect mission-level facts this binary cannot observe directly",
	}
}

// New composes the complete Mission command family. The CLI root supplies
// runtime-specific persistence and path dependencies; transition semantics
// remain owned by internal/domain.
func New(lifecycle LifecycleDependencies, view ViewDependencies, normalize NormalizeDependencies, usage ReportUsageDependencies) *cobra.Command {
	cmd := NewParent()
	cmd.AddCommand(
		NewStart(lifecycle),
		NewStatus(lifecycle),
		NewSubmit(lifecycle),
		NewContext(lifecycle),
		NewView(view),
		NewNormalizeOpenSpec(normalize),
		NewReportUsage(usage),
	)
	return cmd
}

// Register attaches Mission at the root composition boundary.
func Register(root *cobra.Command, lifecycle LifecycleDependencies, view ViewDependencies, normalize NormalizeDependencies, usage ReportUsageDependencies) {
	root.AddCommand(New(lifecycle, view, normalize, usage))
}
