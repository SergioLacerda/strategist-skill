package onboarding

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
)

// ApplyPluginBinding activates one slot binding in store; the binding is blocked when probe fails.
func ApplyPluginBinding(store *lifecycle.Store, desired domain.SlotBinding, probe pluginProbeFunc) error {
	current, ok := store.Binding(desired.Slot)
	if !ok {
		store.Bindings = append(store.Bindings, desired)
		return nil
	}
	if current.InstalledInstanceID == desired.InstalledInstanceID {
		return nil
	}
	instance, ok := store.Instance(desired.InstalledInstanceID)
	if !ok {
		return fmt.Errorf("planned_instance_missing: %s", desired.InstalledInstanceID)
	}
	return SwitchPluginBinding(store, current, desired, instance, probe)
}

// ApplyPluginBindingWithResult activates one slot binding in store using probe and records the probe outcome.
func ApplyPluginBindingWithResult(store *lifecycle.Store, desired domain.SlotBinding, probe pluginProbeResultFunc) error {
	current, ok := store.Binding(desired.Slot)
	if !ok {
		store.Bindings = append(store.Bindings, desired)
		return nil
	}
	if current.InstalledInstanceID == desired.InstalledInstanceID {
		return nil
	}
	instance, ok := store.Instance(desired.InstalledInstanceID)
	if !ok {
		return fmt.Errorf("planned_instance_missing: %s", desired.InstalledInstanceID)
	}
	return SwitchPluginBindingWithResult(store, current, desired, instance, probe)
}

// SwitchPluginBindingWithResult switches one slot binding in store to a new instance using probe and records the outcome.
func SwitchPluginBindingWithResult(store *lifecycle.Store, current, desired domain.SlotBinding, instance domain.InstalledInstance, probe pluginProbeResultFunc) error {
	tx, err := BeginPluginBinding(store, desired)
	if err != nil {
		return err
	}
	if err := store.ProbeResult(tx.ID, probe(desired, instance)); err != nil {
		return fmt.Errorf("probe plugin lifecycle transaction: %w", err)
	}
	return activatePluginBinding(store, tx, current.Generation)
}

// SwitchPluginBinding switches one slot binding in store to a new instance; it is blocked when probe fails.
func SwitchPluginBinding(store *lifecycle.Store, current, desired domain.SlotBinding, instance domain.InstalledInstance, probe pluginProbeFunc) error {
	tx, err := BeginPluginBinding(store, desired)
	if err != nil {
		return err
	}
	if err := store.Probe(tx.ID, probe(desired, instance)); err != nil {
		return fmt.Errorf("probe plugin lifecycle transaction: %w", err)
	}
	return activatePluginBinding(store, tx, current.Generation)
}

// BeginPluginBinding starts the activation of one slot binding in store and returns the lifecycle transaction.
func BeginPluginBinding(store *lifecycle.Store, desired domain.SlotBinding) (lifecycle.Transaction, error) {
	txID := "plugin-onboarding-" + desired.Slot + "-" + desired.InstalledInstanceID
	tx, err := store.Begin(txID, desired.Slot, desired.InstalledInstanceID)
	if err != nil {
		return lifecycle.Transaction{}, fmt.Errorf("begin plugin lifecycle transaction: %w", err)
	}
	if err := store.Stage(tx.ID); err != nil {
		return lifecycle.Transaction{}, fmt.Errorf("stage plugin lifecycle transaction: %w", err)
	}
	return tx, nil
}

func activatePluginBinding(store *lifecycle.Store, tx lifecycle.Transaction, generation int64) error {
	if err := store.Activate(tx.ID, generation); err != nil {
		if rollbackErr := store.Rollback(tx.ID); rollbackErr != nil {
			return fmt.Errorf("activate plugin lifecycle transaction: %w; rollback: %v", err, rollbackErr)
		}
		return fmt.Errorf("activate plugin lifecycle transaction: %w", err)
	}
	return nil
}
