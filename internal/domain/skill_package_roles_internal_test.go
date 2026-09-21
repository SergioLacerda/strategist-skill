package domain

import "testing"

// Provider role affinity is limited to slot-bound roles plus "auxiliary". Adding
// Scout to the registry must not make it a valid provider affinity.
func TestProviderRoleAffinityStaysLimitedToSlotBoundRoles(t *testing.T) {
	for _, role := range []string{"ranger", "archivist", "sniper", "auxiliary"} {
		if invalidRole([]string{role}) {
			t.Errorf("%q must remain a valid provider role affinity", role)
		}
	}
	for _, role := range []string{"scout", "gate", "Ranger", ""} {
		if !invalidRole([]string{role}) {
			t.Errorf("%q must not be a valid provider role affinity", role)
		}
	}
}

func TestContradictoryAffinityUsesTheRegistrySlot(t *testing.T) {
	if contradictoryAffinity([]string{"ranger"}, []string{"discovery"}) {
		t.Error("ranger with the discovery slot is consistent")
	}
	if !contradictoryAffinity([]string{"ranger"}, []string{"execution"}) {
		t.Error("ranger with only the execution slot is contradictory")
	}
	if contradictoryAffinity([]string{"auxiliary"}, []string{"auxiliary"}) {
		t.Error("auxiliary with the auxiliary slot is consistent")
	}
}
