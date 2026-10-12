package plugincatalog

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
)

// ValidateCatalogWeaponCompositions reports an error when a composite provider references a missing or incompatible component.
func ValidateCatalogWeaponCompositions(catalog PluginCatalog) error {
	providers := make(map[string]Provider, len(catalog.Providers))
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

func validateCatalogParent(parent Provider, providers map[string]Provider) error {
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

func validateCatalogComponent(parent, child Provider) error {
	if err := validateComponentRoles(parent, child); err != nil {
		return err
	}
	if err := validateComponentSlots(parent, child); err != nil {
		return err
	}
	if child.CompatibilitySource == names.CompatibilityEmbedded {
		if err := child.Runtime.ValidateActive(); err != nil {
			return fmt.Errorf("plugin catalog: component Weapon %s is not invocable: %w", child.ID, err)
		}
	}
	return nil
}

func validateComponentRoles(parent, child Provider) error {
	for _, role := range ProviderRoles(parent) {
		if !ProviderHasRole(child, role) {
			return fmt.Errorf("plugin catalog: component Weapon %s is incompatible with Role %s of %s", child.ID, role, parent.ID)
		}
	}
	return nil
}

func validateComponentSlots(parent, child Provider) error {
	for _, slot := range parent.SupportedSlots {
		if !ContainsCatalogString(child.SupportedSlots, slot) {
			return fmt.Errorf("plugin catalog: component Weapon %s is incompatible with slot %s of %s", child.ID, slot, parent.ID)
		}
	}
	return nil
}

// ContainsCatalogString reports whether values contains target.
func ContainsCatalogString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// ValidateCatalogProvider reports an error when a catalog provider entry is incomplete or inconsistent.
func ValidateCatalogProvider(provider Provider) error {
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
		return ValidateEmbeddedCatalogProvider(provider)
	}
	return nil
}

// declaresWeaponManifest reports whether the entry is an embedded or Custom
// Weapon whose full manifest must validate.
func declaresWeaponManifest(provider Provider) bool {
	return provider.CompatibilitySource == names.CompatibilityEmbedded || provider.Origin == domain.WeaponOriginCustom
}

// ValidateEmbeddedCatalogProvider reports an error when an embedded catalog provider lacks the fields embedded entries require.
func ValidateEmbeddedCatalogProvider(provider Provider) error {
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
	if err := CatalogProviderManifest(provider).ValidateActive(); err != nil {
		return fmt.Errorf("plugin catalog: Weapon %s: %w", provider.ID, err)
	}
	return nil
}

// CatalogProviderManifest projects a catalog provider onto the Weapon manifest it declares.
func CatalogProviderManifest(provider Provider) domain.WeaponManifest {
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
