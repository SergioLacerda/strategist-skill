// Package main is the entry point for the strategist CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "strategist",
	Short: "Strategist skill CLI",
	Long:  "Strategist — install, compile, and manage the Strategist skill for Claude agents.",
}

func init() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		run := telemetry.NewMissionRun(fmt.Sprintf("%s-%d", cmd.Name(), time.Now().UnixNano()))
		ctx = telemetry.WithMissionRun(ctx, run)
		run.MarkIntake()
		run.AddLines(1)
		slogLine := run.StartLine("local", ".strategist", ".strategist/active.yaml", "unknown", "local_default", "default")
		fmt.Println(slogLine)
		cmd.SetContext(ctx)
		return nil
	}
	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, _ []string) error {
		telemetry.FinishMission(cmd.Context())
		return nil
	}
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(compileCmd)
	rootCmd.AddCommand(checkStaleCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(syncGovernanceCmd)
	rootCmd.AddCommand(versionCmd)
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
