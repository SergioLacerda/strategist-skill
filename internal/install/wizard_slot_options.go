package install

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// excludedProviderOption records why compatibleProviderOptions did not offer
// a given catalog candidate, so promptSlots can print it instead of letting
// the operator wonder whether an empty-looking option list is a bug or an
// intended exclusion (see .analysis/refined/
// 20260914-wizard-weapon-options-not-listed/design.md Task 3).
type excludedProviderOption struct {
	id      string
	reasons []domain.CompatibilityReason
}

// slotOptions is what the Wizard offers for one slot. ids are the Custom
// options and rankedRefs the certified-Ranked ones; each is a Weapon reference,
// spelled "id@version" exactly where a plain id would be ambiguous across
// versions and as the plain id otherwise (ADR-0061 Decision 9).
type slotOptions struct {
	ids           []string
	defaultID     string
	rankedRefs    []string
	rankedDefault string
	excluded      []excludedProviderOption
}

// compatibleSlotOptions lists the catalog candidates for roleName that
// role-affinity validation reports compatible. defaultID is the candidate the
// catalog marks default, else the first one. Every certified version is a Ranked
// option, and rankedDefault pre-selects the certified candidate marked default,
// else the highest certified version: a UI pre-selection the operator confirms,
// never a resolution. Native roles are not substitutes for the required
// discovery/refinement weapons. When nothing is compatible ids is empty and the
// caller must fail before activation.
func compatibleSlotOptions(catalog pluginCatalog, roleName, handoffSchema string) slotOptions {
	role := domain.RoleContract{
		SchemaVersion: domain.RoleContractSchemaVersion,
		Role:          roleName,
		HandoffSchema: handoffSchema,
	}
	compatible, excluded := splitCompatibleCandidates(providerContractsForRole(catalog, roleName), role)
	options := slotOptions{excluded: excluded}
	perID := make(map[string]int, len(compatible))
	for _, candidate := range compatible {
		perID[candidate.ID]++
	}
	var certified []domain.ProviderContract
	for _, candidate := range compatible {
		options.ids = append(options.ids, candidateRef(candidate, perID))
		if candidate.Ranked && candidate.CertificationDigest != "" {
			certified = append(certified, candidate)
		}
	}
	options.defaultID = preselectedRef(compatible, options.ids, perID, false)
	options.rankedRefs = make([]string, 0, len(certified))
	for _, candidate := range certified {
		options.rankedRefs = append(options.rankedRefs, candidateRef(candidate, perID))
	}
	options.rankedDefault = preselectedRef(certified, options.rankedRefs, perID, true)
	return options
}

// splitCompatibleCandidates separates the candidates a role may equip from the
// ones excluded with their reasons; native roles without certification are
// neither (they are not Weapons).
func splitCompatibleCandidates(candidates []domain.ProviderContract, role domain.RoleContract) (compatible []domain.ProviderContract, excluded []excludedProviderOption) {
	for _, candidate := range candidates {
		if candidate.Source == domain.ProviderSourceNativeRole && (!candidate.Ranked || candidate.CertificationDigest == "") {
			continue
		}
		if result := candidate.CheckRoleAffinity(role); !result.Compatible {
			excluded = append(excluded, excludedProviderOption{id: candidate.ID, reasons: result.Reasons})
			continue
		}
		compatible = append(compatible, candidate)
	}
	return compatible, excluded
}

// preselectedRef picks the UI pre-selection among candidates, whose references
// are refs in the same order: the first one the catalog marks default, else the
// last when preferLast (the highest certified version, as candidates are
// version-ordered) or the first listed. Empty when there are no candidates. It
// pre-selects only; the operator confirms.
func preselectedRef(candidates []domain.ProviderContract, refs []string, perID map[string]int, preferLast bool) string {
	if len(refs) == 0 {
		return ""
	}
	for _, candidate := range candidates {
		if candidate.Default {
			return candidateRef(candidate, perID)
		}
	}
	if preferLast {
		return refs[len(refs)-1]
	}
	return refs[0]
}

// candidateRef spells a candidate: "id@version" when the role has several
// compatible versions of that id, the plain id otherwise.
func candidateRef(candidate domain.ProviderContract, perID map[string]int) string {
	if perID[candidate.ID] > 1 {
		return domain.WeaponIdentity(candidate.ID, candidate.Version)
	}
	return candidate.ID
}

// compatibleProviderOptions is the single-Ranked view of compatibleSlotOptions:
// the id list, the Custom default, the pre-selected Ranked reference and the
// excluded candidates.
func compatibleProviderOptions(catalog pluginCatalog, roleName, handoffSchema string) (ids []string, defaultID, rankedID string, excluded []excludedProviderOption) {
	options := compatibleSlotOptions(catalog, roleName, handoffSchema)
	return options.ids, options.defaultID, options.rankedDefault, options.excluded
}
