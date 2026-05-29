package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags "-X main.Version=x.y.z".
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the strategist version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("strategist", Version)
	},
}
