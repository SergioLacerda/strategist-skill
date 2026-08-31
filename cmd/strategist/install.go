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
	installTarget        string
	installSilent        bool
	installWizard        bool
	installGlobal        bool
	installForce         bool
	installStrictCompile bool
	installNoShim        bool
	installShimPath      string
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Strategist skill into a target repository",
	Long: `Install the Strategist skill into a target repository.

Silent mode (default, no flags needed) writes epic-profile defaults and never
prompts. A CompileAll failure after extraction is warning-only by default —
install still completes with a partial/uncompiled runtime. Pass
--strict-compile to make that failure fatal (the install rolls back instead).

By default a Claude Code shim is written to
~/.claude/skills/strategist/SKILL.md so the skill is invocable outside this
repository. Use --no-shim to skip that write entirely (e.g. CI/containers
without a writable home directory), or --shim-path to redirect it.`,
	RunE: runInstall,
}

func runInstall(cmd *cobra.Command, _ []string) (retErr error) {
	if installNoShim && installShimPath != "" {
		return fmt.Errorf("install: --no-shim and --shim-path are mutually exclusive")
	}

	target, err := resolveInstallTarget(installTarget, installGlobal)
	if err != nil {
		return err
	}
	installTarget = target

	ctx := commandContext(cmd)
	markInstallRun(ctx, installWizard)
	ctx, span := startInstallSpan(ctx)
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	addMissionLines(ctx, 1)
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

	cfg := installConfigFromFlags()
	svc := installService(shimHome)

	if err := svc.Install(ctx, cfg); err != nil {
		return fmt.Errorf("install: %w", err)
	}

	addMissionLines(ctx, 2)
	// A successful Install with StrictCompile=false may still have hit a
	// non-fatal CompileAll failure (warning-only). Detect that here by checking
	// for the manifest CompileAll writes on success, so the terminal banner
	// clearly reports partial status instead of a bare "install complete".
	partial := installIsPartial(installTarget)
	slog.InfoContext(ctx, "[Strategist] install complete",
		telemetry.AttrComponent, "install",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
		telemetry.AttrTarget, installTarget,
		"partial", partial,
	)
	printInstallCompleteBanner(installTarget, installWizard, partial)
	return nil
}

func commandContext(cmd *cobra.Command) context.Context {
	ctx := cmd.Context()
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func markInstallRun(ctx context.Context, wizard bool) {
	run := telemetry.MissionRunFromContext(ctx)
	if run == nil {
		return
	}
	run.MarkRanger()
	if wizard {
		run.SetSilent()
	}
}

func startInstallSpan(ctx context.Context) (context.Context, trace.Span) {
	return telemetry.Tracer().Start(ctx, "strategist.install",
		trace.WithAttributes(
			attribute.String(telemetry.AttrComponent, "install"),
			attribute.String(telemetry.AttrRuntimeMode, "cli"),
			attribute.String(telemetry.AttrOutputProfile, "default"),
			attribute.String(telemetry.AttrTarget, installTarget),
		),
	)
}

func addMissionLines(ctx context.Context, lines int64) {
	if run := telemetry.MissionRunFromContext(ctx); run != nil {
		run.AddLines(lines)
	}
}

func installConfigFromFlags() domain.InstallConfig {
	return domain.InstallConfig{
		Target:        installTarget,
		Silent:        installSilent,
		Wizard:        installWizard,
		Global:        installGlobal,
		Force:         installForce,
		StrictCompile: installStrictCompile,
		NoShim:        installNoShim,
		ShimPath:      installShimPath,
	}
}

func installService(shimHome string) install.Service {
	return install.Service{
		Extractor:          embedpkg.Extractor{},
		Lister:             embedpkg.Extractor{},
		Compiler:           compile.Compiler{},
		ShimHomeDir:        shimHome,
		AwarenessRefresher: refreshAgentAwarenessFromEmbed,
		Version:            Version,
	}
}

func refreshAgentAwarenessFromEmbed(strategistRoot, projectRoot, version string) bool {
	tplBytes, err := embedpkg.Extractor{}.ReadFile("templates/agent-protocol.md")
	if err != nil {
		tplBytes = nil
	}
	return compile.RefreshAgentAwareness(strategistRoot, projectRoot, version, tplBytes)
}

func init() {
	installCmd.Flags().StringVar(&installTarget, "target", "", "target repository root (default: current directory)")
	installCmd.Flags().BoolVar(&installSilent, "silent", false, "silent install with epic defaults (default)")
	installCmd.Flags().BoolVar(&installWizard, "wizard", false, "interactive wizard for configuration")
	installCmd.Flags().BoolVar(&installGlobal, "global", false, "install into global root (default: local project)")
	installCmd.Flags().BoolVar(&installForce, "force", false, "overwrite all files, including user-modified ones (default: preserve customizations)")
	installCmd.Flags().BoolVar(&installStrictCompile, "strict-compile", false, "fail install (and roll back) on a CompileAll error, instead of warning-only (default: warning-only)")
	installCmd.Flags().BoolVar(&installNoShim, "no-shim", false, "skip writing the SKILL.md shim under ~/.claude/skills (mutually exclusive with --shim-path)")
	installCmd.Flags().StringVar(&installShimPath, "shim-path", "", "write the SKILL.md shim to this path instead of the default ~/.claude/skills/strategist/SKILL.md (mutually exclusive with --no-shim)")
}
