package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

var (
	installTarget string
	installSilent bool
	installWizard bool
	installGlobal bool
	installForce  bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Strategist skill into a target repository",
	RunE:  runInstall,
}

func runInstall(cmd *cobra.Command, _ []string) (retErr error) {
	if installGlobal {
		if installTarget == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("install: resolve home dir: %w", err)
			}
			installTarget = home
		}
	} else if installTarget == "" {
		installTarget = "."
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	run := telemetry.MissionRunFromContext(ctx)
	if run != nil {
		run.MarkRanger()
	}
	ctx, span := telemetry.Tracer().Start(ctx, "strategist.install",
		trace.WithAttributes(
			attribute.String(telemetry.AttrComponent, "install"),
			attribute.String(telemetry.AttrRuntimeMode, "cli"),
			attribute.String(telemetry.AttrOutputProfile, "default"),
			attribute.String(telemetry.AttrTarget, installTarget),
		),
	)
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	if run != nil {
		run.AddLines(1)
	}
	slog.InfoContext(ctx, "[Strategist] install running",
		telemetry.AttrComponent, "install",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
		telemetry.AttrTarget, installTarget,
	)

	shimHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("install: resolve home dir: %w", err)
	}

	svc := install.Service{
		Extractor:   embedpkg.Extractor{},
		Compiler:    compile.Compiler{},
		ShimHomeDir: shimHome,
	}

	cfg := domain.InstallConfig{
		Target: installTarget,
		Silent: installSilent,
		Wizard: installWizard,
		Global: installGlobal,
		Force:  installForce,
	}

	if err := svc.Install(ctx, cfg); err != nil {
		return fmt.Errorf("install: %w", err)
	}

	if run != nil {
		run.AddLines(2)
	}
	slog.InfoContext(ctx, "[Strategist] install complete",
		telemetry.AttrComponent, "install",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
		telemetry.AttrTarget, installTarget,
	)
	fmt.Println("[Strategist] install complete →", installTarget)
	return nil
}

func init() {
	installCmd.Flags().StringVar(&installTarget, "target", "", "target repository root (default: current directory)")
	installCmd.Flags().BoolVar(&installSilent, "silent", false, "silent install with epic defaults (default)")
	installCmd.Flags().BoolVar(&installWizard, "wizard", false, "interactive wizard for configuration")
	installCmd.Flags().BoolVar(&installGlobal, "global", false, "install into global root (default: local project)")
	installCmd.Flags().BoolVar(&installForce, "force", false, "overwrite all files, including user-modified ones (default: preserve customizations)")
}
