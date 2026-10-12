package plugincatalog

import (
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
)

// BindingRuntimeIdentity returns the runtime and connector id recorded for a catalog provider binding.
func BindingRuntimeIdentity(provider Provider) (domain.WeaponRuntime, string) {
	if provider.CompatibilitySource == names.CompatibilityNativeRole {
		return domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModeCode}, names.ConnectorNativeRole
	}
	return BindingRuntimeIdentityForContract(domain.ProviderContract{
		Runtime: provider.Runtime,
		Source:  providerSourceFromCompatibilitySource(provider.CompatibilitySource),
	})
}

// BindingRuntimeIdentityForContract returns the runtime and connector id recorded for a provider contract binding.
func BindingRuntimeIdentityForContract(provider domain.ProviderContract) (domain.WeaponRuntime, string) {
	runtime := domain.NormalizeRankedRuntime(provider.Runtime)
	if provider.Source == domain.ProviderSourceNativeRole {
		return domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModeCode}, names.ConnectorNativeRole
	}
	if runtime.Kind == domain.RankedRuntimeEmbedded && runtime.ExecutionMode == "" {
		runtime.ExecutionMode = domain.WeaponExecutionModePromptBridge
	}
	connectorID := names.ConnectorCurrentRuntime
	switch runtime.Kind {
	case domain.RankedRuntimeEmbedded:
		connectorID = names.ConnectorEmbedded
	case domain.RankedRuntimeOpenSpecRoot:
		connectorID = names.ConnectorRankedRuntime
	case domain.RankedRuntimeHost:
		connectorID = names.ConnectorHost
	}
	return runtime, connectorID
}

// BindingOrigin returns the Weapon origin recorded for a catalog provider: custom for external sources, embedded otherwise.
func BindingOrigin(provider Provider) string {
	if provider.CompatibilitySource == names.CompatibilityExternal {
		return string(domain.WeaponOriginCustom)
	}
	return string(domain.WeaponOriginEmbedded)
}
