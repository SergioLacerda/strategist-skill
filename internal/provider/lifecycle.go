package provider

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
)

func activateThroughLifecycle(old, candidate domain.PluginLockFile, instanceID, slot string) (domain.PluginInventory, []domain.SlotBinding, error) {
	if !hasBinding(old.Bindings, slot) {
		return candidate.Inventory, candidate.Bindings, nil
	}
	store := lifecycle.NewStore()
	store.Inventory = old.Inventory
	store.Inventory.Instances = replaceInstance(store.Inventory.Instances, findInstance(candidate.Inventory, instanceID))
	store.Bindings = append([]domain.SlotBinding(nil), old.Bindings...)
	if err := store.Validate(); err != nil {
		return domain.PluginInventory{}, nil, fmt.Errorf("validate existing lifecycle state: %w", err)
	}
	tx, err := store.Begin(transactionID(instanceID, slot), slot, instanceID)
	if err != nil {
		return domain.PluginInventory{}, nil, fmt.Errorf("begin lifecycle transaction: %w", err)
	}
	if err := store.Stage(tx.ID); err != nil {
		return domain.PluginInventory{}, nil, fmt.Errorf("stage lifecycle transaction: %w", err)
	}
	if err := store.ProbeResult(tx.ID, lifecycle.ProbeOutcome{Status: domain.ReadinessReady, ReasonCode: "static_contract_probe_passed", Detail: "local contract probe; live invocation remains unknown"}); err != nil {
		return domain.PluginInventory{}, nil, fmt.Errorf("record lifecycle probe result: %w", err)
	}
	if err := store.Activate(tx.ID, currentGeneration(old.Bindings, slot)); err != nil {
		return domain.PluginInventory{}, nil, fmt.Errorf("activate lifecycle transaction: %w", err)
	}
	return store.Inventory, store.Bindings, nil
}

func hasBinding(bindings []domain.SlotBinding, slot string) bool {
	_, ok := existingBinding(bindings, slot)
	return ok
}

func currentGeneration(bindings []domain.SlotBinding, slot string) int64 {
	binding, ok := existingBinding(bindings, slot)
	if !ok {
		return 0
	}
	return binding.Generation
}

func findInstance(inventory domain.PluginInventory, id string) domain.InstalledInstance {
	for _, instance := range inventory.Instances {
		if instance.ID == id {
			return instance
		}
	}
	return domain.InstalledInstance{ID: id}
}
