package metrics

import "github.com/spf13/cobra"

// New creates the complete Metrics command family. Runtime-memory behavior
// remains delegated to internal/telemetry and internal/leveling; the CLI root
// provides only environment-specific dependencies.
func New(deps Dependencies, ledger string, defaultMax int) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Report metrics computed from Strategist's own runtime memory",
		Long:  "Report metrics computed from .strategist/memory/*.jsonl history. Each subcommand covers one metrics domain.",
	}
	cmd.AddCommand(
		NewHandoff(deps),
		NewHandoffRecord(deps),
		NewConfidence(deps),
		NewGateOutcome(deps),
		NewLabel(deps),
		NewLevels(deps, ledger, defaultMax),
		NewMissionQuality(deps),
		NewRecord(deps),
		NewRollout(),
		NewScout(deps),
	)
	return cmd
}

// Register attaches the complete Metrics family at the supplied root.
func Register(root *cobra.Command, deps Dependencies, ledger string, defaultMax int) {
	root.AddCommand(New(deps, ledger, defaultMax))
}
