package installplan_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	domaininstallplan "github.com/SergioLacerda/strategist-skill/internal/domain/installplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallPlanIsDeterministicAndSealed(t *testing.T) {
	t.Parallel()

	slots := map[string]string{"discovery": "brainstorming", "execution": "sniper"}
	modes := map[string]string{"discovery": domain.SlotBindingModeRanked}
	bindings := []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "brainstorming@1.0.0", Status: "enabled"}}
	lock := domain.PluginLock{SchemaVersion: "lock/v1", GraphDigest: "sha256:graph", Nodes: []domain.PluginLockNode{{ID: "brainstorming", Kind: "adapter_contract", Digest: "sha256:adapter"}}}

	plan, err := domaininstallplan.NewInstallPlan(domain.StageRoster, "wizard", "/workspace", slots, modes, lock, bindings)
	require.NoError(t, err)
	require.NoError(t, plan.Validate())

	slots["discovery"] = "changed"
	modes["discovery"] = domain.SlotBindingModeCustom
	lock.Nodes[0].Digest = "tampered"
	bindings[0].Status = "tampered"

	assert.Equal(t, "brainstorming", plan.Slots["discovery"])
	assert.Equal(t, domain.SlotBindingModeRanked, plan.SlotModes["discovery"])
	assert.Equal(t, "sha256:adapter", plan.Lock.Nodes[0].Digest)
	assert.Equal(t, "enabled", plan.Bindings[0].Status)
}

func TestInstallPlanRejectsStaleDigest(t *testing.T) {
	t.Parallel()

	plan, err := domaininstallplan.NewInstallPlan(
		domain.StageRoster,
		"silent",
		"/workspace",
		map[string]string{"discovery": "brainstorming"},
		nil,
		domain.PluginLock{},
		[]domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "brainstorming@1.0.0"}},
	)
	require.NoError(t, err)
	plan.Slots["discovery"] = "other"

	require.ErrorContains(t, plan.Validate(), "plan_digest mismatch")
}

func TestInstallPlanRejectsBindingSelectionDrift(t *testing.T) {
	t.Parallel()

	base := func(bindings []domain.SlotBinding) error {
		_, err := domaininstallplan.NewInstallPlan(
			domain.StageRoster, "silent", "/workspace",
			map[string]string{"discovery": "brainstorming"},
			map[string]string{"discovery": domain.SlotBindingModeCustom},
			domain.PluginLock{}, bindings,
		)
		return err
	}

	require.ErrorContains(t, base([]domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: "weapon@1.0.0"}}), "no matching slot")
	require.ErrorContains(t, base([]domain.SlotBinding{{Slot: "discovery"}, {Slot: "discovery", InstalledInstanceID: "weapon@1.0.0"}}), "no installed Weapon")
	require.ErrorContains(t, base([]domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "a"}, {Slot: "discovery", InstalledInstanceID: "b"}}), "duplicate binding")
}

func TestInstallPlanRejectsUnknownSlotMode(t *testing.T) {
	t.Parallel()

	_, err := domaininstallplan.NewInstallPlan(
		domain.StageRoster, "silent", "/workspace",
		map[string]string{"discovery": "brainstorming"},
		map[string]string{"discovery": "latest"},
		domain.PluginLock{},
		[]domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "weapon@1.0.0"}},
	)
	require.ErrorContains(t, err, "unsupported mode")
}

func TestInstallPlanValidationErrorCoverage(t *testing.T) {
	validSlots := map[string]string{"discovery": "brainstorming"}
	validModes := map[string]string{"discovery": domain.SlotBindingModeRanked}
	validBindings := []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "brainstorming"}}
	newPlan := func() domaininstallplan.InstallPlan {
		plan, err := domaininstallplan.NewInstallPlan(domain.StageRoster, "silent", "/workspace", validSlots, validModes, domain.PluginLock{}, validBindings)
		require.NoError(t, err)
		return plan
	}

	for name, build := range map[string]func() (domaininstallplan.InstallPlan, error){
		"invalid stage": func() (domaininstallplan.InstallPlan, error) {
			return domaininstallplan.NewInstallPlan(domain.StageFull, "silent", "/workspace", validSlots, validModes, domain.PluginLock{}, validBindings)
		},
		"missing slots": func() (domaininstallplan.InstallPlan, error) {
			return domaininstallplan.NewInstallPlan(domain.StageRoster, "silent", "/workspace", nil, validModes, domain.PluginLock{}, validBindings)
		},
		"missing bindings": func() (domaininstallplan.InstallPlan, error) {
			return domaininstallplan.NewInstallPlan(domain.StageRoster, "silent", "/workspace", validSlots, validModes, domain.PluginLock{}, nil)
		},
		"empty slot": func() (domaininstallplan.InstallPlan, error) {
			return domaininstallplan.NewInstallPlan(domain.StageRoster, "silent", "/workspace", map[string]string{"": "brainstorming"}, nil, domain.PluginLock{}, validBindings)
		},
		"orphan mode": func() (domaininstallplan.InstallPlan, error) {
			return domaininstallplan.NewInstallPlan(domain.StageRoster, "silent", "/workspace", validSlots, map[string]string{"execution": domain.SlotBindingModeRanked}, domain.PluginLock{}, validBindings)
		},
		"invalid mode": func() (domaininstallplan.InstallPlan, error) {
			return domaininstallplan.NewInstallPlan(domain.StageRoster, "silent", "/workspace", validSlots, map[string]string{"discovery": "future"}, domain.PluginLock{}, validBindings)
		},
		"orphan binding": func() (domaininstallplan.InstallPlan, error) {
			return domaininstallplan.NewInstallPlan(domain.StageRoster, "silent", "/workspace", validSlots, nil, domain.PluginLock{}, []domain.SlotBinding{{Slot: "execution", InstalledInstanceID: "sniper"}})
		},
		"empty binding": func() (domaininstallplan.InstallPlan, error) {
			return domaininstallplan.NewInstallPlan(domain.StageRoster, "silent", "/workspace", validSlots, nil, domain.PluginLock{}, []domain.SlotBinding{{Slot: "discovery"}})
		},
		"duplicate binding": func() (domaininstallplan.InstallPlan, error) {
			return domaininstallplan.NewInstallPlan(domain.StageRoster, "silent", "/workspace", validSlots, nil, domain.PluginLock{}, append(validBindings, validBindings[0]))
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := build()
			require.Error(t, err)
		})
	}

	plan := newPlan()
	plan.PlanDigest = ""
	require.ErrorContains(t, plan.Validate(), "plan_digest is required")
	plan = newPlan()
	plan.PlanDigest = "sha256:tampered"
	require.ErrorContains(t, plan.Validate(), "plan_digest mismatch")
}
