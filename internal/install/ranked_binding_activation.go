package install

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// rankedModeSlots maps each Wizard-collected mode field to its slot name.
func rankedModeSlots(wc domain.WizardConfig) map[string]string {
	return map[string]string{
		"discovery":  wc.DiscoveryMode,
		"refinement": wc.RefinementMode,
		"execution":  wc.ExecutionMode,
	}
}

// applyRankedBindingChoices overwrites, for every slot the Wizard resolved
// to Ranked, the Custom-mode SlotBinding activateRoleProviderMigration just
// wrote with the pre-generated Ranked record from the catalog's
// certification stamp (docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md
// DEC-005) — an activate/copy step, not fresh construction. Every slot the
// Wizard resolved to Custom (the overwhelming majority: every existing
// installation, since DiscoveryMode/RefinementMode default to "") is
// returned untouched.
func applyRankedBindingChoices(catalog pluginCatalog, wc domain.WizardConfig, lockFile domain.PluginLockFile) (domain.PluginLockFile, error) {
	slots := wizardSlots(wc)
	for slot, mode := range rankedModeSlots(wc) {
		if mode != domain.SlotBindingModeRanked {
			continue
		}
		binding, err := rankedBindingForSlot(catalog, slot, slots[slot])
		if err != nil {
			return domain.PluginLockFile{}, fmt.Errorf("apply ranked binding: slot %s: %w", slot, err)
		}
		lockFile.Bindings = replaceSlotBinding(lockFile.Bindings, domain.SlotBinding{
			SchemaVersion:       "strategist-plugin-binding/v1",
			TaxonomyVersion:     domain.CanonicalTaxonomyVersion,
			Slot:                slot,
			InstalledInstanceID: binding.WeaponID,
			Role:                binding.Role,
			WeaponVersion:       binding.WeaponVersion,
			WeaponDigest:        binding.WeaponDigest,
			SourceDigest:        binding.SourceDigest,
			BindingDigest:       binding.BindingDigest,
			ExecutionMode:       binding.ExecutionMode,
			Origin:              string(domain.WeaponOriginEmbedded),
			RuntimeKind:         binding.Runtime.Kind,
			ConnectorID:         binding.ConnectorID,
			Entrypoint:          binding.Entrypoint,
			CertificationDigest: binding.CertificationDigest,
			Generation:          binding.Generation,
			Status:              binding.Status,
			Mode:                domain.SlotBindingModeRanked,
		})
	}
	return lockFile, nil
}

// refreshPersistedRankedBindings replaces only bindings that were already
// selected as Ranked with their immutable catalog records. Reinstalling an
// existing runtime preserves active.yaml and therefore cannot rely on the
// wizard's mode fields to perform this refresh. Custom selections are user
// choices and are deliberately left untouched.
func refreshPersistedRankedBindings(catalog pluginCatalog, lockFile domain.PluginLockFile) (domain.PluginLockFile, error) {
	for _, existing := range lockFile.Bindings {
		if existing.EffectiveMode() != domain.SlotBindingModeRanked {
			continue
		}
		binding, err := rankedBindingForSlot(catalog, existing.Slot, existing.InstalledInstanceID)
		if err != nil {
			return domain.PluginLockFile{}, fmt.Errorf("refresh ranked binding: slot %s: %w", existing.Slot, err)
		}
		lockFile.Bindings = replaceSlotBinding(lockFile.Bindings, domain.SlotBinding{
			SchemaVersion:       "strategist-plugin-binding/v1",
			TaxonomyVersion:     domain.CanonicalTaxonomyVersion,
			Slot:                existing.Slot,
			InstalledInstanceID: binding.WeaponID,
			Role:                binding.Role,
			WeaponVersion:       binding.WeaponVersion,
			WeaponDigest:        binding.WeaponDigest,
			SourceDigest:        binding.SourceDigest,
			BindingDigest:       binding.BindingDigest,
			ExecutionMode:       binding.ExecutionMode,
			Origin:              string(domain.WeaponOriginEmbedded),
			RuntimeKind:         binding.Runtime.Kind,
			ConnectorID:         binding.ConnectorID,
			Entrypoint:          binding.Entrypoint,
			CertificationDigest: binding.CertificationDigest,
			Generation:          binding.Generation,
			Status:              binding.Status,
			Mode:                domain.SlotBindingModeRanked,
		})
	}
	return lockFile, nil
}

