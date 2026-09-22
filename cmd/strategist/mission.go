package main

import "github.com/spf13/cobra"

// missionCmd is the parent for Strategist mission-level commands that the
// invoking agent (not this binary) drives directly, as opposed to
// "metrics" (read-only, computed from runtime memory) or "handoff" (a
// scoped verification tool for one transition).
var missionCmd = &cobra.Command{
	Use:   "mission",
	Short: "Report and inspect mission-level facts this binary cannot observe directly",
}

// registerMission attaches Mission commands at the root composition boundary.
// Its handlers remain thin adapters over the existing internal authorities.
func registerMission(root *cobra.Command) {
	registerMissionSubcommands(missionCmd)
	root.AddCommand(missionCmd)
}

func registerMissionSubcommands(parent *cobra.Command) {
	parent.AddCommand(
		missionStartCmd,
		missionStatusCmd,
		missionSubmitCmd,
		missionContextCmd,
		missionViewCmd,
		missionNormalizeOpenSpecCmd,
		missionReportUsageCmd,
	)
}

// newMissionCommand creates an isolated Mission command tree for tests and
// embedding. It is a CLI adapter boundary; domain transitions, persistence,
// usage records and refinement behavior remain in their existing authorities.
func newMissionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mission",
		Short: "Report and inspect mission-level facts this binary cannot observe directly",
	}
	cmd.AddCommand(
		newMissionStartCommand(),
		newMissionStatusCommand(),
		newMissionSubmitCommand(),
		newMissionContextCommand(),
		newMissionViewCommand(),
		newMissionNormalizeOpenSpecCommand(),
		newMissionReportUsageCommand(),
	)
	return cmd
}
