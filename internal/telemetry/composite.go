package telemetry

import "strings"

// Composite Weapon telemetry identifiers.
const (
	CompositeWeaponEventName    = "strategist.weapon.component_invocation"
	CompositeWeaponContractID   = "weapon-composition/v1"
	AttrWeaponComponentRequired = "strategist.weapon.component_required"
	AttrWeaponComponentStatus   = "strategist.weapon.component_status"
	AttrWeaponAggregateDegraded = "strategist.weapon.aggregate_degraded"
)

// NewCompositeWeaponEvent records only bounded identity and evidence
// references. Component artifacts and raw skill payloads never enter
// telemetry.
func NewCompositeWeaponEvent(runID, role, slot, weaponID, componentID, parentInvocationID, status, reason, evidence string, required, degraded bool) Event {
	failed := status != "ready" || strings.TrimSpace(reason) != ""
	severity := SeverityInfo
	if failed {
		severity = SeverityError
	}
	event := NewEvent(CompositeWeaponEventName, severity, runID, true)
	event.Attributes = map[string]any{
		AttrEventContractID:         CompositeWeaponContractID,
		AttrEventAuthority:          AuthorityStrategistLocal,
		AttrRole:                    role,
		AttrPhase:                   slot,
		AttrWeapon:                  weaponID,
		AttrWeaponComponent:         componentID,
		AttrWeaponParentInvocation:  parentInvocationID,
		AttrWeaponComponentRequired: required,
		AttrWeaponComponentStatus:   status,
		AttrWeaponAggregateDegraded: degraded,
		AttrStatus:                  map[bool]string{true: "blocked", false: "done"}[failed],
	}
	if reason != "" {
		event.Attributes[AttrReason] = reason
	}
	if evidence != "" {
		event.Attributes[AttrWeaponInvocationEvidence] = evidence
	}
	return event
}
