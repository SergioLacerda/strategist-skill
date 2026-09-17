package install

import (
	"fmt"
	"sort"
)

// DependencyViolation names one catalog provider's declared dependency that
// cannot be resolved against the rest of the catalog — a master skill (Arma)
// is never usable while a declared dependency is absent (see
// .analysis/refined/20260913-role-skill-weapon-taxonomy/proposal.md § 3,
// the brainstorming -> writing-plans example). Checked against both
// AuxiliaryTools (a flat skill-id list) and Dependencies (the structured,
// resolver-native form already consumed by plugins.Resolve via
// catalogDependencies) — a catalog entry may use either or both.
type DependencyViolation struct {
	ProviderID   string
	DependencyID string
	Source       string // "auxiliary_tools_allowed" | "dependencies"
}

func (v DependencyViolation) String() string {
	return fmt.Sprintf("provider %q declares dependency %q (via %s) which is not present in the catalog", v.ProviderID, v.DependencyID, v.Source)
}

// ValidateCatalogDependencies reports every declared-but-unresolvable
// dependency across catalog. It never mutates the catalog and never drops a
// provider — the caller decides whether a violation blocks ingestion
// (tasks.md Task 2: the directory-to-catalog generator must call this and
// flag, not silently catalogue, an offending package).
func ValidateCatalogDependencies(catalog pluginCatalog) []DependencyViolation {
	known := make(map[string]bool, len(catalog.Providers))
	for _, provider := range catalog.Providers {
		known[provider.ID] = true
	}

	var violations []DependencyViolation
	for _, provider := range catalog.Providers {
		violations = append(violations, missingToolViolations(provider, known)...)
		violations = append(violations, missingDependencyViolations(provider, known)...)
	}
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].ProviderID != violations[j].ProviderID {
			return violations[i].ProviderID < violations[j].ProviderID
		}
		return violations[i].DependencyID < violations[j].DependencyID
	})
	return violations
}

func missingToolViolations(provider pluginCatalogProvider, known map[string]bool) []DependencyViolation {
	var violations []DependencyViolation
	for _, toolID := range provider.AuxiliaryTools {
		if !known[toolID] {
			violations = append(violations, DependencyViolation{ProviderID: provider.ID, DependencyID: toolID, Source: "auxiliary_tools_allowed"})
		}
	}
	return violations
}

func missingDependencyViolations(provider pluginCatalogProvider, known map[string]bool) []DependencyViolation {
	var violations []DependencyViolation
	for _, dep := range provider.Dependencies {
		if dep.Optional || known[dep.ID] {
			continue
		}
		violations = append(violations, DependencyViolation{ProviderID: provider.ID, DependencyID: dep.ID, Source: "dependencies"})
	}
	return violations
}
