package install

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// slotRoleID returns the id of the role that fills a slot, read from the role
// registry instead of a hardcoded slot-to-role table.
func slotRoleID(slot domain.SlotName) string {
	role, _ := domain.DefaultRoleRegistry().RoleForSlot(string(slot))
	return role.ID
}

// slotHandoffSchema returns the handoff schema of the role that fills a slot;
// empty for the terminal role.
func slotHandoffSchema(slot domain.SlotName) string {
	role, _ := domain.DefaultRoleRegistry().RoleForSlot(string(slot))
	return role.HandoffSchema
}
