package install

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"golang.org/x/term"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

// applyConfig writes active.yaml either from the epic template (silent) or
// from wizard input.
//
// In silent mode, active.yaml is only written when it does not already exist
// (first install) or when cfg.Force is true. This preserves any customizations
// the user has made after the initial install.
func (s Service) applyConfig(strategistDir string, cfg domain.InstallConfig) error {
	if !cfg.Wizard {
		return s.applySilentConfig(strategistDir, cfg)
	}
	return s.applyWizardConfig(strategistDir)
}

func (s Service) applySilentConfig(strategistDir string, cfg domain.InstallConfig) error {
	activeYAMLPath := filepath.Join(strategistDir, activeYAMLName)
	if !cfg.Force && runtimefs.Exists(activeYAMLPath) {
		return nil // preserve user customizations
	}
	if cfg.Force && runtimefs.Exists(activeYAMLPath) {
		slog.Info("[Strategist] install force-overwriting user-owned config",
			telemetry.AttrComponent, "install",
			"path", activeYAMLPath,
		)
	}
	data, err := s.Extractor.ReadFile(epicStandaloneTemplatePath)
	if err != nil {
		return fmt.Errorf("install: read template: %w", err)
	}
	// Route through writeActiveYAMLBytes so silent installs seal .config.lock the
	// same way wizard installs do — a silent install must never start unlocked.
	if err := writeActiveYAMLBytes(strategistDir, data); err != nil {
		return fmt.Errorf("install: %w", err)
	}
	return nil
}

func (s Service) applyWizardConfig(strategistDir string) error {
	p := s.resolvePrompter()
	wc, err := runWizard(p, s.Extractor)
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
	if err := writeTreasureChestManifest(strategistDir, wc); err != nil {
		return fmt.Errorf("install: write treasure-chests.yaml: %w", err)
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
		if err := s.writeSelectedProviderManifest(strategistDir, provider); err != nil {
			return err
		}
	}

	return nil
}

func (s Service) writeSelectedProviderManifest(strategistDir, provider string) error {
	manifestPath, ok := installableDefaultProviders[provider]
	if !ok {
		return nil
	}
	data, err := s.Extractor.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", manifestPath, err)
	}
	providerDir := filepath.Join(strategistDir, installedProvidersDirName, provider)
	targetPath := filepath.Join(providerDir, skillYAMLName)
	if err := atomicWriteFile(targetPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", targetPath, err)
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
