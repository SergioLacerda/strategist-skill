package domain

import (
	"reflect"
	"testing"
)

func fixtureLockForRoleInvocationPlan() PluginLockFile {
	return PluginLockFile{
		SchemaVersion: PluginLockFileSchemaVersion,
		Lock: PluginLock{
			Nodes: []PluginLockNode{
				{ID: "brainstorming", Kind: "adapter_contract", Digest: "sha256:weapon"},
				{ID: "ranger:brainstorming", Kind: "role_provider_binding", Digest: "sha256:binding"},
			},
		},
		Bindings: []SlotBinding{
			{Slot: "discovery", InstalledInstanceID: "brainstorming", Generation: 3, Status: "enabled"},
			{Slot: "refinement", InstalledInstanceID: "openspec-propose", Generation: 1, Status: "enabled"},
		},
	}
}

func TestNewRoleInvocationPlanFromLock_FieldForFieldEqualityWithSourceRecord(t *testing.T) {
	lock := fixtureLockForRoleInvocationPlan()

	plan, err := NewRoleInvocationPlanFromLock("ranger", "discovery", lock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := RoleInvocationPlan{
		Role:              "ranger",
		Slot:              "discovery",
		Mode:              SlotBindingModeCustom,
		WeaponID:          "brainstorming",
		WeaponDigest:      "sha256:weapon",
		BindingDigest:     "sha256:binding",
		BindingGeneration: 3,
		BindingStatus:     "enabled",
	}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("plan = %+v, want %+v", plan, want)
	}
}

func TestNewRoleInvocationPlanFromLock_NoBindingForSlot(t *testing.T) {
	lock := fixtureLockForRoleInvocationPlan()

	if _, err := NewRoleInvocationPlanFromLock("sniper", "execution", lock); err == nil {
		t.Fatal("expected an error for a slot with no persisted binding")
	}
}

func TestNewRoleInvocationPlanFromLock_MultipleBindingsForSlot(t *testing.T) {
	lock := fixtureLockForRoleInvocationPlan()
	lock.Bindings = append(lock.Bindings, SlotBinding{Slot: "discovery", InstalledInstanceID: "openspec-explore"})

	if _, err := NewRoleInvocationPlanFromLock("ranger", "discovery", lock); err == nil {
		t.Fatal("expected an error for a slot with more than one persisted binding")
	}
}

func TestNewRoleInvocationPlanFromLock_RankedModeIsRejected(t *testing.T) {
	lock := PluginLockFile{
		Bindings: []SlotBinding{{Slot: "discovery", InstalledInstanceID: "brainstorming", Mode: SlotBindingModeRanked}},
	}

	if _, err := NewRoleInvocationPlanFromLock("ranger", "discovery", lock); err == nil {
		t.Fatal("expected an error — NewRoleInvocationPlanFromLock only resolves Custom bindings")
	}
}

func TestNewRankedRoleInvocationPlanFromCatalog_Success(t *testing.T) {
	binding := SlotBinding{Slot: "discovery", InstalledInstanceID: "brainstorming", Generation: 1, Status: "active", Mode: SlotBindingModeRanked}
	stamp := CatalogRankedStamp{ID: "brainstorming", CanonicalRole: "ranger", Roles: []string{"ranger"}, Ranked: true, CertificationDigest: "sha256:cert"}

	plan, err := NewRankedRoleInvocationPlanFromCatalog("ranger", "discovery", binding, stamp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := RoleInvocationPlan{
		Role: "ranger", Slot: "discovery", Mode: SlotBindingModeRanked,
		WeaponID: "brainstorming", WeaponDigest: "sha256:cert", BindingDigest: "sha256:cert",
		BindingGeneration: 1, BindingStatus: "active",
	}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("plan = %+v, want %+v", plan, want)
	}
}

func TestNewRankedRoleInvocationPlanFromCatalog_RejectsUncertifiedStamp(t *testing.T) {
	binding := SlotBinding{Slot: "discovery", InstalledInstanceID: "brainstorming", Mode: SlotBindingModeRanked}
	stamp := CatalogRankedStamp{ID: "brainstorming", CanonicalRole: "ranger", Roles: []string{"ranger"}}

	if _, err := NewRankedRoleInvocationPlanFromCatalog("ranger", "discovery", binding, stamp); err == nil {
		t.Fatal("expected an error for an uncertified stamp")
	}
}

func TestNewRankedRoleInvocationPlanFromCatalog_RejectsProviderMismatch(t *testing.T) {
	binding := SlotBinding{Slot: "discovery", InstalledInstanceID: "brainstorming", Mode: SlotBindingModeRanked}
	stamp := CatalogRankedStamp{ID: "other", Ranked: true, CertificationDigest: "sha256:cert", Roles: []string{"ranger"}}

	if _, err := NewRankedRoleInvocationPlanFromCatalog("ranger", "discovery", binding, stamp); err == nil {
		t.Fatal("expected an error for a provider mismatch")
	}
}

func TestNewRankedRoleInvocationPlanFromCatalog_RejectsMissingRoleAffinity(t *testing.T) {
	binding := SlotBinding{Slot: "discovery", InstalledInstanceID: "brainstorming", Mode: SlotBindingModeRanked}
	stamp := CatalogRankedStamp{ID: "brainstorming", Ranked: true, CertificationDigest: "sha256:cert", Roles: []string{"archivist"}}

	if _, err := NewRankedRoleInvocationPlanFromCatalog("ranger", "discovery", binding, stamp); err == nil {
		t.Fatal("expected an error for missing role affinity")
	}
}

func TestNewRoleInvocationPlanFromLock_InvalidModeIsRejected(t *testing.T) {
	lock := PluginLockFile{
		Bindings: []SlotBinding{{Slot: "discovery", InstalledInstanceID: "brainstorming", Mode: "typo"}},
	}

	if _, err := NewRoleInvocationPlanFromLock("ranger", "discovery", lock); err == nil {
		t.Fatal("expected an error for an unrecognized binding mode")
	}
}

func TestSlotBinding_EffectiveModeDefaultsToCustom(t *testing.T) {
	b := SlotBinding{}
	if got := b.EffectiveMode(); got != SlotBindingModeCustom {
		t.Fatalf("EffectiveMode() = %q, want %q", got, SlotBindingModeCustom)
	}
}

func TestSlotBinding_ValidMode(t *testing.T) {
	cases := []struct {
		mode string
		want bool
	}{
		{"", true},
		{SlotBindingModeCustom, true},
		{SlotBindingModeRanked, true},
		{"typo", false},
	}
	for _, c := range cases {
		b := SlotBinding{Mode: c.mode}
		if got := b.ValidMode(); got != c.want {
			t.Errorf("SlotBinding{Mode: %q}.ValidMode() = %v, want %v", c.mode, got, c.want)
		}
	}
}

func TestNewRoleInvocationPlanFromLock_MissingDigestsAreEmptyNotFabricated(t *testing.T) {
	lock := PluginLockFile{
		Bindings: []SlotBinding{{Slot: "discovery", InstalledInstanceID: "brainstorming"}},
	}

	plan, err := NewRoleInvocationPlanFromLock("ranger", "discovery", lock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.WeaponDigest != "" || plan.BindingDigest != "" {
		t.Fatalf("expected empty digests when lock has no matching nodes, got weapon=%q binding=%q", plan.WeaponDigest, plan.BindingDigest)
	}
}
