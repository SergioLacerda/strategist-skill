// Package main is the entry point for the strategist CLI.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/authorization"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "strategist",
	Short: "Strategist skill CLI",
	Long:  "Strategist — install, compile, and manage the Strategist skill for Claude agents.",
}

// requireStrategistDir returns an error if .strategist/active.yaml is absent in
// the current directory. Used by subcommands that depend on an installed workspace.
func requireStrategistDir() error {
	if _, err := os.Stat(".strategist/active.yaml"); os.IsNotExist(err) {
		return fmt.Errorf("[Strategist] error=not_installed\n→ Run: strategist install")
	}
	return nil
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(exitCodeFor(err))
	}
}

// exitCodeFor maps known error categories to distinct exit codes so that
// CI/CD pipelines can distinguish policy violations from generic failures.
//
//	0 — success
//	1 — generic / unknown error
//	2 — governance / policy violation
//	3 — stale artifact or config integrity error
func exitCodeFor(err error) int {
	switch {
	case errors.Is(err, authorization.ErrDenied), errors.Is(err, authorization.ErrBlocked):
		return 2
	case errors.Is(err, authorization.ErrStale):
		return 3
	case errors.Is(err, domain.ErrPipelineBypassDetected):
		return 2
	case errors.Is(err, domain.ErrSourceStale), errors.Is(err, domain.ErrArtifactAbsent), errors.Is(err, domain.ErrManifestMissing):
		return 3
	default:
		return 1
	}
}

func init() {
	installRootHooks(rootCmd)
	registerCommands(rootCmd)
}
