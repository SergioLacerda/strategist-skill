package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

const (
	outputFormatTable = "table"
	outputFormatJSON  = "json"

	flagFormat            = "format"
	flagRoot              = "root"
	flagIndex             = "index"
	flagIncludeHistorical = "include-historical"
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
	root, err := cmd.Flags().GetString(flagRoot)
	if err == nil {
		return root
	}
	root, err = cmd.InheritedFlags().GetString(flagRoot)
	if err == nil {
		return root
	}
	return ""
}

func resolveTreasureChestActionRoot(cmd *cobra.Command, action string) (string, error) {
	prefix := treasureChestActionPrefix(action)
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("%s: get cwd: %w", prefix, err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRootFromCmd(cmd), cwd)
	if err != nil {
		return "", fmt.Errorf("%s: %w", prefix, err)
	}
	return root, nil
}

func treasureChestActionPrefix(action string) string {
	if strings.HasPrefix(action, "treasure-chest") {
		return action
	}
	return "treasure-chest " + action
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
