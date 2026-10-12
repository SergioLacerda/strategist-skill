package onboarding

import (
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
)

func enrichRoleProviderBinding(binding *domain.SlotBinding, provider domain.ProviderContract) {
	runtime, connectorID := plugincatalog.BindingRuntimeIdentityForContract(provider)
	binding.WeaponVersion = provider.Version
	binding.Origin = providerOriginForContract(provider)
	binding.RuntimeKind = runtime.Kind
	binding.ConnectorID = connectorID
	binding.Entrypoint = roleProviderEntrypoint(binding.Slot)
}

func persistRoleProviderBindingMetadata(store *lifecycle.Store, entry RoleProviderPreviewEntry) {
	for index := range store.Bindings {
		if store.Bindings[index].Slot != entry.Slot {
			continue
		}
		enrichRoleProviderBinding(&store.Bindings[index], entry.Resolved.Provider)
		store.Bindings[index].Mode = domain.SlotBindingModeCustom
		return
	}
}

func persistRoleProviderInstanceMetadata(store *lifecycle.Store, entry RoleProviderPreviewEntry) {
	runtime, connectorID := plugincatalog.BindingRuntimeIdentityForContract(entry.Resolved.Provider)
	for index := range store.Inventory.Instances {
		instance := &store.Inventory.Instances[index]
		if instance.ID != entry.Resolved.Provider.ID {
			continue
		}
		instance.ConnectorID = connectorID
		instance.ProviderOrigin = providerOriginForContract(entry.Resolved.Provider)
		instance.Entrypoint = roleProviderEntrypoint(entry.Slot)
		instance.VerificationEvidence = roleProviderStructuralEvidence(runtime)
		return
	}
}

func providerOriginForContract(provider domain.ProviderContract) string {
	if provider.Source == domain.ProviderSourceExternal {
		return string(domain.WeaponOriginCustom)
	}
	return string(domain.WeaponOriginEmbedded)
}

func roleProviderEntrypoint(slot string) string {
	if slot == string(domain.SlotDiscovery) {
		return "discover"
	}
	if slot == string(domain.SlotRefinement) {
		return "refine"
	}
	return "invoke"
}

func roleProviderStructuralEvidence(runtime domain.WeaponRuntime) string {
	if runtime.Kind == domain.RankedRuntimeHost {
		return "catalog_host_package_structural_validation"
	}
	return "catalog_provider_resolution"
}
