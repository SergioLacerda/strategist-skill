package install

import "github.com/SergioLacerda/strategist-skill/internal/domain"

func bindingRuntimeIdentity(provider pluginCatalogProvider) (domain.WeaponRuntime, string) {
	if provider.CompatibilitySource == "native_role" {
		return domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModeCode}, "strategist-native-role"
	}
	return bindingRuntimeIdentityForContract(domain.ProviderContract{
		Runtime: provider.Runtime,
		Source:  providerSourceFromCompatibilitySource(provider.CompatibilitySource),
	})
}

func bindingRuntimeIdentityForContract(provider domain.ProviderContract) (domain.WeaponRuntime, string) {
	runtime := domain.NormalizeRankedRuntime(provider.Runtime)
	if provider.Source == domain.ProviderSourceNativeRole {
		return domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModeCode}, "strategist-native-role"
	}
	if runtime.Kind == domain.RankedRuntimeEmbedded && runtime.ExecutionMode == "" {
		runtime.ExecutionMode = domain.WeaponExecutionModePromptBridge
	}
	connectorID := "current-runtime"
	switch runtime.Kind {
	case domain.RankedRuntimeEmbedded:
		connectorID = "strategist-embedded"
	case domain.RankedRuntimeOpenSpecRoot:
		connectorID = "strategist-ranked-runtime"
	case domain.RankedRuntimeHost:
		connectorID = "strategist-host"
	}
	return runtime, connectorID
}

func bindingOrigin(provider pluginCatalogProvider) string {
	if provider.CompatibilitySource == "external" {
		return string(domain.WeaponOriginCustom)
	}
	return string(domain.WeaponOriginEmbedded)
}
