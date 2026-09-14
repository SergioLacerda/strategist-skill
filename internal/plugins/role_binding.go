package plugins

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// RoleBindingLockKind is the PluginLockNode.Kind used to record a resolved
// Role -> Provider binding inside the existing plugin lock graph, instead of
// creating a parallel lock store (proposal.md Decision 4: lifecycle reuse).
const RoleBindingLockKind = "role_provider_binding"

// ResolveRoleBinding picks the single Provider that binds to role among
// candidates, applying deterministic collision and compatibility rules
// (proposal.md Decision 5; tasks.md Task 3):
//
//   - candidates are first filtered to those declaring role.Role as their
//     CanonicalRole;
//   - if two or more candidates share the same ID from different Source
//     values (an external provider implicitly shadowing an embedded one, or
//     vice versa), resolution fails with a stable id_shadowing error unless
//     shadowOverride names the Source that must win every such collision;
//   - among the surviving candidates, exactly one must be compatible with
//     role (via ProviderContract.CheckRoleCompatibility) — zero compatible
//     candidates is role_binding_missing. More than one is
//     role_binding_ambiguous, unless preferredProviderID names one of the
//     compatible candidates: a legitimately ambiguous role (e.g. more than
//     one embedded Provider can serve "archivist") is expected, not an
//     error, once a preference disambiguates it — "default is a selection
//     preference, not proof of ... readiness" (proposal.md Decision 3).
//     preferredProviderID never overrides ID-shadowing collisions — those
//     always require shadowOverride.
//
// ResolveRoleBinding never persists anything: the caller records the
// returned ProviderBinding as a lock node (see RoleBindingLockNode) inside
// the existing PluginLock, and as a SlotBinding for workspace-owned
// persistence — preserving offline replay, staged activation, and
// last-known-good rollback through the existing lifecycle machinery instead
// of a second binding store.
func ResolveRoleBinding(role domain.RoleContract, candidates []domain.ProviderContract, shadowOverride domain.ProviderSource, preferredProviderID string) (domain.ProviderBinding, error) {
	scoped := make([]domain.ProviderContract, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.CanonicalRole == role.Role {
			scoped = append(scoped, candidate)
		}
	}

	deduped, err := deduplicateShadowedIDs(scoped, shadowOverride)
	if err != nil {
		return domain.ProviderBinding{}, err
	}

	var compatible []domain.ProviderContract
	for _, candidate := range deduped {
		if candidate.CheckRoleCompatibility(role).Compatible {
			compatible = append(compatible, candidate)
		}
	}
	sort.Slice(compatible, func(i, j int) bool { return compatible[i].ID < compatible[j].ID })

	switch len(compatible) {
	case 0:
		return domain.ProviderBinding{}, fmt.Errorf("role_binding_missing: role=%s no compatible provider among %d candidate(s)", role.Role, len(scoped))
	case 1:
		return domain.ResolveProviderBinding(role, compatible[0]), nil
	default:
		if preferredProviderID != "" {
			if winner, ok := findByID(compatible, preferredProviderID); ok {
				return domain.ResolveProviderBinding(role, winner), nil
			}
		}
		return domain.ProviderBinding{}, fmt.Errorf("role_binding_ambiguous: role=%s candidates=%s", role.Role, strings.Join(candidateIDs(compatible), ","))
	}
}

// deduplicateShadowedIDs collapses candidates that share an ID across
// different Source values into the single candidate whose Source equals
// shadowOverride. An empty shadowOverride, or a collision none of whose
// candidates match it, is rejected with a stable id_shadowing error — a
// Provider ID never silently shadows another.
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
		group := byID[id]
		if len(group) == 1 {
			deduped = append(deduped, group[0])
			continue
		}
		if shadowOverride == "" {
			return nil, fmt.Errorf("id_shadowing: id=%s sources=%s requires an explicit shadow override", id, sourceList(group))
		}
		winner, ok := findBySource(group, shadowOverride)
		if !ok {
			return nil, fmt.Errorf("id_shadowing: id=%s sources=%s does not include override source %s", id, sourceList(group), shadowOverride)
		}
		deduped = append(deduped, winner)
	}
	return deduped, nil
}

func findBySource(group []domain.ProviderContract, source domain.ProviderSource) (domain.ProviderContract, bool) {
	for _, candidate := range group {
		if candidate.Source == source {
			return candidate, true
		}
	}
	return domain.ProviderContract{}, false
}

func findByID(candidates []domain.ProviderContract, id string) (domain.ProviderContract, bool) {
	for _, candidate := range candidates {
		if candidate.ID == id {
			return candidate, true
		}
	}
	return domain.ProviderContract{}, false
}

func sourceList(group []domain.ProviderContract) string {
	sources := make([]string, 0, len(group))
	for _, candidate := range group {
		sources = append(sources, string(candidate.Source))
	}
	sort.Strings(sources)
	return strings.Join(sources, ",")
}

func candidateIDs(candidates []domain.ProviderContract) []string {
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.ID)
	}
	sort.Strings(ids)
	return ids
}

// RoleBindingLockNode renders a resolved ProviderBinding as the lock node
// recorded into the existing PluginLock graph (see resolver.go), so a
// role/provider binding participates in the same digest-pinned, replayable
// lock as every other resolved resource — no parallel lock file.
func RoleBindingLockNode(binding domain.ProviderBinding) domain.PluginLockNode {
	return domain.PluginLockNode{
		ID:     binding.Role.Role + ":" + binding.Provider.ID,
		Kind:   RoleBindingLockKind,
		Digest: roleBindingDigest(binding),
	}
}

func roleBindingDigest(binding domain.ProviderBinding) string {
	var b strings.Builder
	b.WriteString(binding.Role.Role)
	b.WriteString("\t")
	b.WriteString(binding.Role.SchemaVersion)
	b.WriteString("\t")
	b.WriteString(binding.Provider.ID)
	b.WriteString("\t")
	b.WriteString(binding.Provider.Version)
	b.WriteString("\t")
	b.WriteString(string(binding.Provider.Source))
	sum := sha256.Sum256([]byte(b.String()))
	return fmt.Sprintf("sha256:%x", sum)
}
