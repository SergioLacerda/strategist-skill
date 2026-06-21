package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"golang.org/x/term"
)

// Service implements domain.Installer.
type Service struct {
	Extractor domain.FileExtractor
	Compiler  domain.Compiler
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
	// ProgressFn is called after each major install step with (done, total).
	// Nil means no progress reporting. Set by the CLI to drive the Spell Charge bar.
	ProgressFn func(done, total int)
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
	var manifest []string // tracks created paths for rollback

	slog.InfoContext(ctx, "[Strategist] install starting",
		telemetry.AttrComponent, "install",
		telemetry.AttrTarget, telemetry.SanitizePath(cfg.Target),
	)

	succeeded := false
	defer func() {
		if !succeeded {
			rollbackManifest(ctx, manifest)
		}
	}()

	slog.InfoContext(ctx, "[Strategist] install extracting-defaults",
		telemetry.AttrComponent, "install",
		telemetry.AttrTarget, telemetry.SanitizePath(strategistDir),
	)
	if err := s.Extractor.Extract(strategistDir, cfg.Force); err != nil {
		return fmt.Errorf("install: extract defaults: %w", err)
	}
	manifest = append(manifest, strategistDir)
	s.reportProgress(1, 4)

	slog.InfoContext(ctx, "[Strategist] install applying-config",
		telemetry.AttrComponent, "install",
		"wizard", cfg.Wizard,
	)
	if err := s.applyConfig(strategistDir, cfg); err != nil {
		return err
	}
	manifest = append(manifest, filepath.Join(strategistDir, "active.yaml"))
	s.reportProgress(2, 4)

	if !cfg.Global {
		gitignorePath := filepath.Join(cfg.Target, ".gitignore")
		gitignoreExisted := fileExists(gitignorePath)
		if err := ensureGitignore(cfg.Target); err != nil {
			return fmt.Errorf("install: gitignore: %w", err)
		}
		if !gitignoreExisted {
			manifest = append(manifest, gitignorePath)
		}
	}

	slog.InfoContext(ctx, "[Strategist] install shim-step",
		telemetry.AttrComponent, "install",
		telemetry.AttrTarget, telemetry.SanitizePath(cfg.Target),
	)
	shimPath, err := s.installShimStep(ctx, cfg.Target)
	if err != nil {
		return err
	}
	manifest = append(manifest, shimPath)
	manifest = append(manifest, filepath.Dir(shimPath)) // shim dir — removed only if empty
	s.reportProgress(3, 4)

	// Compile after install; non-fatal — warn but do not abort.
	kiPath := filepath.Join(strategistDir, "knowledge.index.yaml")
	if compileErr := s.Compiler.CompileAll(strategistDir, kiPath); compileErr != nil {
		slog.WarnContext(ctx, "[Strategist] compile warning",
			"error", compileErr,
			telemetry.AttrComponent, "install",
			telemetry.AttrRuntimeMode, "cli",
			telemetry.AttrOutputProfile, "default",
			telemetry.AttrTarget, strategistDir,
		)
	}

	succeeded = true
	s.reportProgress(4, 4)
	slog.InfoContext(ctx, "[Strategist] install complete",
		telemetry.AttrComponent, "install",
		telemetry.AttrTarget, telemetry.SanitizePath(cfg.Target),
	)
	return nil
}

func (s Service) reportProgress(done, total int) {
	if s.ProgressFn != nil {
		s.ProgressFn(done, total)
	}
}

// applyConfig writes active.yaml either from the epic template (silent) or
// from wizard input.
//
// In silent mode, active.yaml is only written when it does not already exist
// (first install) or when cfg.Force is true. This preserves any customizations
// the user has made after the initial install.
func (s Service) applyConfig(strategistDir string, cfg domain.InstallConfig) error {
	if !cfg.Wizard {
		activeYAMLPath := filepath.Join(strategistDir, "active.yaml")
		if !cfg.Force && fileExists(activeYAMLPath) {
			return nil // preserve user customizations
		}
		data, err := s.Extractor.ReadFile("templates/epic-standalone.yaml")
		if err != nil {
			return fmt.Errorf("install: read template: %w", err)
		}
		if err := os.WriteFile(activeYAMLPath, data, 0o644); err != nil {
			return fmt.Errorf("install: write active.yaml: %w", err)
		}
		return nil
	}

	p := s.resolvePrompter()
	wc, err := runWizard(p)
	if err != nil {
		return fmt.Errorf("install: wizard: %w", err)
	}
	if err := writeActiveYAML(strategistDir, wc); err != nil {
		return fmt.Errorf("install: write active.yaml: %w", err)
	}
	if err := s.writeSelectedProviderManifests(strategistDir, wc); err != nil {
		return fmt.Errorf("install: write provider manifests: %w", err)
	}
	if err := writeKnowledgeIndexSource(strategistDir, wc); err != nil {
		return fmt.Errorf("install: write knowledge.index.yaml: %w", err)
	}
	return nil
}

