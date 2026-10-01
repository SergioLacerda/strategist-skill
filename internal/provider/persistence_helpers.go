package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func validateProviderID(id string) error {
	if id == "" || id == "." || id == ".." || strings.ContainsAny(id, `/\\`) {
		return fmt.Errorf("provider id invalid: %q", id)
	}
	return nil
}

func transactionID(instanceID, slot string) string {
	return "provider-add-" + slot + "-" + instanceID
}

func newLockBindingGeneration(lock domain.PluginLockFile, slot string) int64 {
	return nextGeneration(lock, slot)
}

func bindingGeneration(lock domain.PluginLockFile, slot string) int64 {
	for _, binding := range lock.Bindings {
		if binding.Slot == slot {
			return binding.Generation
		}
	}
	return 0
}

func compileWorkspace(root string) error {
	if _, err := os.Stat(filepath.Join(root, "active.yaml")); err != nil {
		return fmt.Errorf("compile prerequisite active.yaml: %w", err)
	}
	if err := (compile.Compiler{}).CompileAll(root, filepath.Join(root, "knowledge.index.yaml")); err != nil {
		return fmt.Errorf("compile provider runtime: %w", err)
	}
	return nil
}
func existingBinding(bindings []domain.SlotBinding, slot string) (domain.SlotBinding, bool) {
	for _, binding := range bindings {
		if binding.Slot == slot {
			return binding, true
		}
	}
	return domain.SlotBinding{}, false
}

func replaceInstance(instances []domain.InstalledInstance, candidate domain.InstalledInstance) []domain.InstalledInstance {
	for i, instance := range instances {
		if instance.ID == candidate.ID {
			instances[i] = candidate
			return instances
		}
	}
	return append(instances, candidate)
}

func replaceBinding(bindings []domain.SlotBinding, candidate domain.SlotBinding) []domain.SlotBinding {
	for i, binding := range bindings {
		if binding.Slot == candidate.Slot {
			candidate.Generation = binding.Generation + 1
			bindings[i] = candidate
			return bindings
		}
	}
	return append(bindings, candidate)
}

// replaceLockNodes publishes nodes, replacing any node with the same id and kind.
func replaceLockNodes(lock domain.PluginLock, nodes []domain.PluginLockNode) domain.PluginLock {
	filtered := make([]domain.PluginLockNode, 0, len(lock.Nodes)+len(nodes))
	for _, node := range lock.Nodes {
		if !hasLockNode(nodes, node) {
			filtered = append(filtered, node)
		}
	}
	merged := make([]domain.PluginLockNode, 0, len(filtered)+len(nodes))
	merged = append(merged, filtered...)
	merged = append(merged, nodes...)
	lock.Nodes = merged
	return lock
}

func hasLockNode(nodes []domain.PluginLockNode, candidate domain.PluginLockNode) bool {
	for _, node := range nodes {
		if node.ID == candidate.ID && node.Kind == candidate.Kind {
			return true
		}
	}
	return false
}