// replaceSlotBinding overwrites bindings' entry for replacement.Slot in
// place, or appends replacement if none exists yet.
func replaceSlotBinding(bindings []domain.SlotBinding, replacement domain.SlotBinding) []domain.SlotBinding {
	for i, b := range bindings {
		if b.Slot == replacement.Slot {
			bindings[i] = replacement
			return bindings
		}
	}
	return append(bindings, replacement)
}

// enrichLockBindingMetadata projects the catalog's immutable Weapon identity
// into the workspace lock for Custom bindings as well as legacy bindings that
// were created before the Role/Weapon fields existed. Ranked bindings are
// already copied from the compiled registry and are only completed when a
// legacy fixture omitted the newer fields.
func enrichLockBindingMetadata(catalog pluginCatalog, lockFile domain.PluginLockFile) (domain.PluginLockFile, error) {
	for i, binding := range lockFile.Bindings {
		enriched, ok := enrichBinding(catalog, lockFile, binding)
		if ok {
			lockFile.Bindings[i] = enriched
		}
	}
	return lockFile, nil
}

func enrichBinding(catalog pluginCatalog, lockFile domain.PluginLockFile, binding domain.SlotBinding) (domain.SlotBinding, bool) {
	if binding.EffectiveMode() == domain.SlotBindingModeRanked {
		if enriched, ok := enrichCompiledRankedBinding(catalog, binding); ok {
			return enriched, true
		}
	}
	provider, ok := catalogProviderForBinding(catalog, binding)
	if !ok {
		return binding, false
	}
	return applyCatalogBindingMetadata(lockFile, binding, provider), true
}

func catalogProviderForBinding(catalog pluginCatalog, binding domain.SlotBinding) (pluginCatalogProvider, bool) {
	providerID := binding.InstalledInstanceID
	if at := strings.IndexByte(providerID, '@'); at > 0 {
		providerID = providerID[:at]
	}
	return findCatalogProvider(catalog, providerID)
}

func applyCatalogBindingMetadata(lockFile domain.PluginLockFile, binding domain.SlotBinding, provider pluginCatalogProvider) domain.SlotBinding {
	role := binding.Role
	if role == "" {
		role = slotRoleID(domain.SlotName(binding.Slot))
	}
	runtime, connectorID := bindingRuntimeIdentity(provider)
	binding.Role = role
	if binding.WeaponVersion == "" {
		binding.WeaponVersion = providerVersionOrDefault(provider.Version)
	}
	binding.WeaponDigest = catalogProviderDigest(provider)
	binding.Origin = bindingOrigin(provider)
	binding.RuntimeKind = runtime.Kind
	binding.ConnectorID = connectorID
	binding.Entrypoint = compiledEntrypoint(provider.SupportedSlots, provider.ID)
	if binding.BindingDigest == "" {
		binding.BindingDigest = lockFile.NodeDigest(role+":"+provider.ID, "role_provider_binding")
	}
	return binding
}

func enrichCompiledRankedBinding(catalog pluginCatalog, binding domain.SlotBinding) (domain.SlotBinding, bool) {
	ref := domain.WeaponIdentity(binding.InstalledInstanceID, binding.WeaponVersion)
	if binding.WeaponVersion == "" {
		ref = binding.InstalledInstanceID
	}
	if compiled, found, err := selectCompiledRankedBinding(catalog.RankedBindings, binding.Role, binding.Slot, ref); err == nil && found {
		binding.WeaponVersion = compiled.WeaponVersion
		binding.WeaponDigest = compiled.WeaponDigest
		binding.SourceDigest = compiled.SourceDigest
		binding.BindingDigest = compiled.BindingDigest
		binding.ExecutionMode = compiled.ExecutionMode
		binding.Origin = string(domain.WeaponOriginEmbedded)
		binding.RuntimeKind = compiled.Runtime.Kind
		binding.ConnectorID = compiled.ConnectorID
		binding.Entrypoint = compiled.Entrypoint
		binding.CertificationDigest = compiled.CertificationDigest
		binding.Generation = compiled.Generation
		binding.Status = compiled.Status
		return binding, true
	}
	return binding, false
}
