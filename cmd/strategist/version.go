package main

import (
	"fmt"
	"log/slog"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags "-X main.Version=x.y.z".
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the strategist version",
	Run: func(cmd *cobra.Command, _ []string) {
		slog.InfoContext(cmd.Context(), "[Strategist] version",
			telemetry.AttrComponent, "version",
			telemetry.AttrRuntimeMode, "cli",
			telemetry.AttrOutputProfile, "default",
			"strategist.version", Version,
		)
		fmt.Println("strategist", Version)
	},
}
