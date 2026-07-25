package install

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Service implements domain.Installer.
type Service struct {
	Extractor domain.FileExtractor
	Compiler  domain.Compiler
	// AwarenessRefresher generates agent-protocol.md and updates per-agent entrypoint
	// files. Called after CompileAll on every install (wizard and silent). Nil means skip.
	// Returns true if agent-protocol.md was successfully written; false on partial failure.
	// All failures are non-blocking: the refresher logs warnings internally.
	AwarenessRefresher func(strategistRoot, projectRoot, version string) bool
	// Version is passed to AwarenessRefresher to stamp agent-protocol.md.
	Version string
	// WizardPrompter overrides the auto-detected Prompter for wizard prompts.
	// Nil means auto-detect: TUIPrompter when stdin is a TTY, TextPrompter otherwise.
	// Set this in tests to provide scripted input without blocking on stdin.
	WizardPrompter Prompter
	// ShimHomeDir overrides os.UserHomeDir() for shim installation. Nil means use real home.
	// Set this in tests to install the shim in a temporary directory.
	ShimHomeDir string
	// terminalDetector overrides TTY detection for tests. Nil means use term.IsTerminal.
	terminalDetector func() bool
	// stdinReader overrides os.Stdin for the TextPrompter fallback. Nil means use os.Stdin.
	// Set this in tests to avoid blocking on a real terminal.
	stdinReader io.Reader
	// tuiPrompterFn overrides NewTUIPrompter for tests. Nil means use NewTUIPrompter.
	// Set this in tests to avoid huh.Run blocking on a real or open-pipe stdin.
	tuiPrompterFn func() Prompter
}

// Install installs the skill into cfg.Target. In silent mode it extracts defaults
// and writes active.yaml from the epic template. In wizard mode it prompts
// the user for configuration before writing active.yaml.
//
// On failure, Install removes any files and directories it created, restoring the
// workspace to its pre-install state (best-effort: non-empty directories are skipped).
//
// The context is threaded through for future cancellation support.
func (s Service) Install(ctx context.Context, cfg domain.InstallConfig) error {
	strategistDir := filepath.Join(cfg.Target, ".strategist")
	strategistDirExisted := fileExists(strategistDir)
	var manifest []string // tracks created paths for rollback (existing-tree case only)

	slog.InfoContext(ctx, "[Strategist] install starting",
		telemetry.AttrComponent, "install",
		telemetry.AttrTarget, telemetry.SanitizePath(cfg.Target),
	)

	succeeded := false
	defer func() {
		if !succeeded {
			rollbackInstall(ctx, strategistDir, strategistDirExisted, manifest)
		}
	}()

	runtimePlan, err := s.planRuntimeDefaultUpgrade(strategistDir, cfg.Force)
	if err != nil {
		return err
	}

	if err := s.runInstallSteps(ctx, strategistDir, cfg, runtimePlan, &manifest); err != nil {
		return err
	}

	succeeded = true
	slog.InfoContext(ctx, "[Strategist] install complete",
		telemetry.AttrComponent, "install",
		telemetry.AttrTarget, telemetry.SanitizePath(cfg.Target),
	)
	return nil
}

func (s Service) runInstallSteps(ctx context.Context, strategistDir string, cfg domain.InstallConfig, runtimePlan runtimeDefaultPlan, manifest *[]string) error {
	created, err := s.prepareRuntime(ctx, strategistDir, cfg, runtimePlan)
	if err != nil {
		return err
	}
	*manifest = append(*manifest, created...)
	created, err = s.applyWorkspaceConfig(ctx, strategistDir, cfg)
	if err != nil {
		return err
	}
	*manifest = append(*manifest, created...)
	gitignoreManifest, err := ensureProjectGitignore(cfg)
	if err != nil {
		return err
	}
	*manifest = append(*manifest, gitignoreManifest...)
	shimManifest, err := s.runShimStep(ctx, cfg)
	if err != nil {
		return err
	}
	*manifest = append(*manifest, shimManifest...)
	if err := s.finalizeInstall(ctx, cfg, strategistDir, runtimePlan); err != nil {
		return err
	}
	*manifest = append(*manifest, filepath.Join(strategistDir, domain.InstallManifestRelPath))
	return nil
}

func (s Service) prepareRuntime(ctx context.Context, strategistDir string, cfg domain.InstallConfig, plan runtimeDefaultPlan) ([]string, error) {
	slog.InfoContext(ctx, "[Strategist] install extracting-defaults",
		telemetry.AttrComponent, "install",
		telemetry.AttrTarget, telemetry.SanitizePath(strategistDir),
	)
	if err := s.Extractor.Extract(strategistDir, cfg.Force); err != nil {
		return nil, fmt.Errorf("install: extract defaults: %w", err)
	}
	if err := s.applyRuntimeDefaultPlan(strategistDir, plan); err != nil {
		return nil, err
	}
	return []string{strategistDir}, nil
}

func (s Service) applyWorkspaceConfig(ctx context.Context, strategistDir string, cfg domain.InstallConfig) ([]string, error) {
	slog.InfoContext(ctx, "[Strategist] install applying-config",
		telemetry.AttrComponent, "install",
		"wizard", cfg.Wizard,
	)
	if err := s.applyConfig(strategistDir, cfg); err != nil {
		return nil, err
	}
	return []string{filepath.Join(strategistDir, "active.yaml")}, nil
}

func ensureProjectGitignore(cfg domain.InstallConfig) ([]string, error) {
	if cfg.Global {
		return nil, nil
	}
	gitignorePath := filepath.Join(cfg.Target, ".gitignore")
	gitignoreExisted := fileExists(gitignorePath)
	if err := ensureGitignore(cfg.Target); err != nil {
		return nil, fmt.Errorf("install: gitignore: %w", err)
	}
	if gitignoreExisted {
		return nil, nil
	}
	return []string{gitignorePath}, nil
}

func (s Service) finalizeInstall(ctx context.Context, cfg domain.InstallConfig, strategistDir string, plan runtimeDefaultPlan) error {
	if err := s.compileAfterInstall(ctx, cfg, strategistDir); err != nil {
		return err
	}
	if s.AwarenessRefresher != nil {
		s.AwarenessRefresher(strategistDir, cfg.Target, s.Version)
	}
	installManifest := domain.NewInstallManifest(packageID(s.Version), plan.embeddedHashes)
	if err := saveInstallManifest(strategistDir, installManifest); err != nil {
		return err
	}
	return nil
}

// Ensure Service satisfies the domain interface via the adapter method below.
// Note: domain.Installer has Install(cfg InstallConfig) error — we expose a
// context-aware variant and adapt via the wrapper.
var _ domain.Installer = &serviceAdapter{}

// serviceAdapter wraps Service to satisfy domain.Installer (context-free signature).
type serviceAdapter struct{ svc Service }

func (a *serviceAdapter) Install(cfg domain.InstallConfig) error {
	return a.svc.Install(context.Background(), cfg)
}

// NewInstaller returns a minimal domain.Installer adapter. It intentionally leaves
// AwarenessRefresher unwired — command-layer installs construct Service directly
// (with AwarenessRefresher set) when agent-protocol.md/entrypoint awareness refresh
// is required. Callers using this constructor get install without awareness refresh.
func NewInstaller(extractor domain.FileExtractor, compiler domain.Compiler) domain.Installer {
	return &serviceAdapter{svc: Service{Extractor: extractor, Compiler: compiler}}
}
