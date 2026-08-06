package main

import (
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// Thin wrappers over internal/cliutil so every existing unqualified call site in
// this package (eval*.go, metrics*.go, handoff_verify.go, ...) keeps working
// unchanged. internal/treasurecli has its own equivalent wrappers (see its
// cli_bridge.go) pointing at the same internal/cliutil source of truth.

const (
	flagFormat            = cliutil.FlagFormat
	flagRoot              = cliutil.FlagRoot
	flagIndex             = cliutil.FlagIndex
	flagIncludeHistorical = cliutil.FlagIncludeHistorical
)

func stringFlag(cmd *cobra.Command, name, fallback string) string {
	return cliutil.StringFlag(cmd, name, fallback)
}

func boolFlag(cmd *cobra.Command, name string, fallback bool) bool {
	return cliutil.BoolFlag(cmd, name, fallback)
}

func telemetryRunFromCmd(cmd *cobra.Command) *telemetry.MissionRun {
	return cliutil.TelemetryRunFromCmd(cmd)
}
