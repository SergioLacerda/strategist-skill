package plugins

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// providerIdentity is the candidate identity "id@version": two versions of one
// id from the same Source are distinct candidates and never shadow each other
// (ADR-0061 Decision 8).
func providerIdentity(candidate domain.ProviderContract) string {
	return domain.WeaponIdentity(candidate.ID, candidate.Version)
}

// deduplicateShadowedIDs collapses candidates that share an ID across
// different Source values into the candidates whose Source equals
// shadowOverride. An empty shadowOverride, or a collision none of whose
// candidates match it, is rejected with a stable id_shadowing error — a
// Provider ID never silently shadows another. Several versions of one ID from a
// single Source are not a collision: each stays a candidate.
func deduplicateShadowedIDs(candidates []domain.ProviderContract, shadowOverride domain.ProviderSource) ([]domain.ProviderContract, error) {
	byID := map[string][]domain.ProviderContract{}
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if _, seen := byID[candidate.ID]; !seen {
			ids = append(ids, candidate.ID)
		}
		byID[candidate.ID] = append(byID[candidate.ID], candidate)
	}
	sort.Strings(ids)

	deduped := make([]domain.ProviderContract, 0, len(candidates))
	for _, id := range ids {
		winners, err := deduplicateShadowedGroup(id, byID[id], shadowOverride)
		if err != nil {
			return nil, err
		}
		deduped = append(deduped, winners...)
	}
	return deduped, nil
}

func deduplicateShadowedGroup(id string, group []domain.ProviderContract, shadowOverride domain.ProviderSource) ([]domain.ProviderContract, error) {
	if singleSource(group) {
		return group, nil
	}
	if shadowOverride == "" {
		return nil, fmt.Errorf("id_shadowing: id=%s sources=%s requires an explicit shadow override", id, sourceList(group))
	}
	winners := filterBySource(group, shadowOverride)
	if len(winners) == 0 {
		return nil, fmt.Errorf("id_shadowing: id=%s sources=%s does not include override source %s", id, sourceList(group), shadowOverride)
	}
	return winners, nil
}

// singleSource reports whether every candidate of the group shares one Source,
// which makes it a set of versions of one Weapon rather than a shadowing.
func singleSource(group []domain.ProviderContract) bool {
	for _, candidate := range group[1:] {
		if candidate.Source != group[0].Source {
			return false
		}
	}
	return true
}

func filterBySource(group []domain.ProviderContract, source domain.ProviderSource) []domain.ProviderContract {
	var winners []domain.ProviderContract
	for _, candidate := range group {
		if candidate.Source == source {
			winners = append(winners, candidate)
		}
	}
	return winners
}

func sourceList(group []domain.ProviderContract) string {
	sources := make([]string, 0, len(group))
	for _, candidate := range group {
		sources = append(sources, string(candidate.Source))
	}
	sort.Strings(sources)
	return strings.Join(sources, ",")
}
