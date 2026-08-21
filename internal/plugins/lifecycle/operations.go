package lifecycle

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

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
