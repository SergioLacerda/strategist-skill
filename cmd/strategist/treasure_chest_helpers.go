package main

import (
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// telemetryRunFromCmd extracts the MissionRun from the command context, if any.
func telemetryRunFromCmd(cmd *cobra.Command) *telemetry.MissionRun {
	if cmd == nil {
		return nil
	}
	ctx := cmd.Context()
	if ctx == nil {
		return nil
	}
	return telemetry.MissionRunFromContext(ctx)
}

func treasureChestRootFromCmd(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	root, err := cmd.Flags().GetString("root")
	if err == nil {
		return root
	}
	root, err = cmd.InheritedFlags().GetString("root")
	if err == nil {
		return root
	}
	return ""
}

func stringFlag(cmd *cobra.Command, name, fallback string) string {
	if cmd == nil {
		return fallback
	}
	value, err := cmd.Flags().GetString(name)
	if err == nil {
		return value
	}
	value, err = cmd.InheritedFlags().GetString(name)
	if err == nil {
		return value
	}
	return fallback
}

func boolFlag(cmd *cobra.Command, name string, fallback bool) bool {
	if cmd == nil {
		return fallback
	}
	value, err := cmd.Flags().GetBool(name)
	if err == nil {
		return value
	}
	value, err = cmd.InheritedFlags().GetBool(name)
	if err == nil {
		return value
	}
	return fallback
}
