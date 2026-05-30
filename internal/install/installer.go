package install

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Service implements domain.Installer.
type Service struct {
	Extractor domain.FileExtractor
	Compiler  domain.Compiler
	// WizardReader overrides os.Stdin for wizard prompts. Nil means use os.Stdin.
	// Set this in tests to provide scripted input without blocking on stdin.
	WizardReader io.Reader
	// ShimHomeDir overrides os.UserHomeDir() for shim installation. Nil means use real home.
	// Set this in tests to install the shim in a temporary directory.
	ShimHomeDir string
}

// Install installs the skill into cfg.Target. In silent mode it extracts defaults
// and writes active.yaml from the pragmatic template. In wizard mode it prompts
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
		if succeeded {
			return
		}
		// Rollback in reverse order: files first, then directories.
		for i := len(manifest) - 1; i >= 0; i-- {
			p := manifest[i]
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				// For non-empty directories Remove returns an error — that's intentional:
				// we only remove what we created, leaving pre-existing content intact.
				_ = err
			}
		}
		slog.WarnContext(ctx, "[Strategist] install rolled back", "workspace", "restored")
	}()

	if err := s.Extractor.Extract(strategistDir, cfg.Force); err != nil {
		return fmt.Errorf("install: extract defaults: %w", err)
	}
	manifest = append(manifest, strategistDir)

	if err := s.applyConfig(strategistDir, cfg); err != nil {
		return err
	}
	manifest = append(manifest, filepath.Join(strategistDir, "active.yaml"))

	gitignorePath := filepath.Join(cfg.Target, ".gitignore")
	gitignoreExisted := fileExists(gitignorePath)
	if err := ensureGitignore(cfg.Target); err != nil {
		return fmt.Errorf("install: gitignore: %w", err)
	}
	if !gitignoreExisted {
		manifest = append(manifest, gitignorePath)
	}

	shimPath, err := s.resolveShimPath()
	if err != nil {
		return fmt.Errorf("install: resolve shim path: %w", err)
	}
	if err := s.installShimFor(cfg.Target); err != nil {
		return fmt.Errorf("install: shim: %w", err)
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

// applyConfig writes active.yaml either from the pragmatic template (silent) or
// from wizard input.
func (s Service) applyConfig(strategistDir string, cfg domain.InstallConfig) error {
	if !cfg.Wizard {
		if err := copyTemplate(strategistDir, "templates/pragmatic-standalone.yaml", "active.yaml"); err != nil {
			return fmt.Errorf("install: copy template: %w", err)
		}
		return nil
	}

	r := io.Reader(os.Stdin)
	if s.WizardReader != nil {
		r = s.WizardReader
	}
	wc, err := runWizard(r)
	if err != nil {
		return fmt.Errorf("install: wizard: %w", err)
	}
	if err := writeActiveYAML(strategistDir, wc); err != nil {
		return fmt.Errorf("install: write active.yaml: %w", err)
	}
	return nil
}

// installShimFor installs the shim, using ShimHomeDir if set (for tests).
func (s Service) installShimFor(target string) error {
	if s.ShimHomeDir != "" {
		return installShimTo(s.ShimHomeDir)
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
