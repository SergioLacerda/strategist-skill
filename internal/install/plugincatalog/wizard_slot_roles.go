package plugincatalog

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// SlotRoleID returns the id of the role that fills a slot, read from the role
// registry instead of a hardcoded slot-to-role table.
func SlotRoleID(slot domain.SlotName) string {
	role, _ := domain.DefaultRoleRegistry().RoleForSlot(string(slot))
	return role.ID
}

// SlotHandoffSchema returns the handoff schema of the role that fills a slot;
// empty for the terminal role.
func SlotHandoffSchema(slot domain.SlotName) string {
	role, _ := domain.DefaultRoleRegistry().RoleForSlot(string(slot))
	return role.HandoffSchema
}
