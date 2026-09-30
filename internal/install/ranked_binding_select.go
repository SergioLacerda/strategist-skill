package install

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// selectCompiledRankedBinding picks the certified Ranked binding of a Weapon
// reference ("id" or "id@version") for role and slot. A plain id resolves only
// while exactly one version is certified: with several it is an error naming
// them, never a first match or "latest" (ADR-0061 Decisions 9-10).
func selectCompiledRankedBinding(bindings []domain.CompiledRankedBinding, role, slot, ref string) (domain.CompiledRankedBinding, bool, error) {
	id, version := domain.ParseWeaponRef(ref)
	var matches []domain.CompiledRankedBinding
	for _, binding := range bindings {
		if binding.Role == role && binding.Slot == slot && binding.WeaponID == id && (version == "" || binding.WeaponVersion == version) {
			matches = append(matches, binding)
		}
	}
	switch len(matches) {
	case 0:
		return domain.CompiledRankedBinding{}, false, nil
	case 1:
		return matches[0], true, nil
	}
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, domain.WeaponIdentity(match.WeaponID, match.WeaponVersion))
	}
	return domain.CompiledRankedBinding{}, false, fmt.Errorf("%s/%s has several certified versions of %q (%s); select one as %s@<version>", role, slot, id, strings.Join(names, ", "), id)
}

func rankedBindingForSlot(catalog pluginCatalog, slot, providerRef string) (domain.CompiledRankedBinding, error) {
	role := slotRoleID(domain.SlotName(slot))
	providerID, _ := domain.ParseWeaponRef(providerRef)
	if binding, found, err := selectCompiledRankedBinding(catalog.RankedBindings, role, slot, providerRef); err != nil || found {
		return binding, err
	}

	// Synthetic catalog fixtures predating the compiled registry do not carry
	// ranked_bindings. Preserve their focused lifecycle coverage while keeping
	// the real embedded catalog on the copied, immutable binding path above.
	contract, err := rankedContractForProvider(catalog, providerID)
	if err != nil {
		return domain.CompiledRankedBinding{}, err
	}
	provider, _ := findCatalogProvider(catalog, providerID)
	runtime := domain.NormalizeRankedRuntime(provider.Runtime)
	if runtime.Kind == domain.RankedRuntimeNone && provider.CompatibilitySource == "native_role" {
		runtime = domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded}
	}
	return domain.CompiledRankedBinding{
		Role: role, Slot: slot, WeaponID: contract.ID, WeaponVersion: providerVersionOrDefault(provider.Version), WeaponDigest: catalogProviderDigest(provider),
		RoleDigest: "legacy-role-digest", BindingDigest: contract.CertificationDigest,
		CertificationDigest: contract.CertificationDigest, ConnectorID: "strategist-embedded", Runtime: runtime,
		Entrypoint: compiledEntrypoint(provider.SupportedSlots, provider.ID), Generation: contract.RankedBindingGeneration, Status: contract.RankedBindingStatus,
	}, nil
}

func rankedContractForProvider(catalog pluginCatalog, providerID string) (domain.ProviderContract, error) {
	provider, ok := findCatalogProvider(catalog, providerID)
	if !ok {
		return domain.ProviderContract{}, fmt.Errorf("provider %q not found in catalog", providerID)
	}
	contract := providerContractFromCatalogEntry(provider)
	if !contract.Ranked || contract.CertificationDigest == "" {
		return domain.ProviderContract{}, fmt.Errorf("provider %q is not a certified ranked candidate", providerID)
	}
	return contract, nil
}
