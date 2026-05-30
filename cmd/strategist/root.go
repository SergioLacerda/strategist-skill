// Package main is the entry point for the strategist CLI.
package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "strategist",
	Short: "Strategist skill CLI",
	Long:  "Strategist — install, compile, and manage the Strategist skill for Claude agents.",
}

func init() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		cmd.SetContext(context.Background())
		return nil
	}
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(installGlobalCmd)
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
