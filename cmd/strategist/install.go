package main

import (
	"context"
	"fmt"
	"log/slog"

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
	installForce  bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Strategist skill into a target repository",
	RunE:  runInstall,
}

func runInstall(cmd *cobra.Command, _ []string) (retErr error) {
	if installTarget == "" {
		installTarget = "."
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := telemetry.Tracer().Start(ctx, "strategist.install",
		trace.WithAttributes(attribute.String(telemetry.AttrTarget, installTarget)),
	)
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	slog.InfoContext(ctx, "[Strategist] install running", "target", installTarget)

	svc := install.Service{
		Extractor: embedpkg.Extractor{},
		Compiler:  compile.Compiler{},
	}

	cfg := domain.InstallConfig{
		Target: installTarget,
		Silent: installSilent,
		Wizard: installWizard,
		Force:  installForce,
	}

	if err := svc.Install(context.Background(), cfg); err != nil {
		return fmt.Errorf("install: %w", err)
	}

	slog.InfoContext(ctx, "[Strategist] install complete", "target", installTarget)
	fmt.Println("[Strategist] install complete →", installTarget)
	return nil
}

func init() {
	installCmd.Flags().StringVar(&installTarget, "target", "", "target repository root (default: current directory)")
	installCmd.Flags().BoolVar(&installSilent, "silent", false, "silent install with pragmatic defaults (default)")
	installCmd.Flags().BoolVar(&installWizard, "wizard", false, "interactive wizard for configuration")
	installCmd.Flags().BoolVar(&installForce, "force", false, "overwrite all files, including user-modified ones (default: preserve customizations)")
}
