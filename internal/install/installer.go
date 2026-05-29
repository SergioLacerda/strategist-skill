package install

import (
	"context"
	"fmt"
	"io"
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
// The context is threaded through for future cancellation support.
func (s Service) Install(_ context.Context, cfg domain.InstallConfig) error {
	strategistDir := filepath.Join(cfg.Target, ".strategist")

	if err := s.Extractor.Extract(strategistDir); err != nil {
		return fmt.Errorf("install: extract defaults: %w", err)
	}

	if err := s.applyConfig(strategistDir, cfg); err != nil {
		return err
	}

	if err := ensureGitignore(cfg.Target); err != nil {
		return fmt.Errorf("install: gitignore: %w", err)
	}

	if err := s.installShimFor(cfg.Target); err != nil {
		return fmt.Errorf("install: shim: %w", err)
	}

	// Compile after install; non-fatal — warn but do not abort.
	kiPath := filepath.Join(strategistDir, "knowledge.index.yaml")
	if compileErr := s.Compiler.CompileAll(strategistDir, kiPath); compileErr != nil {
		fmt.Fprintf(os.Stderr, "[Strategist] WARN: compile failed: %v\n", compileErr)
	}

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
