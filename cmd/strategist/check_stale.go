package main

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/spf13/cobra"
)

var checkStaleCmd = &cobra.Command{
	Use:   "check-stale <artifact.gz>",
	Short: "Check if a compiled artifact is stale (exit 0=fresh, exit 1=stale)",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		isStale, err := stale.Checker{}.IsStale(args[0])
		if err != nil {
			return fmt.Errorf("check-stale: %w", err)
		}
		if isStale {
			os.Exit(1)
		}
		return nil
	},
}
