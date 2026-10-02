package install

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// scaffoldRisk maps a slot to the risk_score its contract requires.
var scaffoldRisk = map[string]string{
	string(domain.SlotDiscovery):  "write_analysis",
	string(domain.SlotRefinement): "write_analysis",
	string(domain.SlotExecution):  "controlled",
}

func validateScaffoldDeclaration(roles, slots []string) (string, error) {
	if len(roles) == 0 {
		return "", scaffoldError("role_unknown", "at least one --role is required")
	}
	if len(slots) == 0 {
		return "", scaffoldError("slot_unknown", "at least one --slot is required")
	}
	if err := validateScaffoldSlots(slots); err != nil {
		return "", err
	}
	if err := validateScaffoldRoles(roles, slots); err != nil {
		return "", err
	}
	return scaffoldRiskFor(slots)
}

func validateScaffoldSlots(slots []string) error {
	for _, slot := range slots {
		if !domain.IsValidSlot(slot) {
			return scaffoldError("slot_unknown", "slot %q is not a Strategist slot", slot)
		}
	}
	return nil
}

func validateScaffoldRoles(roles, slots []string) error {
	owned := make(map[string]bool, len(slots))
	for _, id := range roles {
		slot, err := scaffoldRoleSlot(id, slots)
		if err != nil {
			return err
		}
		owned[slot] = true
	}
	for _, slot := range slots {
		if !owned[slot] {
			return scaffoldError("role_slot_mismatch", "slot %q has no declared role that owns it", slot)
		}
	}
	return nil
}

// scaffoldRoleSlot returns the slot role id owns, which must be among slots.
func scaffoldRoleSlot(id string, slots []string) (string, error) {
	role, ok := domain.DefaultRoleRegistry().Get(id)
	if !ok {
		return "", scaffoldError("role_unknown", "role %q is not in the Role registry", id)
	}
	if !containsString(slots, role.Slot) {
		return "", scaffoldError("role_slot_mismatch", "role %q owns slot %q, which is not among %v", id, role.Slot, slots)
	}
	return role.Slot, nil
}

func scaffoldRiskFor(slots []string) (string, error) {
	risk := scaffoldRisk[slots[0]]
	for _, slot := range slots[1:] {
		if scaffoldRisk[slot] != risk {
			return "", scaffoldError("role_slot_mismatch", "slots %v require different risk_score contracts; declare one weapon per risk class", slots)
		}
	}
	return risk, nil
}
