// Package install contains the Cobra adapter for Strategist installation.
// Installation behavior remains owned by internal/install.Service.
package install

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internalinstall "github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// Installer is the install service the command drives; production wires
// internal/install.Service, tests supply a fake.
type Installer interface {
	InstallWithReport(context.Context, domain.InstallConfig) (internalinstall.Report, error)
}

// Dependencies supplies host-specific concerns at the CLI composition edge.
type Dependencies struct {
	ResolveTarget  func(explicit string, global bool) (string, error)
	UserHomeDir    func() (string, error)
	ServiceFactory func(shimHome string) Installer
}

type options struct {
	Target, ShimPath                                     string
	Silent, Wizard, Global, Force, StrictCompile, NoShim bool
	AllowDowngrade                                       bool
}

// TestOptions is an explicit command snapshot for adapter tests and external
// CLI contract fixtures. Production callers should use New and Cobra flags.
type TestOptions struct {
	Target, ShimPath                                     string
	Silent, Wizard, Global, Force, StrictCompile, NoShim bool
	AllowDowngrade                                       bool
}

// RunForTest runs the install command from an explicit TestOptions snapshot
// instead of parsed Cobra flags.
func RunForTest(cmd *cobra.Command, deps Dependencies, opts TestOptions) error {
	// A direct conversion: TestOptions must keep exactly the fields of options.
	return Run(cmd, deps, options(opts))
}

// New creates an isolated install command with no package-level flag state.
func New(deps Dependencies) *cobra.Command {
	opts := options{}
	cmd := &cobra.Command{
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
	}
	flags := cmd.Flags()
	flags.StringVar(&opts.Target, "target", "", "target repository root (default: current directory)")
	flags.BoolVar(&opts.Silent, "silent", false, "silent install with epic defaults (default)")
	flags.BoolVar(&opts.Wizard, "wizard", false, "interactive wizard for configuration")
	flags.BoolVar(&opts.Global, "global", false, "install into global root (default: local project)")
	flags.BoolVar(&opts.Force, "force", false, "overwrite all files, including user-modified ones (default: preserve customizations)")
	flags.BoolVar(&opts.AllowDowngrade, "allow-downgrade", false, "let this binary replace normative runtime files installed by a newer binary (deliberate rollback; default: refuse with runtime_newer_than_binary)")
	flags.BoolVar(&opts.StrictCompile, "strict-compile", false, "fail install (and roll back) on a CompileAll error, instead of warning-only (default: warning-only)")
	flags.BoolVar(&opts.NoShim, "no-shim", false, "skip writing the SKILL.md shim under ~/.claude/skills (mutually exclusive with --shim-path)")
	flags.StringVar(&opts.ShimPath, "shim-path", "", "write the SKILL.md shim to this path instead of the default ~/.claude/skills/strategist/SKILL.md (mutually exclusive with --no-shim)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return Run(cmd, deps, opts) }
	return cmd
}

// Register attaches the install command at the supplied root.
func Register(root *cobra.Command, deps Dependencies) {
	root.AddCommand(New(deps))
}

// Run executes an installation from an explicit immutable command snapshot.
func Run(cmd *cobra.Command, deps Dependencies, opts options) (retErr error) {
	if opts.NoShim && opts.ShimPath != "" {
		return fmt.Errorf("install: --no-shim and --shim-path are mutually exclusive")
	}
	if deps.ResolveTarget == nil {
		return fmt.Errorf("install: target resolver is not configured")
	}
	if deps.UserHomeDir == nil {
		return fmt.Errorf("install: home directory resolver is not configured")
	}
	target, err := deps.ResolveTarget(opts.Target, opts.Global)
	if err != nil {
		return err
	}
	ctx := commandContext(cmd)
	markInstallRun(ctx, opts.Wizard)
	ctx, span := startInstallSpan(ctx, target)
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	addMissionLines(ctx, 1)
	slog.InfoContext(ctx, "[Strategist] install running", telemetry.AttrComponent, "install", telemetry.AttrRuntimeMode, "cli", telemetry.AttrOutputProfile, "default", telemetry.AttrTarget, target)
	return execute(ctx, cmd, deps, target, opts)
}

func execute(ctx context.Context, cmd *cobra.Command, deps Dependencies, target string, opts options) error {
	shimHome, err := deps.UserHomeDir()
	if err != nil {
		return fmt.Errorf("install: resolve home dir: %w", err)
	}
	if deps.ServiceFactory == nil {
		return fmt.Errorf("install: service factory is not configured")
	}
	report, err := deps.ServiceFactory(shimHome).InstallWithReport(ctx, domain.InstallConfig{Target: target, Silent: opts.Silent, Wizard: opts.Wizard, Global: opts.Global, Force: opts.Force, AllowDowngrade: opts.AllowDowngrade, StrictCompile: opts.StrictCompile, NoShim: opts.NoShim, ShimPath: opts.ShimPath})
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	if report.BackupDir != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "\nBacked up overwritten files to %s\n", report.BackupDir); err != nil {
			return fmt.Errorf("install: write output: %w", err)
		}
	}
	addMissionLines(ctx, 2)
	partial := isPartial(target)
	slog.InfoContext(ctx, "[Strategist] install complete", telemetry.AttrComponent, "install", telemetry.AttrRuntimeMode, "cli", telemetry.AttrOutputProfile, "default", telemetry.AttrTarget, target, "partial", partial)
	printCompleteBanner(target, opts.Wizard, partial)
	return nil
}

func commandContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}
func markInstallRun(ctx context.Context, wizard bool) {
	if run := telemetry.MissionRunFromContext(ctx); run != nil {
		run.MarkRanger()
		if wizard {
			run.SetSilent()
		}
	}
}
func addMissionLines(ctx context.Context, lines int64) {
	if run := telemetry.MissionRunFromContext(ctx); run != nil {
		run.AddLines(lines)
	}
}
func startInstallSpan(ctx context.Context, target string) (context.Context, trace.Span) {
	return telemetry.Tracer().Start(ctx, "strategist.install", trace.WithAttributes(attribute.String(telemetry.AttrComponent, "install"), attribute.String(telemetry.AttrRuntimeMode, "cli"), attribute.String(telemetry.AttrOutputProfile, "default"), attribute.String(telemetry.AttrTarget, target)))
}
func isPartial(target string) bool {
	_, err := os.Stat(filepath.Join(target, ".strategist", ".compiled", ".manifest.gz"))
	return err != nil
}
