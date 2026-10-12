package customprovider

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/tools/resolver"
)

const (
	// customHostEntrypoint is the normalized entrypoint of a host skill: its
	// SKILL.md prompt. The host path it was acquired from is never an entrypoint.
	customHostEntrypoint = "host.prompt"
	// customHostConnector is the connector of a Custom package acquired through
	// the wizard (the host loads the skill).
	customHostConnector = "host"
)

// MaterializeCustomProviders normalizes every typed Custom host Weapon into the
// installed representation `strategist check` and mission planning consume —
// providers/<id@version>/package.yaml and adapter.yaml — and rewrites the lock
// so its instance, binding and package, adapter and role-binding nodes carry
// the complete Custom contract. The external directory stays acquisition
// provenance only. Created directories are removed again if any step fails.
func MaterializeCustomProviders(strategistDir string, lock domain.PluginLockFile, providers map[string]Resolution) (domain.PluginLockFile, error) {
	var created []string
	for _, providerID := range SortedProviderIDs(providers) {
		next, dir, err := materializeCustomProvider(strategistDir, lock, providerID, providers[providerID])
		if dir != "" {
			created = append(created, dir)
		}
		if err != nil {
			removeAll(created)
			return domain.PluginLockFile{}, fmt.Errorf("materialize custom provider %q: %w", providerID, err)
		}
		lock = next
	}
	return lock, nil
}

// removeAll is the best-effort rollback of directories this call created.
func removeAll(paths []string) {
	for _, path := range paths {
		_ = os.RemoveAll(path) //nolint:errcheck // best-effort rollback of a directory this call created.
	}
}

// materializeCustomProvider stages one package and binds it; it returns the
// directory it created, if any, so the caller can roll it back.
func materializeCustomProvider(strategistDir string, lock domain.PluginLockFile, providerID string, resolution Resolution) (domain.PluginLockFile, string, error) {
	facts, dir, err := StageCustomPackage(strategistDir, resolution)
	if err != nil {
		return domain.PluginLockFile{}, "", err
	}
	next, err := bindCustomPackage(lock, providerID, facts, resolution)
	if err != nil {
		return domain.PluginLockFile{}, dir, err
	}
	return next, dir, nil
}

// SortedProviderIDs returns the keys of providers in ascending order.
func SortedProviderIDs(providers map[string]Resolution) []string {
	ids := make([]string, 0, len(providers))
	for id := range providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// bindCustomPackage replaces the wizard's bare-id instance, binding and lock
// nodes for providerID with the complete Custom evidence, keeping the slot's
// lifecycle generation.
func bindCustomPackage(lock domain.PluginLockFile, providerID string, facts domain.CustomPackageFacts, resolution Resolution) (domain.PluginLockFile, error) {
	generation, status, err := CurrentCustomBindingState(lock, facts.Slot)
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("read the slot's current binding: %w", err)
	}
	evidence, err := domain.NewCustomBindingEvidence(facts, generation, status)
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("derive the Custom binding: %w", err)
	}
	next := lock
	next.Inventory.Instances = renameCustomInstance(lock.Inventory.Instances, providerID, facts, resolution)
	next.Bindings = plugincatalog.ReplaceSlotBinding(append([]domain.SlotBinding(nil), lock.Bindings...), evidence.Binding)
	next.Lock.Nodes = replaceCustomNodes(lock.Lock.Nodes, providerID, evidence.Nodes)
	next.Lock.GraphDigest = resolver.DigestLockNodes(next.Lock.Nodes)
	next.Lock.ResolutionID = next.Lock.GraphDigest
	return next, nil
}

// CurrentCustomBindingState returns the generation (at least 1) and the lifecycle state of the single binding of
// slot in lock.
func CurrentCustomBindingState(lock domain.PluginLockFile, slot string) (int64, string, error) {
	binding, err := domain.SingleLockBindingForSlot(lock, slot)
	if err != nil {
		return 0, "", fmt.Errorf("find the slot binding: %w", err)
	}
	generation := binding.Generation
	if generation < 1 {
		generation = 1
	}
	// The wizard's lifecycle labels a freshly resolved binding "enabled"; a
	// materialized Custom binding is active, exactly as `provider add` records it.
	return generation, "active", nil
}

func renameCustomInstance(instances []domain.InstalledInstance, providerID string, facts domain.CustomPackageFacts, resolution Resolution) []domain.InstalledInstance {
	out := append([]domain.InstalledInstance(nil), instances...)
	for i := range out {
		if out[i].ID != providerID {
			continue
		}
		out[i].ID = facts.InstanceID()
		out[i].PackageDigest, out[i].AdapterDigest = facts.PackageDigest, facts.AdapterDigest
		out[i].ConnectorID, out[i].Entrypoint = facts.ConnectorID, facts.Entrypoint
		out[i].ProviderOrigin, out[i].SeedPath = resolution.Package.Origin, resolution.Package.SeedPath // acquisition provenance only
	}
	return out
}

// replaceCustomNodes drops every node the bare-id plan produced for providerID
// (its package/adapter nodes and its role-binding node) and publishes nodes.
func replaceCustomNodes(existing []domain.PluginLockNode, providerID string, nodes []domain.PluginLockNode) []domain.PluginLockNode {
	out := make([]domain.PluginLockNode, 0, len(existing)+len(nodes))
	for _, node := range existing {
		if node.ID == providerID || strings.HasSuffix(node.ID, ":"+providerID) {
			continue
		}
		out = append(out, node)
	}
	return append(out, nodes...)
}

// PinCustomInstanceRefs makes active.yaml name a Custom package by its
// versioned instance id, the only spelling check and planning accept.
func PinCustomInstanceRefs(wc domain.WizardConfig, lock domain.PluginLockFile) domain.WizardConfig {
	wc.DiscoveryProvider = customInstanceRef(lock, wc.DiscoveryProvider, string(domain.SlotDiscovery), wc.DiscoveryMode)
	wc.RefinementProvider = customInstanceRef(lock, wc.RefinementProvider, string(domain.SlotRefinement), wc.RefinementMode)
	wc.ExecutionProvider = customInstanceRef(lock, wc.ExecutionProvider, string(domain.SlotExecution), wc.ExecutionMode)
	return wc
}

func customInstanceRef(lock domain.PluginLockFile, provider, slot, mode string) string {
	if mode != domain.SlotBindingModeCustom {
		return provider
	}
	for _, binding := range lock.Bindings {
		if binding.Slot == slot && binding.EffectiveMode() == domain.SlotBindingModeCustom && strings.HasPrefix(binding.InstalledInstanceID, provider+"@") {
			return binding.InstalledInstanceID
		}
	}
	return provider
}
