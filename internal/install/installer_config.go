package install

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/term"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

// applyConfig writes active.yaml either from the epic template (silent) or
// from wizard input.
//
// In silent mode, active.yaml is only written when it does not already exist
// (first install) or when cfg.Force is true. This preserves any customizations
// the user has made after the initial install.
func (s Service) applyConfig(ctx context.Context, strategistDir string, cfg domain.InstallConfig) (retErr error) {
	ctx, span := telemetry.Tracer().Start(ctx, "install.apply_config")
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	if !cfg.Wizard {
		return s.applySilentConfig(ctx, strategistDir, cfg)
	}
	return s.applyWizardConfig(ctx, strategistDir)
}

func (s Service) applySilentConfig(_ context.Context, strategistDir string, cfg domain.InstallConfig) error {
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
	data, err := s.readNormalizedSilentConfig()
	if err != nil {
		return err
	}
	// Route through writeActiveYAMLBytes so silent installs seal .config.lock the
	// same way wizard installs do — a silent install must never start unlocked.
	if err := writeActiveYAMLBytes(strategistDir, data); err != nil {
		return fmt.Errorf("install: %w", err)
	}
	return s.activateSilentBindings(strategistDir, data)
}

func (s Service) readNormalizedSilentConfig() ([]byte, error) {
	data, err := s.Extractor.ReadFile(epicStandaloneTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("install: read template: %w", err)
	}
	data, err = normalizeStandaloneLanguage(data)
	if err != nil {
		return nil, fmt.Errorf("install: normalize language: %w", err)
	}
	return data, nil
}

func (s Service) activateSilentBindings(strategistDir string, data []byte) error {
	// A silent install's template slots (discovery/refinement) are external
	// skill providers, not native roles — strategist check now fails closed
	// when discovery/refinement lack a persisted plugins.lock binding
	// (rolevalidation.ValidateRuntimeBindings), so a silent install activates
	// the same Role/Provider binding the wizard path activates via
	// activateRoleProviderMigration, or a fresh `strategist install --silent`
	// would never pass `strategist check`. This is best-effort: a fixture or
	// workspace extractor that doesn't model plugins/catalog.yaml (unlike the
	// real embedded defaults, which always do) must not block install itself
	// — strategist check remains the fail-closed gate for role readiness.
	if err := s.activateSilentRoleProviderBindings(strategistDir, data); err != nil {
		slog.Warn("[Strategist] install role/provider binding activation skipped",
			telemetry.AttrComponent, "install",
			"reason", err.Error(),
		)
	}
	return nil
}

func (s Service) applyWizardConfig(ctx context.Context, strategistDir string) error {
	p := s.resolvePrompter()
	wc, err := runWizard(ctx, p, s.Extractor, strategistDir)
	if err != nil {
		return fmt.Errorf("install: wizard: %w", err)
	}
	if err := writeActiveYAML(strategistDir, wc); err != nil {
		return fmt.Errorf("install: write active.yaml: %w", err)
	}
	return s.persistWizardConfig(strategistDir, wc)
}

func (s Service) persistWizardConfig(strategistDir string, wc domain.WizardConfig) error {
	if len(wc.ResolvedPluginLock.Bindings) > 0 {
		if err := writePluginLockFile(strategistDir, wc.ResolvedPluginLock); err != nil {
			return fmt.Errorf("install: write plugins.lock: %w", err)
		}
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
	if err := persistGovernanceState(strategistDir, wc.GovernancePolicy, wc.PermissionGrants); err != nil {
		return fmt.Errorf("install: write governance state: %w", err)
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
	installable, err := resolveInstallableDefaultProviders(s.Extractor)
	if err != nil {
		return fmt.Errorf("resolve installable providers for %s: %w", provider, err)
	}
	if _, ok := installable[provider]; !ok {
		return nil
	}
	data, err := providerManifestBytes(s.Extractor, provider)
	if err != nil {
		return err
	}
	providerDir := filepath.Join(strategistDir, installedProvidersDirName, provider)
	targetPath := filepath.Join(providerDir, skillYAMLName)
	if err := atomicWriteFile(targetPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", targetPath, err)
	}
	return nil
}

func providerManifestBytes(extractor domain.FileExtractor, provider string) ([]byte, error) {
	catalog, err := loadPluginCatalog(extractor)
	if err != nil {
		return nil, fmt.Errorf("load plugin catalog for %s: %w", provider, err)
	}
	data, err := generateLegacyProviderManifest(catalog, provider)
	if err != nil {
		return nil, err
	}
	return data, nil
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
