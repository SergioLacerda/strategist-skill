// Package plugins resolves plugin candidates into deterministic lock graphs.
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
//   - candidates are first filtered to those declaring role.Role in their
//     explicit role affinity;
//   - if two or more candidates share the same ID from different Source
//     values (an external provider implicitly shadowing an embedded one, or
//     vice versa), resolution fails with a stable id_shadowing error unless
//     shadowOverride names the Source that must win every such collision;
//   - among the surviving candidates, exactly one must declare affinity
//     with role (via ProviderContract.CheckRoleAffinity) — zero compatible
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
	scoped := compatibleProviders(role, candidates)

	deduped, err := deduplicateShadowedIDs(scoped, shadowOverride)
	if err != nil {
		return domain.ProviderBinding{}, err
	}

	compatible := sortedCompatibleProviders(role, deduped)
	return chooseRoleBinding(role, compatible, len(scoped), preferredProviderID)
}

func compatibleProviders(role domain.RoleContract, candidates []domain.ProviderContract) []domain.ProviderContract {
	var scoped []domain.ProviderContract
	for _, candidate := range candidates {
		if candidate.CheckRoleAffinity(role).Compatible {
			scoped = append(scoped, candidate)
		}
	}
	return scoped
}

func sortedCompatibleProviders(role domain.RoleContract, candidates []domain.ProviderContract) []domain.ProviderContract {
	compatible := compatibleProviders(role, candidates)
	sort.Slice(compatible, func(i, j int) bool { return providerIdentity(compatible[i]) < providerIdentity(compatible[j]) })
	return compatible
}

func chooseRoleBinding(role domain.RoleContract, compatible []domain.ProviderContract, scopedCount int, preferredProviderID string) (domain.ProviderBinding, error) {
	switch len(compatible) {
	case 0:
		return domain.ProviderBinding{}, fmt.Errorf("role_binding_missing: role=%s no compatible provider among %d candidate(s)", role.Role, scopedCount)
	case 1:
		return domain.ProviderBinding{Role: role, Provider: compatible[0], Compatibility: compatible[0].CheckRoleAffinity(role)}, nil
	default:
		if preferredProviderID != "" {
			if winner, ok := findByRef(compatible, preferredProviderID); ok {
				return domain.ProviderBinding{Role: role, Provider: winner, Compatibility: winner.CheckRoleAffinity(role)}, nil
			}
		}
		return domain.ProviderBinding{}, fmt.Errorf("role_binding_ambiguous: role=%s candidates=%s", role.Role, strings.Join(candidateIDs(compatible), ","))
	}
}

// findByRef resolves a preferred "id" or "id@version" reference. A plain id
// resolves only while exactly one version of it is a candidate; with several it
// stays ambiguous rather than picking one.
func findByRef(candidates []domain.ProviderContract, ref string) (domain.ProviderContract, bool) {
	id, version := domain.ParseWeaponRef(ref)
	var found domain.ProviderContract
	count := 0
	for _, candidate := range candidates {
		if candidate.ID == id && (version == "" || candidate.Version == version) {
			found = candidate
			count++
		}
	}
	return found, count == 1
}

func candidateIDs(candidates []domain.ProviderContract) []string {
	ids := make([]string, 0, len(candidates))
	perID := make(map[string]int, len(candidates))
	for _, candidate := range candidates {
		perID[candidate.ID]++
	}
	for _, candidate := range candidates {
		if perID[candidate.ID] > 1 {
			ids = append(ids, providerIdentity(candidate))
			continue
		}
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