func (s Service) writeSelectedProviderManifests(strategistDir string, wc domain.WizardConfig) error {
	slog.Info("[Strategist] install writing-manifests",
		telemetry.AttrComponent, "install",
		"discovery_provider", wc.DiscoveryProvider,
		"refinement_provider", wc.RefinementProvider,
	)
	selectedProviders := []string{wc.DiscoveryProvider, wc.RefinementProvider}

	for _, provider := range selectedProviders {
		manifestPath, ok := installableDefaultProviders[provider]
		if !ok {
			continue
		}

		data, err := s.Extractor.ReadFile(manifestPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", manifestPath, err)
		}

		providerDir := filepath.Join(strategistDir, provider)
		if err := os.MkdirAll(providerDir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", providerDir, err)
		}

		targetPath := filepath.Join(providerDir, "skill.yaml")
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", targetPath, err)
		}
	}

	return nil
}

// resolvePrompter returns the Prompter to use for wizard mode.
// Precedence: WizardPrompter (explicit) → TUIPrompter (TTY) → TextPrompter (non-TTY).
func (s Service) resolvePrompter() Prompter {
	if s.WizardPrompter != nil {
		return s.WizardPrompter
	}
	isTTY := s.terminalDetector
	if isTTY == nil {
		isTTY = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
	}
	if isTTY() {
		newTUI := s.tuiPrompterFn
		if newTUI == nil {
			newTUI = NewTUIPrompter
		}
		return newTUI()
	}
	stdin := s.stdinReader
	if stdin == nil {
		stdin = os.Stdin
	}
	return NewTextPrompter(stdin)
}

// installShimStep reads target/.strategist/SKILL.md and writes the full shim.
// Returns the shim path for rollback tracking.
func (s Service) installShimStep(ctx context.Context, target string) (string, error) {
	skillContent, err := s.readLocalSKILLMD(ctx, target)
	if err != nil {
		return "", fmt.Errorf("install: read SKILL.md: %w", err)
	}
	shimPath, err := s.resolveShimPath()
	if err != nil {
		return "", fmt.Errorf("install: resolve shim path: %w", err)
	}
	if err := s.installShimFor(target, skillContent); err != nil {
		return "", fmt.Errorf("install: shim: %w", err)
	}
	return shimPath, nil
}

// readLocalSKILLMD reads SKILL.md from the embedded FS.
// .strategist/ is write-only — we never read back from it.
func (s Service) readLocalSKILLMD(ctx context.Context, _ string) (string, error) {
	data, err := s.Extractor.ReadFile("SKILL.md")
	if err != nil {
		return "", fmt.Errorf("read embedded SKILL.md: %w", err)
	}
	slog.InfoContext(ctx, "[Strategist] SKILL.md read from embedded FS",
		telemetry.AttrComponent, "install",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
	)
	return string(data), nil
}

// installShimFor installs the shim, using ShimHomeDir if set (for tests).
func (s Service) installShimFor(target, skillContent string) error {
	if s.ShimHomeDir != "" {
		skillRoot := ""
		if target != "" {
			absTarget, err := filepath.Abs(target)
			if err != nil {
				return fmt.Errorf("resolve target: %w", err)
			}
			skillRoot = filepath.Join(absTarget, ".strategist")
		}
		return installShimTo(s.ShimHomeDir, skillContent, skillRoot)
	}
	return installShim(target)
}

// resolveShimPath returns the path of the SKILL.md shim that will be installed,
// without actually installing it. Used to track the shim in the rollback manifest.
func (s Service) resolveShimPath() (string, error) {
	homeDir := s.ShimHomeDir
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home dir: %w", err)
		}
	}
	return filepath.Join(homeDir, ".claude", "skills", "strategist", "SKILL.md"), nil
}

// rollbackManifest removes created paths in reverse order (best-effort).
// Non-empty directories are silently skipped — Remove only removes empty dirs.
func rollbackManifest(ctx context.Context, manifest []string) {
	for i := len(manifest) - 1; i >= 0; i-- {
		if err := os.Remove(manifest[i]); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.WarnContext(ctx, "[Strategist] rollback remove failed",
				"path", manifest[i],
				"error", err,
				telemetry.AttrComponent, "install",
				telemetry.AttrRuntimeMode, "cli",
				telemetry.AttrOutputProfile, "default",
				telemetry.AttrTarget, manifest[i],
			)
		}
	}
	slog.WarnContext(ctx, "[Strategist] install rolled back",
		"workspace", "restored",
		telemetry.AttrComponent, "install",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
	)
}

// fileExists reports whether path exists (any type).
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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

// NewInstaller returns a domain.Installer backed by Service.
func NewInstaller(extractor domain.FileExtractor, compiler domain.Compiler) domain.Installer {
	return &serviceAdapter{svc: Service{Extractor: extractor, Compiler: compiler}}
}
