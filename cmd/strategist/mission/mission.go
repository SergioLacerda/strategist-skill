// Package mission contains Cobra adapters for Strategist mission operations.
// State transitions remain owned by internal/domain.
package mission

import "github.com/spf13/cobra"

// Composition contains the complete set of runtime ports required to build
// the mission command family. It keeps executable wiring explicit while the
// package remains responsible only for Cobra adapters.
type Composition struct {
	Lifecycle  LifecycleDependencies
	View       ViewDependencies
	Normalize  NormalizeDependencies
	Usage      ReportUsageDependencies
	Invocation InvocationDependencies
}

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
func New(composition Composition) *cobra.Command {
	cmd := NewParent()
	cmd.AddCommand(
		NewStart(composition.Lifecycle),
		NewStatus(composition.Lifecycle),
		NewSubmit(composition.Lifecycle),
		NewRoute(composition.Lifecycle),
		NewContext(composition.Lifecycle),
		NewView(composition.View),
		NewNormalizeOpenSpec(composition.Normalize),
		NewReportUsage(composition.Usage),
		NewInvoke(composition.Invocation),
		NewComplete(composition.Invocation),
		NewRequests(composition.Invocation),
		NewAcceptSideQuest(composition.Lifecycle),
		NewDeclineSideQuest(composition.Lifecycle),
		NewADRTarget(composition.Lifecycle),
	)
	return cmd
}

// Register attaches Mission at the root composition boundary.
func Register(root *cobra.Command, composition Composition) {
	root.AddCommand(New(composition))
}
