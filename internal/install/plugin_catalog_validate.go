package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func validateCatalogWeaponCompositions(catalog pluginCatalog) error {
	providers := make(map[string]pluginCatalogProvider, len(catalog.Providers))
	for _, provider := range catalog.Providers {
		providers[provider.ID] = provider
	}
	for _, parent := range catalog.Providers {
		if parent.Kind != domain.WeaponKindComposite || parent.Composition == nil {
			continue
		}
		if err := validateCatalogParent(parent, providers); err != nil {
			return err
		}
	}
	return nil
}

func validateCatalogParent(parent pluginCatalogProvider, providers map[string]pluginCatalogProvider) error {
	for _, component := range parent.Composition.Components {
		child, ok := providers[component.ID]
		if !ok {
			return fmt.Errorf("plugin catalog: composite Weapon %s references missing component Weapon %s", parent.ID, component.ID)
		}
		if err := validateCatalogComponent(parent, child); err != nil {
			return err
		}
	}
	return nil
}

func validateCatalogComponent(parent, child pluginCatalogProvider) error {
	if err := validateComponentRoles(parent, child); err != nil {
		return err
	}
	if err := validateComponentSlots(parent, child); err != nil {
		return err
	}
	if child.CompatibilitySource == "embedded" {
		if err := child.Runtime.ValidateActive(); err != nil {
			return fmt.Errorf("plugin catalog: component Weapon %s is not invocable: %w", child.ID, err)
		}
	}
	return nil
}

func validateComponentRoles(parent, child pluginCatalogProvider) error {
	for _, role := range providerRoles(parent) {
		if !providerHasRole(child, role) {
			return fmt.Errorf("plugin catalog: component Weapon %s is incompatible with Role %s of %s", child.ID, role, parent.ID)
		}
	}
	return nil
}

func validateComponentSlots(parent, child pluginCatalogProvider) error {
	for _, slot := range parent.SupportedSlots {
		if !containsCatalogString(child.SupportedSlots, slot) {
			return fmt.Errorf("plugin catalog: component Weapon %s is incompatible with slot %s of %s", child.ID, slot, parent.ID)
		}
	}
	return nil
}

func containsCatalogString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func validateCatalogProvider(provider pluginCatalogProvider) error {
	if provider.ID == "" || provider.RiskScore == "" {
		return fmt.Errorf("plugin catalog: provider id and risk_score are required")
	}
	for _, roleID := range append([]string{provider.CanonicalRole}, provider.Roles...) {
		if err := domain.ValidateRoleReference(roleID); err != nil {
			return fmt.Errorf("plugin catalog: provider %s: %w", provider.ID, err)
		}
	}
	if err := provider.WeaponContract.Validate(); err != nil {
		return fmt.Errorf("plugin catalog: provider %s: %w", provider.ID, err)
	}
	if declaresWeaponManifest(provider) {
		return validateEmbeddedCatalogProvider(provider)
	}
	return nil
}

// declaresWeaponManifest reports whether the entry is an embedded or Custom
// Weapon whose full manifest must validate.
func declaresWeaponManifest(provider pluginCatalogProvider) bool {
	return provider.CompatibilitySource == "embedded" || provider.Origin == domain.WeaponOriginCustom
}

func validateEmbeddedCatalogProvider(provider pluginCatalogProvider) error {
	// Existing generated catalogs are accepted long enough for the
	// prepare-embedded migration to rewrite them from the authoritative
	// external manifests. Newly ingested entries are strict at the adapter
	// boundary and are always emitted with these fields.
	if provider.Kind == "" {
		return nil
	}
	if len(provider.SupportedSlots) == 0 {
		return fmt.Errorf("plugin catalog: Weapon %s must declare supported_slots", provider.ID)
	}
	if err := catalogProviderManifest(provider).ValidateActive(); err != nil {
		return fmt.Errorf("plugin catalog: Weapon %s: %w", provider.ID, err)
	}
	return nil
}

func catalogProviderManifest(provider pluginCatalogProvider) domain.WeaponManifest {
	manifest := domain.WeaponManifest{
		ID: provider.ID, Version: provider.Version, Kind: provider.Kind, Origin: provider.Origin, Roles: provider.Roles,
		SupportedSlots: provider.SupportedSlots, RiskScore: provider.RiskScore,
		Runtime: provider.Runtime, Contract: provider.WeaponContract, Composition: provider.Composition,
	}
	if manifest.Version == "" {
		manifest.Version = "legacy"
	}
	if provider.CanonicalRole != "" && len(manifest.Roles) == 0 {
		manifest.Roles = []string{provider.CanonicalRole}
	}
	return manifest
}
