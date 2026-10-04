package weapon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// ResolveWeaponFacts returns the manifest of provider under a .strategist
// root. The catalog entry is the authority; the adapter.yaml of a package
// bound with mode custom comes next. An unreadable catalog is an error: a
// broken authority never silently falls back.
func ResolveWeaponFacts(strategistRoot, provider string) (domain.WeaponFacts, error) {
	facts, found, err := resolveCatalogFacts(strategistRoot, provider)
	if err != nil || found {
		return facts, err
	}
	if facts, found, err = resolveAdapterFacts(strategistRoot, provider); err != nil || found {
		return facts, err
	}
	return domain.WeaponFacts{}, domain.ErrWeaponFactsNotFound
}

// ResolveCatalogWeaponFacts returns only the catalog entry. It never falls
// back to adapter.yaml; slot resolution uses it to distinguish catalogued
// Weapons from custom packages.
func ResolveCatalogWeaponFacts(strategistRoot, provider string) (domain.WeaponFacts, bool, error) {
	return resolveCatalogFacts(strategistRoot, provider)
}

// ListCatalogWeaponFacts returns every catalog entry in catalog order. An
// absent catalog yields none.
func ListCatalogWeaponFacts(strategistRoot string) ([]domain.WeaponFacts, error) {
	docs, err := readCatalogDocs(strategistRoot)
	if err != nil {
		return nil, err
	}
	out := make([]domain.WeaponFacts, 0, len(docs))
	for _, doc := range docs {
		out = append(out, doc.WeaponFacts(domain.WeaponFactsSourceCatalog))
	}
	return out, nil
}

func resolveCatalogFacts(strategistRoot, provider string) (domain.WeaponFacts, bool, error) {
	docs, err := readCatalogDocs(strategistRoot)
	if err != nil {
		return domain.WeaponFacts{}, false, err
	}
	matches := domain.CatalogDocumentsForReference(docs, provider)
	switch len(matches) {
	case 0:
		return domain.WeaponFacts{}, false, nil
	case 1:
		return matches[0].WeaponFacts(domain.WeaponFactsSourceCatalog), true, nil
	default:
		return domain.WeaponFacts{}, false, fmt.Errorf("ambiguous weapon facts for %q: %w", provider, domain.AmbiguousWeaponFactsError(provider, matches))
	}
}

func readCatalogDocs(strategistRoot string) ([]domain.WeaponFactsDocument, error) {
	raw, err := os.ReadFile(filepath.Join(strategistRoot, "plugins", "catalog.yaml")) //nolint:gosec // fixed path under the runtime root
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read plugin catalog: %w", err)
	}
	var file struct {
		Providers []domain.WeaponFactsDocument `yaml:"providers"`
	}
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse plugin catalog: %w", err)
	}
	return file.Providers, nil
}

// ResolveCustomPackageFacts returns the facts of a staged package bound with
// mode custom, looked up by installed instance id. It never consults the
// catalog.
func ResolveCustomPackageFacts(strategistRoot, instance string) (domain.WeaponFacts, bool, error) {
	return resolveAdapterFacts(strategistRoot, instance)
}

func resolveAdapterFacts(strategistRoot, instance string) (domain.WeaponFacts, bool, error) {
	lockedInstance, err := customInstance(strategistRoot, instance)
	if err != nil || lockedInstance == "" {
		return domain.WeaponFacts{}, false, err
	}
	raw, err := os.ReadFile(filepath.Join(strategistRoot, "providers", lockedInstance, "adapter.yaml")) //nolint:gosec // path derives from the locked instance id
	if errors.Is(err, os.ErrNotExist) {
		return domain.WeaponFacts{}, false, nil
	}
	if err != nil {
		return domain.WeaponFacts{}, false, fmt.Errorf("read adapter for %s: %w", lockedInstance, err)
	}
	var adapter domain.AdapterContract
	if err := yaml.Unmarshal(raw, &adapter); err != nil {
		return domain.WeaponFacts{}, false, fmt.Errorf("parse adapter for %s: %w", lockedInstance, err)
	}
	return domain.WeaponFactsFromAdapter(instance, adapter), true, nil
}

func customInstance(strategistRoot, provider string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(strategistRoot, "plugins.lock")) //nolint:gosec // fixed path under the runtime root
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read plugins.lock: %w", err)
	}
	var lock domain.PluginLockFile
	if err := yaml.Unmarshal(raw, &lock); err != nil {
		return "", fmt.Errorf("parse plugins.lock: %w", err)
	}
	return customInstanceFromLock(lock, provider), nil
}

func customInstanceFromLock(lock domain.PluginLockFile, provider string) string {
	for _, binding := range lock.Bindings {
		id := binding.InstalledInstanceID
		if binding.EffectiveMode() == domain.SlotBindingModeCustom && (id == provider || strings.HasPrefix(id, provider+"@")) {
			return id
		}
	}
	return ""
}
