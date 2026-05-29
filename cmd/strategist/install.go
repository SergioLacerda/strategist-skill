package main

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/spf13/cobra"
)

var (
	installTarget string
	installSilent bool
	installWizard bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Strategist skill into a target repository",
	RunE: func(_ *cobra.Command, _ []string) error {
		if installTarget == "" {
			installTarget = "."
		}

		svc := install.Service{
			Extractor: embedpkg.Extractor{},
			Compiler:  compile.Compiler{},
		}

		cfg := domain.InstallConfig{
			Target: installTarget,
			Silent: installSilent,
			Wizard: installWizard,
		}

		if err := svc.Install(context.Background(), cfg); err != nil {
			return fmt.Errorf("install: %w", err)
		}

		fmt.Println("[Strategist] install complete →", installTarget)
		return nil
	},
}

func init() {
	installCmd.Flags().StringVar(&installTarget, "target", "", "target repository root (default: current directory)")
	installCmd.Flags().BoolVar(&installSilent, "silent", false, "silent install with pragmatic defaults (default)")
	installCmd.Flags().BoolVar(&installWizard, "wizard", false, "interactive wizard for configuration")
}
