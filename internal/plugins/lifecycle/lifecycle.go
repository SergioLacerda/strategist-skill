// Package lifecycle tracks plugin lifecycle transactions and binding changes.
package lifecycle

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

const (
	// StateResolved means a candidate is selected but not staged.
	StateResolved = "resolved"
	// StateVerified means trust and contract checks passed.
	StateVerified = "verified"
	// StateStaged means the candidate is prepared for probing.
	StateStaged = "staged"
	// StateProbed means the candidate passed its probe.
	StateProbed = "probed"
	// StateActive means the binding points at the candidate.
	StateActive = "active"
	// StateFailed means lifecycle activation failed.
	StateFailed = "failed"
	// StateDeprecated means an instance is obsolete.
	StateDeprecated = "deprecated"
	// StateQuarantined means an instance is blocked from use.
	StateQuarantined = "quarantined"
	// StateComplete means the transaction finished successfully.
	StateComplete = "complete"
	// StateRolledBack means the transaction restored a previous binding.
	StateRolledBack = "rolled_back"
)

// Store keeps plugin lifecycle state separated by resource authority.
type Store struct {
	Inventory    domain.PluginInventory
	Bindings     []domain.SlotBinding
	Transactions map[string]*Transaction
	Dependents   map[string][]string
}

// Transaction is the journaled lifecycle operation for one slot candidate.
type Transaction struct {
	ID                string
	Slot              string
	CandidateInstance string
	RollbackTarget    string
	State             string
	FromGeneration    int64
	ToGeneration      int64
	ProbePassed       bool
	Journal           []JournalEntry
}

// JournalEntry records one idempotent transition.
type JournalEntry struct {
	State string
	Code  string
}

// JournalStates returns the transaction states in append order.
func (t Transaction) JournalStates() []string {
	states := make([]string, 0, len(t.Journal))
	for _, entry := range t.Journal {
		states = append(states, entry.State)
	}
	return states
}

// NewStore returns an empty lifecycle store.
func NewStore() *Store {
	return &Store{
		Inventory:    domain.PluginInventory{SchemaVersion: "strategist-plugin-inventory/v1"},
		Transactions: map[string]*Transaction{},
		Dependents:   map[string][]string{},
	}
}

// Begin creates or resumes a transaction for a candidate instance.
func (s *Store) Begin(id, slot, candidateInstance string) (Transaction, error) {
	if tx, ok := s.Transactions[id]; ok {
		return *tx, nil
	}
	if _, ok := s.Instance(candidateInstance); !ok {
		return Transaction{}, fmt.Errorf("candidate_instance_missing: %s", candidateInstance)
	}
	binding, ok := s.Binding(slot)
	if !ok {
		return Transaction{}, fmt.Errorf("binding_missing: %s", slot)
	}
	tx := &Transaction{
		ID:                id,
		Slot:              slot,
		CandidateInstance: candidateInstance,
		RollbackTarget:    binding.InstalledInstanceID,
		State:             StateResolved,
		FromGeneration:    binding.Generation,
		ToGeneration:      binding.Generation,
	}
	tx.append(StateResolved, "transaction_started")
	s.Transactions[id] = tx
	return *tx, nil
}

// Stage marks the candidate staged. Repeating it is idempotent.
func (s *Store) Stage(txID string) error {
	tx, err := s.transaction(txID)
	if err != nil {
		return err
	}
	if tx.State == StateStaged || tx.State == StateProbed || tx.State == StateActive || tx.State == StateComplete {
		return nil
	}
	if tx.State != StateResolved {
		return fmt.Errorf("stage_invalid_state: %s", tx.State)
	}
	return s.transitionCandidate(tx, StateStaged, "candidate_staged")
}

// Probe records a non-mutating probe result. Passing probes can activate.
func (s *Store) Probe(txID string, passed bool) error {
	tx, err := s.transaction(txID)
	if err != nil {
		return err
	}
	if tx.State == StateProbed && tx.ProbePassed == passed {
		return nil
	}
	if tx.State != StateStaged {
		return fmt.Errorf("probe_invalid_state: %s", tx.State)
	}
	tx.ProbePassed = passed
	if !passed {
		return s.transitionCandidate(tx, StateFailed, "probe_failed")
	}
	return s.transitionCandidate(tx, StateProbed, "probe_passed")
}

// Activate performs the compare-and-swap binding switch.
func (s *Store) Activate(txID string, expectedGeneration int64) error {
	tx, err := s.transaction(txID)
	if err != nil {
		return err
	}
	if tx.State == StateComplete {
		return nil
	}
	if tx.State != StateProbed || !tx.ProbePassed {
		return fmt.Errorf("activation_requires_successful_probe")
	}
	bindingIndex, binding, ok := s.bindingIndex(tx.Slot)
	if !ok {
		return fmt.Errorf("binding_missing: %s", tx.Slot)
	}
	if binding.Generation != expectedGeneration {
		return fmt.Errorf("binding_generation_conflict: got %d want %d", expectedGeneration, binding.Generation)
	}
	binding.InstalledInstanceID = tx.CandidateInstance
	binding.Generation++
	s.Bindings[bindingIndex] = binding
	s.markLastKnownGood(tx.CandidateInstance)
	if err := s.transitionCandidate(tx, StateActive, "binding_activated"); err != nil {
		return err
	}
	tx.ToGeneration = binding.Generation
	tx.State = StateComplete
	tx.append(StateComplete, "transaction_complete")
	return nil
}

// Rollback restores the recorded last-known-good binding target.
func (s *Store) Rollback(txID string) error {
	tx, err := s.transaction(txID)
	if err != nil {
		return err
	}
	if tx.State == StateRolledBack || tx.State == StateComplete {
		return nil
	}
	rollbackTarget := tx.RollbackTarget
	if _, ok := s.Instance(rollbackTarget); !ok {
		rollbackTarget = s.lastKnownGoodInstanceID()
	}
	if rollbackTarget != "" {
		idx, binding, ok := s.bindingIndex(tx.Slot)
		if ok {
			binding.InstalledInstanceID = rollbackTarget
			s.Bindings[idx] = binding
		}
	}
	s.setInstanceState(tx.CandidateInstance, StateFailed)
	tx.State = StateRolledBack
	tx.append(StateRolledBack, "rollback_complete")
	return nil
}
