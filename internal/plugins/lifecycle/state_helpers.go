package lifecycle

import "github.com/SergioLacerda/strategist-skill/internal/domain"

func (s *Store) setInstanceState(id, state string) bool {
	for i, instance := range s.Inventory.Instances {
		if instance.ID == id {
			instance.State = state
			s.Inventory.Instances[i] = instance
			return true
		}
	}
	return false
}

func (s *Store) markLastKnownGood(id string) {
	for i, instance := range s.Inventory.Instances {
		instance.LastKnownGood = instance.ID == id
		if instance.ID == id {
			instance.State = StateActive
		}
		s.Inventory.Instances[i] = instance
	}
}

func (s *Store) bindingIndex(slot string) (int, domain.SlotBinding, bool) {
	for i, binding := range s.Bindings {
		if binding.Slot == slot {
			return i, binding, true
		}
	}
	return -1, domain.SlotBinding{}, false
}

func (s *Store) lastKnownGoodInstanceID() string {
	for _, instance := range s.Inventory.Instances {
		if instance.LastKnownGood {
			return instance.ID
		}
	}
	return ""
}

func (t *Transaction) append(state, code string) {
	for _, entry := range t.Journal {
		if entry.State == state && entry.Code == code {
			return
		}
	}
	t.Journal = append(t.Journal, JournalEntry{State: state, Code: code})
}
