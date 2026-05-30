package main

import (
	"context"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/spf13/cobra"
)

// installGlobalTarget is the base directory for the global install (defaults to $HOME).
var installGlobalTarget string

var installGlobalCmd = &cobra.Command{
	Use:   "install-global",
	Short: "Install the Strategist skill globally into ~/.strategist/",
	Long: `Installs the Strategist skill defaults into ~/.strategist/ so the agent shim
(~/.claude/skills/strategist/SKILL.md) can resolve the skill root globally,
outside of any specific project directory.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		target := installGlobalTarget
		if target == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("install-global: resolve home dir: %w", err)
			}
			target = home
			installGlobalTarget = target
		}

		svc := install.Service{
			Extractor: embedpkg.Extractor{},
			Compiler:  compile.Compiler{},
		}

		cfg := domain.InstallConfig{
			Target: target,
			Silent: true,
		}

		if err := svc.Install(context.Background(), cfg); err != nil {
			return fmt.Errorf("install-global: %w", err)
		}

		fmt.Printf("[Strategist] global install complete — skill root: %s/.strategist/\n", target)
		return nil
	},
}
