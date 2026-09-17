package lifecycle

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Validate checks the durable inventory/binding boundary before a lifecycle
// store is used. Persistence belongs to the installer, while this package
// owns the invariants every restored store must satisfy.
func (s *Store) Validate() error {
	if s == nil {
		return fmt.Errorf("lifecycle_store_nil")
	}
	seenInstances, err := validateInstances(s.Inventory.Instances)
	if err != nil {
		return err
	}
	return validateBindings(s.Bindings, seenInstances)
}

func validateInstances(instances []domain.InstalledInstance) (map[string]struct{}, error) {
	seenInstances := make(map[string]struct{}, len(instances))
	for _, instance := range instances {
		if instance.ID == "" {
			return nil, fmt.Errorf("instance_id_missing")
		}
		if _, exists := seenInstances[instance.ID]; exists {
			return nil, fmt.Errorf("duplicate_instance: %s", instance.ID)
		}
		seenInstances[instance.ID] = struct{}{}
	}
	return seenInstances, nil
}

func validateBindings(bindings []domain.SlotBinding, instances map[string]struct{}) error {
	seenSlots := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		if _, exists := seenSlots[binding.Slot]; exists {
			return fmt.Errorf("duplicate_binding_slot: %s", binding.Slot)
		}
		seenSlots[binding.Slot] = struct{}{}
		if err := validateBinding(binding, instances); err != nil {
			return err
		}
	}
	return nil
}

func validateBinding(binding domain.SlotBinding, instances map[string]struct{}) error {
	if binding.Slot == "" || binding.InstalledInstanceID == "" {
		return fmt.Errorf("binding_incomplete: %s", binding.Slot)
	}
	if !binding.ValidMode() {
		return fmt.Errorf("binding_invalid_mode: slot=%s mode=%s", binding.Slot, binding.Mode)
	}
	if binding.Generation < 0 {
		return fmt.Errorf("binding_negative_generation: %s", binding.Slot)
	}
	if _, exists := instances[binding.InstalledInstanceID]; !exists {
		return fmt.Errorf("binding_instance_missing: slot=%s instance=%s", binding.Slot, binding.InstalledInstanceID)
	}
	return nil
}

// Recover rolls back every incomplete transaction.
func (s *Store) Recover() error {
	for id, tx := range s.Transactions {
		if tx.State == StateComplete || tx.State == StateRolledBack {
			continue
		}
		if err := s.Rollback(id); err != nil {
			return err
		}
	}
	return nil
}

// Uninstall removes an unbound, unreferenced instance from inventory.
func (s *Store) Uninstall(instanceID string) error {
	for _, binding := range s.Bindings {
		if binding.InstalledInstanceID == instanceID {
			return fmt.Errorf("uninstall_blocked_bound_instance: %s", instanceID)
		}
	}
	if dependents := s.Dependents[instanceID]; len(dependents) > 0 {
		return fmt.Errorf("uninstall_blocked_required_dependent: %s", instanceID)
	}
	for i, instance := range s.Inventory.Instances {
		if instance.ID == instanceID {
			s.Inventory.Instances = append(s.Inventory.Instances[:i], s.Inventory.Instances[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("instance_missing: %s", instanceID)
}

// Quarantine marks an instance unusable and disables any binding that points at it.
func (s *Store) Quarantine(instanceID, reasonCode string) error {
	return s.disableInstance(instanceID, StateQuarantined, "quarantined", reasonCode)
}

// Deprecate marks an instance obsolete while preserving inventory history.
func (s *Store) Deprecate(instanceID, reasonCode string) error {
	return s.disableInstance(instanceID, StateDeprecated, "deprecated", reasonCode)
}

func (s *Store) disableInstance(instanceID, instanceState, bindingStatus, reasonCode string) error {
	found := false
	for i, instance := range s.Inventory.Instances {
		if instance.ID != instanceID {
			continue
		}
		instance.State = instanceState
		instance.LastKnownGood = false
		instance.VerificationEvidence = reasonCode
		s.Inventory.Instances[i] = instance
		found = true
	}
	if !found {
		return fmt.Errorf("instance_missing: %s", instanceID)
	}
	for i, binding := range s.Bindings {
		if binding.InstalledInstanceID == instanceID {
			binding.Status = bindingStatus
			binding.Generation++
			s.Bindings[i] = binding
		}
	}
	return nil
}

// Instance returns one installed instance.
func (s *Store) Instance(id string) (domain.InstalledInstance, bool) {
	for _, instance := range s.Inventory.Instances {
		if instance.ID == id {
			return instance, true
		}
	}
	return domain.InstalledInstance{}, false
}

// Binding returns one active slot binding.
func (s *Store) Binding(slot string) (domain.SlotBinding, bool) {
	_, binding, ok := s.bindingIndex(slot)
	return binding, ok
}

// Transaction returns one transaction by ID. Missing IDs return an empty value.
func (s *Store) Transaction(id string) Transaction {
	if tx, ok := s.Transactions[id]; ok {
		return *tx
	}
	return Transaction{}
}

func (s *Store) transaction(id string) (*Transaction, error) {
	tx, ok := s.Transactions[id]
	if !ok {
		return nil, fmt.Errorf("transaction_missing: %s", id)
	}
	return tx, nil
}

func (s *Store) transitionCandidate(tx *Transaction, state, code string) error {
	if ok := s.setInstanceState(tx.CandidateInstance, state); !ok {
		return fmt.Errorf("candidate_instance_missing: %s", tx.CandidateInstance)
	}
	tx.State = state
	tx.append(state, code)
	return nil
}
