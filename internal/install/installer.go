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

	succeeded := false
	defer func() {
		if !succeeded {
			rollbackManifest(ctx, manifest)
		}
	}()

	if err := s.Extractor.Extract(strategistDir, cfg.Force); err != nil {
		return fmt.Errorf("install: extract defaults: %w", err)
	}
	manifest = append(manifest, strategistDir)

	if err := s.applyConfig(strategistDir, cfg); err != nil {
		return err
	}
	manifest = append(manifest, filepath.Join(strategistDir, "active.yaml"))

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

	shimPath, err := s.installShimStep(ctx, cfg.Target)
	if err != nil {
		return err
	}
	manifest = append(manifest, shimPath)
	manifest = append(manifest, filepath.Dir(shimPath)) // shim dir — removed only if empty

	// Compile after install; non-fatal — warn but do not abort.
	kiPath := filepath.Join(strategistDir, "knowledge.index.yaml")
	if compileErr := s.Compiler.CompileAll(strategistDir, kiPath); compileErr != nil {
		slog.WarnContext(ctx, "[Strategist] compile warning", "error", compileErr)
	}

	succeeded = true
	return nil
}

// applyConfig writes active.yaml either from the epic template (silent) or
// from wizard input.
func (s Service) applyConfig(strategistDir string, cfg domain.InstallConfig) error {
	if !cfg.Wizard {
		if err := copyTemplate(strategistDir, "templates/epic-standalone.yaml", "active.yaml"); err != nil {
			return fmt.Errorf("install: copy template: %w", err)
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
	if err := writeKnowledgeIndexSource(strategistDir, wc); err != nil {
		return fmt.Errorf("install: write knowledge.index.yaml: %w", err)
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

// readLocalSKILLMD reads target/.strategist/SKILL.md extracted in this install run.
func (s Service) readLocalSKILLMD(ctx context.Context, target string) (string, error) {
	path := filepath.Join(target, ".strategist", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read local SKILL.md: %w", err)
	}
	slog.InfoContext(ctx, "[Strategist] SKILL.md read for shim", "path", path)
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
			slog.WarnContext(ctx, "[Strategist] rollback remove failed", "path", manifest[i], "error", err)
		}
	}
	slog.WarnContext(ctx, "[Strategist] install rolled back", "workspace", "restored")
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
