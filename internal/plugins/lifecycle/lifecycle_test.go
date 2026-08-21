package lifecycle_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	instanceActive    = "inst-active"
	instanceCandidate = "inst-candidate"
)

func TestLifecycleStagesProbesAndActivatesWithCASGeneration(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateVerified},
	}
	store.Bindings = []domain.SlotBinding{{
		SchemaVersion:       "strategist-plugin-binding/v1",
		Slot:                "refinement",
		InstalledInstanceID: instanceActive,
		Generation:          7,
		Status:              "enabled",
	}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))
	require.NoError(t, store.Probe(tx.ID, true))
	require.NoError(t, store.Activate(tx.ID, 7))

	binding, ok := store.Binding("refinement")
	require.True(t, ok)
	assert.Equal(t, instanceCandidate, binding.InstalledInstanceID)
	assert.Equal(t, int64(8), binding.Generation)

	active, _ := store.Instance(instanceCandidate)
	old, _ := store.Instance(instanceActive)
	assert.Equal(t, lifecycle.StateActive, active.State)
	assert.True(t, active.LastKnownGood)
	assert.False(t, old.LastKnownGood)
	assert.Equal(t, lifecycle.StateComplete, store.Transaction(tx.ID).State)
	assert.Equal(t, []string{
		lifecycle.StateResolved,
		lifecycle.StateStaged,
		lifecycle.StateProbed,
		lifecycle.StateActive,
		lifecycle.StateComplete,
	}, store.Transaction(tx.ID).JournalStates())
}

func TestLifecycleRejectsActivateBeforeProbe(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateVerified}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))

	err = store.Activate(tx.ID, 1)
	require.Error(t, err)
	require.ErrorContains(t, err, "activation_requires_successful_probe")
}

func TestLifecycleRejectsCASGenerationConflict(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceCandidate, State: lifecycle.StateVerified}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 3, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))
	require.NoError(t, store.Probe(tx.ID, true))

	err = store.Activate(tx.ID, 2)
	require.Error(t, err)
	require.ErrorContains(t, err, "binding_generation_conflict")
}

func TestLifecycleRollsBackFailedProbeToLastKnownGood(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateVerified},
	}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 5, Status: "enabled"}}

	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))
	require.NoError(t, store.Probe(tx.ID, false))

	err = store.Activate(tx.ID, 5)
	require.Error(t, err)
	require.ErrorContains(t, err, "activation_requires_successful_probe")

	require.NoError(t, store.Rollback(tx.ID))
	binding, _ := store.Binding("refinement")
	candidate, _ := store.Instance(instanceCandidate)
	assert.Equal(t, instanceActive, binding.InstalledInstanceID)
	assert.Equal(t, lifecycle.StateFailed, candidate.State)
	assert.Equal(t, lifecycle.StateRolledBack, store.Transaction(tx.ID).State)
}

func TestLifecycleRecoveryRollsBackIncompleteTransaction(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateStaged},
	}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 4, Status: "enabled"}}
	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))

	require.NoError(t, store.Recover())

	binding, _ := store.Binding("refinement")
	candidate, _ := store.Instance(instanceCandidate)
	assert.Equal(t, instanceActive, binding.InstalledInstanceID)
	assert.Equal(t, lifecycle.StateFailed, candidate.State)
	assert.Equal(t, lifecycle.StateRolledBack, store.Transaction(tx.ID).State)
}

func TestLifecycleRollbackUsesLastKnownGoodWhenRecordedTargetMissing(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateStaged},
	}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: "missing-old", Generation: 2, Status: "enabled"}}
	tx, err := store.Begin("tx-1", "refinement", instanceCandidate)
	require.NoError(t, err)
	require.NoError(t, store.Stage(tx.ID))

	require.NoError(t, store.Rollback(tx.ID))

	binding, _ := store.Binding("refinement")
	assert.Equal(t, instanceActive, binding.InstalledInstanceID)
}

func TestLifecycleUninstallBlockedWhenBoundOrReferenced(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true},
		{ID: instanceCandidate, State: lifecycle.StateResolved},
	}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 1, Status: "enabled"}}
	store.Dependents[instanceCandidate] = []string{"dep-1"}

	err := store.Uninstall(instanceActive)
	require.Error(t, err)
	require.ErrorContains(t, err, "uninstall_blocked_bound_instance")

	err = store.Uninstall(instanceCandidate)
	require.Error(t, err)
	require.ErrorContains(t, err, "uninstall_blocked_required_dependent")
}

func TestLifecycleQuarantineDisablesBoundInstance(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 3, Status: "enabled"}}

	require.NoError(t, store.Quarantine(instanceActive, "malicious_adapter_fixture"))

	instance, _ := store.Instance(instanceActive)
	binding, _ := store.Binding("refinement")
	assert.Equal(t, lifecycle.StateQuarantined, instance.State)
	assert.False(t, instance.LastKnownGood)
	assert.Equal(t, "quarantined", binding.Status)
	assert.Equal(t, int64(4), binding.Generation)

	err := store.Uninstall(instanceActive)
	require.Error(t, err)
	require.ErrorContains(t, err, "uninstall_blocked_bound_instance")
}

func TestLifecycleDeprecateMarksBoundInstanceWithoutDeletingIt(t *testing.T) {
	t.Parallel()

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: instanceActive, State: lifecycle.StateActive, LastKnownGood: true}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: instanceActive, Generation: 9, Status: "enabled"}}

	require.NoError(t, store.Deprecate(instanceActive, "upstream_eol"))

	instance, _ := store.Instance(instanceActive)
	binding, _ := store.Binding("refinement")
	assert.Equal(t, lifecycle.StateDeprecated, instance.State)
	assert.False(t, instance.LastKnownGood)
	assert.Equal(t, "deprecated", binding.Status)
	assert.Equal(t, int64(10), binding.Generation)
	assert.Equal(t, "upstream_eol", instance.VerificationEvidence)
}
